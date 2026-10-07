package artifact

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/controller"
	"github.com/erikeah/clavel/internal/core"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

const evaluationResult = `{"storePath":"/nix/store/abc123-nixos-system-server"}`

type fakeArtifactClient struct {
	corev1connect.ArtifactServiceClient
	objects map[string]*corev1.Artifact
	shows   int
	updates int
	deletes int
	masks   [][]string
}

func newFakeArtifactClient(artifacts ...*core.Artifact) *fakeArtifactClient {
	client := &fakeArtifactClient{objects: map[string]*corev1.Artifact{}}
	for _, artifact := range artifacts {
		proto := &corev1.Artifact{}
		proto.Set(artifact)
		client.objects[artifact.Metadata.Name] = proto
	}
	return client
}

func (f *fakeArtifactClient) Show(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceShowRequest],
) (*connect.Response[corev1.ArtifactServiceShowResponse], error) {
	f.shows++
	artifact, ok := f.objects[request.Msg.GetName()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("artifact not found"))
	}
	return connect.NewResponse(&corev1.ArtifactServiceShowResponse{Data: artifact}), nil
}

func (f *fakeArtifactClient) Update(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceUpdateRequest],
) (*connect.Response[corev1.ArtifactServiceUpdateResponse], error) {
	f.updates++
	f.masks = append(f.masks, request.Msg.GetUpdateMask().GetPaths())
	f.objects[request.Msg.GetName()] = request.Msg.GetData()
	return connect.NewResponse(&corev1.ArtifactServiceUpdateResponse{}), nil
}

func (f *fakeArtifactClient) Delete(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceDeleteRequest],
) (*connect.Response[corev1.ArtifactServiceDeleteResponse], error) {
	f.deletes++
	delete(f.objects, request.Msg.GetName())
	return connect.NewResponse(&corev1.ArtifactServiceDeleteResponse{}), nil
}

type fakeEvaluationClient struct {
	corev1connect.EvaluationServiceClient
	objects map[string]*corev1.Evaluation
	shows   int
}

func newFakeEvaluationClient(evaluations ...*core.Evaluation) *fakeEvaluationClient {
	client := &fakeEvaluationClient{objects: map[string]*corev1.Evaluation{}}
	for _, evaluation := range evaluations {
		proto := &corev1.Evaluation{}
		proto.Set(evaluation)
		client.objects[evaluation.Metadata.Name] = proto
	}
	return client
}

func (f *fakeEvaluationClient) Show(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceShowRequest],
) (*connect.Response[corev1.EvaluationServiceShowResponse], error) {
	f.shows++
	evaluation, ok := f.objects[request.Msg.GetName()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("evaluation not found"))
	}
	return connect.NewResponse(&corev1.EvaluationServiceShowResponse{Data: evaluation}), nil
}

// succeededEvaluation is the upstream an artifact is allowed to resolve.
func succeededEvaluation(name string) *core.Evaluation {
	return &core.Evaluation{
		Spec:     core.EvaluationSpecification{Reference: "ref#" + name},
		Metadata: core.Metadata{Name: name},
		Status: core.EvaluationStatus{
			Phase:  core.EvaluationPhaseSucceeded,
			Result: core.EncodeResult([]byte(evaluationResult)),
		},
	}
}

func newTestInformer(artifacts ...*core.Artifact) *controller.Informer[*core.Artifact] {
	queue := controller.NewWorkQueue()
	informer := controller.NewInformer[*core.Artifact](queue, func(artifact *core.Artifact) string {
		return artifact.Metadata.Name
	}, controller.WithOnChange(trackDependency(controller.NewDependencyTracker())))
	for _, artifact := range artifacts {
		informer.Add(artifact)
	}
	return informer
}

// pendingArtifact is a freshly created artifact pointing at evalRef.
func pendingArtifact(name, evalRef string) *core.Artifact {
	return &core.Artifact{
		APIVersion: core.ArtifactAPIVersion,
		Kind:       core.ArtifactKind,
		Spec:       core.ArtifactSpecification{StorePath: core.ArtifactStorePath{EvalRef: evalRef}},
		Metadata:   core.Metadata{Name: name, Generation: 1},
	}
}

// storeProbe records the queries made against a store.
type storeProbe struct {
	present bool
	err     error
	calls   []string
}

func (p *storeProbe) pathInfo(_ context.Context, store string, storePath string) (bool, error) {
	p.calls = append(p.calls, store+"|"+storePath)
	return p.present, p.err
}

func statusOf(client *fakeArtifactClient, name string) *corev1.ArtifactStatus {
	return client.objects[name].GetStatus()
}

func TestReconcileSucceedsWhenPathIsPresent(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	evaluations := newFakeEvaluationClient(succeededEvaluation("server-eval"))
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, evaluations, newTestInformer(artifact), probe.pathInfo)

	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
	status := statusOf(client, "server")
	if status.GetPhase() != corev1.ArtifactStatusPhase_ARTIFACT_STATUS_PHASE_SUCCEEDED {
		t.Fatalf("phase = %v, want SUCCEEDED", status.GetPhase())
	}
	if status.GetStorePath() != "/nix/store/abc123-nixos-system-server" {
		t.Fatalf("store_path = %q, want the resolved path", status.GetStorePath())
	}
	if status.GetObservedGeneration() != 1 {
		t.Fatalf("observed_generation = %d, want 1", status.GetObservedGeneration())
	}
	if status.GetMessage() != "" {
		t.Fatalf("message = %q, want empty", status.GetMessage())
	}
	// Only the terminal write: unlike an evaluation, resolving a reference is a
	// read, so there is no in-progress status worth writing first.
	if client.updates != 1 {
		t.Fatalf("updates = %d, want 1 (succeeded)", client.updates)
	}
	if len(probe.calls) != 1 || probe.calls[0] != "|/nix/store/abc123-nixos-system-server" {
		t.Fatalf("store probes = %v, want one with an empty (local) store", probe.calls)
	}
}

func TestReconcilePassesConfiguredStoreToQuery(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	artifact.Spec.Store = "https://cache.nixos.org"
	client := newFakeArtifactClient(artifact)
	evaluations := newFakeEvaluationClient(succeededEvaluation("server-eval"))
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, evaluations, newTestInformer(artifact), probe.pathInfo)

	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
	if probe.calls[0] != "https://cache.nixos.org|/nix/store/abc123-nixos-system-server" {
		t.Fatalf("store probe = %q, want the configured store", probe.calls[0])
	}
}

func TestReconcileBlocksWhenEvaluationIsMissing(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, newFakeEvaluationClient(), newTestInformer(artifact), probe.pathInfo)

	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v, want nil while blocked", err)
	}
	status := statusOf(client, "server")
	if status.GetPhase() != corev1.ArtifactStatusPhase_ARTIFACT_STATUS_PHASE_PENDING {
		t.Fatalf("phase = %v, want PENDING", status.GetPhase())
	}
	if !strings.Contains(status.GetMessage(), "server-eval") {
		t.Fatalf("message = %q, want the evaluation named", status.GetMessage())
	}
	if len(probe.calls) != 0 {
		t.Fatalf("store probes = %v, want none before the upstream is ready", probe.calls)
	}
}

func TestReconcileBlocksWhileEvaluationIsNotSucceeded(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	pending := &core.Evaluation{
		Spec:     core.EvaluationSpecification{Reference: "ref#server-eval"},
		Metadata: core.Metadata{Name: "server-eval"},
		Status:   core.EvaluationStatus{Phase: core.EvaluationPhasePending},
	}
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, newFakeEvaluationClient(pending), newTestInformer(artifact), probe.pathInfo)

	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v, want nil while blocked", err)
	}
	if statusOf(client, "server").GetPhase() != corev1.ArtifactStatusPhase_ARTIFACT_STATUS_PHASE_PENDING {
		t.Fatal("phase = want PENDING")
	}
	if len(probe.calls) != 0 {
		t.Fatalf("store probes = %v, want none before the upstream is ready", probe.calls)
	}
}

func TestReconcileBlockedWritesPendingOnlyOncePerGeneration(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	informer := newTestInformer(artifact)
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, newFakeEvaluationClient(), informer, probe.pathInfo)

	for attempt := 0; attempt < 5; attempt++ {
		if err := reconcile(context.Background(), "server"); err != nil {
			t.Fatalf("reconcile() error = %v on attempt %d", err, attempt)
		}
		// Simulate the watch event echoing the status write back into the cache.
		informer.Add(client.objects["server"].Convert(nil))
	}
	if client.updates != 1 {
		t.Fatalf("updates = %d, want 1 pending write for the generation", client.updates)
	}
}

func TestReconcileFailsWithoutRetryWhenPathIsAbsent(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	evaluations := newFakeEvaluationClient(succeededEvaluation("server-eval"))
	probe := &storeProbe{present: false}
	reconcile := newReconcile(client, evaluations, newTestInformer(artifact), probe.pathInfo)

	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v, want nil: an absent path is not retryable", err)
	}
	status := statusOf(client, "server")
	if status.GetPhase() != corev1.ArtifactStatusPhase_ARTIFACT_STATUS_PHASE_FAILED {
		t.Fatalf("phase = %v, want FAILED", status.GetPhase())
	}
	if !strings.Contains(status.GetMessage(), "not present") {
		t.Fatalf("message = %q, want the absence reported", status.GetMessage())
	}
	if !strings.Contains(status.GetMessage(), "local store") {
		t.Fatalf("message = %q, want the empty store named as local", status.GetMessage())
	}
}

func TestReconcileRetriesWhenTheStoreCannotBeQueried(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	evaluations := newFakeEvaluationClient(succeededEvaluation("server-eval"))
	queryErr := errors.New("store unreachable")
	probe := &storeProbe{err: queryErr}
	reconcile := newReconcile(client, evaluations, newTestInformer(artifact), probe.pathInfo)

	err := reconcile(context.Background(), "server")
	if err == nil {
		t.Fatal("reconcile() error = nil, want the query failure")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error = %v, want the query failure joined in", err)
	}
	status := statusOf(client, "server")
	if status.GetPhase() != corev1.ArtifactStatusPhase_ARTIFACT_STATUS_PHASE_FAILED {
		t.Fatalf("phase = %v, want FAILED", status.GetPhase())
	}
	if status.GetStorePath() != "/nix/store/abc123-nixos-system-server" {
		t.Fatalf("store_path = %q, want the path that was being resolved", status.GetStorePath())
	}
}

func TestReconcileFailsOnResultWithoutStorePath(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	client := newFakeArtifactClient(artifact)
	broken := succeededEvaluation("server-eval")
	broken.Status.Result = core.EncodeResult([]byte(`{"noStorePath":true}`))
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, newFakeEvaluationClient(broken), newTestInformer(artifact), probe.pathInfo)

	err := reconcile(context.Background(), "server")
	if err == nil {
		t.Fatal("reconcile() error = nil, want a contract violation")
	}
	if statusOf(client, "server").GetPhase() != corev1.ArtifactStatusPhase_ARTIFACT_STATUS_PHASE_FAILED {
		t.Fatal("phase = want FAILED")
	}
	if len(probe.calls) != 0 {
		t.Fatalf("store probes = %v, want none for a malformed result", probe.calls)
	}
}

func TestReconcileSkipsAlreadySucceededGeneration(t *testing.T) {
	artifact := pendingArtifact("server", "server-eval")
	artifact.Status = core.ArtifactStatus{
		Phase:              core.ArtifactPhaseSucceeded,
		StorePath:          "/nix/store/kept",
		ObservedGeneration: 1,
	}
	client := newFakeArtifactClient(artifact)
	probe := &storeProbe{present: true}
	reconcile := newReconcile(client, newFakeEvaluationClient(), newTestInformer(artifact), probe.pathInfo)

	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
	if client.shows != 0 || client.updates != 0 {
		t.Fatalf("shows = %d updates = %d, want no API calls", client.shows, client.updates)
	}
	if len(probe.calls) != 0 {
		t.Fatalf("store probes = %v, want none", probe.calls)
	}
}

func TestReconcileMissingKeyReturnsNil(t *testing.T) {
	probe := &storeProbe{present: true}
	reconcile := newReconcile(newFakeArtifactClient(), newFakeEvaluationClient(), newTestInformer(), probe.pathInfo)
	if err := reconcile(context.Background(), "gone"); err != nil {
		t.Fatalf("reconcile() error = %v, want nil", err)
	}
}

// terminating builds an artifact that is on its way out.
func terminating(finalizers []string) *core.Artifact {
	now := time.Now()
	artifact := pendingArtifact("server", "server-eval")
	artifact.Metadata.Finalizers = finalizers
	artifact.Metadata.DeletionTimestamp = &now
	return artifact
}

func TestReconcileLeavesTerminatingArtifactToTheService(t *testing.T) {
	probe := &storeProbe{present: true}
	for name, finalizers := range map[string][]string{
		"own finalizer":     {"clavel.core/artifact"},
		"foreign finalizer": {"example.com/cleanup"},
		"no finalizer":      nil,
	} {
		artifact := terminating(finalizers)
		client := newFakeArtifactClient(artifact)
		reconcile := newReconcile(client, newFakeEvaluationClient(), newTestInformer(artifact), probe.pathInfo)

		if err := reconcile(context.Background(), "server"); err != nil {
			t.Fatalf("%s: reconcile() error = %v", name, err)
		}
		if client.deletes != 0 || client.updates != 0 {
			t.Fatalf("%s: deletes = %d updates = %d, want neither: an artifact owns no store content", name, client.deletes, client.updates)
		}
		if len(probe.calls) != 0 {
			t.Fatalf("%s: store probes = %v, want none", name, probe.calls)
		}
	}
}

func TestTrackDependencyRegistersEvalRefEdge(t *testing.T) {
	tracker := controller.NewDependencyTracker()
	handler := trackDependency(tracker)

	artifact := pendingArtifact("server", "server-eval")
	keys := handler("server", artifact, true)
	if len(keys) != 1 || keys[0] != "server" {
		t.Fatalf("keys = %v, want [server]: the artifact itself is what reconciles", keys)
	}
	if dependents := tracker.Dependents("server-eval"); len(dependents) != 1 || dependents[0] != "server" {
		t.Fatalf("dependents = %v, want [server]", dependents)
	}

	// Retargeting replaces the edge rather than adding a second one.
	artifact.Spec.StorePath.EvalRef = "other-eval"
	handler("server", artifact, true)
	if dependents := tracker.Dependents("server-eval"); len(dependents) != 0 {
		t.Fatalf("dependents = %v, want the old evaluation to stop waking it", dependents)
	}
	if dependents := tracker.Dependents("other-eval"); len(dependents) != 1 {
		t.Fatalf("dependents = %v, want [server]", dependents)
	}

	handler("server", nil, false)
	if dependents := tracker.Dependents("other-eval"); len(dependents) != 0 {
		t.Fatalf("dependents = %v, want none once the artifact is gone", dependents)
	}
}

func TestWatchApplierReplacesCacheAtSync(t *testing.T) {
	informer := newTestInformer(&core.Artifact{Metadata: core.Metadata{Name: "stale"}})
	applier := newWatchApplier(informer)

	applier.apply(&corev1.ArtifactServiceWatchResponse{
		Type: corev1.EventType_EVENT_TYPE_PUT,
		Name: "kept",
		Data: &corev1.Artifact{Metadata: &corev1.Metadata{Name: "kept"}},
	})
	if _, ok := informer.Get("kept"); ok {
		t.Fatal("snapshot item was applied before the sync marker")
	}
	applier.apply(&corev1.ArtifactServiceWatchResponse{Type: corev1.EventType_EVENT_TYPE_SYNC})

	if _, ok := informer.Get("stale"); ok {
		t.Fatal("resource missing from the snapshot survived the sync")
	}
	if _, ok := informer.Get("kept"); !ok {
		t.Fatal("snapshot item was not cached at the sync marker")
	}
}

func TestWatchApplierAppliesChangesAfterSync(t *testing.T) {
	informer := newTestInformer()
	applier := newWatchApplier(informer)
	applier.apply(&corev1.ArtifactServiceWatchResponse{Type: corev1.EventType_EVENT_TYPE_SYNC})

	applier.apply(&corev1.ArtifactServiceWatchResponse{
		Type: corev1.EventType_EVENT_TYPE_PUT,
		Name: "server",
		Data: &corev1.Artifact{Metadata: &corev1.Metadata{Name: "server"}},
	})
	if _, ok := informer.Get("server"); !ok {
		t.Fatal("put after the sync marker was not cached")
	}
	applier.apply(&corev1.ArtifactServiceWatchResponse{
		Type: corev1.EventType_EVENT_TYPE_DELETE,
		Name: "server",
	})
	if _, ok := informer.Get("server"); ok {
		t.Fatal("delete event did not evict the resource")
	}
}
