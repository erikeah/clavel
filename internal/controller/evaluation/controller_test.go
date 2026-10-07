package evaluation

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/controller"
	"github.com/erikeah/clavel/internal/core"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

type fakeClient struct {
	corev1connect.EvaluationServiceClient
	objects map[string]*corev1.Evaluation
	shows   int
	updates int
	deletes int
	masks   [][]string
}

func newFakeClient(evaluations ...*core.Evaluation) *fakeClient {
	client := &fakeClient{objects: map[string]*corev1.Evaluation{}}
	for _, evaluation := range evaluations {
		proto := &corev1.Evaluation{}
		proto.Set(evaluation)
		client.objects[evaluation.Metadata.Name] = proto
	}
	return client
}

func (f *fakeClient) Show(
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

func (f *fakeClient) Update(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceUpdateRequest],
) (*connect.Response[corev1.EvaluationServiceUpdateResponse], error) {
	f.updates++
	f.masks = append(f.masks, request.Msg.GetUpdateMask().GetPaths())
	f.objects[request.Msg.GetName()] = request.Msg.GetData()
	return connect.NewResponse(&corev1.EvaluationServiceUpdateResponse{}), nil
}

func (f *fakeClient) Delete(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceDeleteRequest],
) (*connect.Response[corev1.EvaluationServiceDeleteResponse], error) {
	f.deletes++
	delete(f.objects, request.Msg.GetName())
	return connect.NewResponse(&corev1.EvaluationServiceDeleteResponse{}), nil
}

func newTestInformer(evaluations ...*core.Evaluation) *controller.Informer[*core.Evaluation] {
	queue := controller.NewWorkQueue()
	informer := controller.NewInformer[*core.Evaluation](queue, func(evaluation *core.Evaluation) string {
		return evaluation.Metadata.Name
	})
	for _, evaluation := range evaluations {
		informer.Add(evaluation)
	}
	return informer
}

func TestReconcileSuccess(t *testing.T) {
	evaluation := &core.Evaluation{
		Spec:     core.EvaluationSpecification{Reference: "github:erikeah/clavel?dir=example#server"},
		Metadata: core.Metadata{Name: "server", Generation: 3},
	}
	rawResult := `{"storePath":"/nix/store/abc123-nixos-system-server"}`
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		if reference != evaluation.Spec.Reference {
			t.Fatalf("reference = %q, want %q", reference, evaluation.Spec.Reference)
		}
		return rawResult, nil
	})

	if err := reconcile(context.Background(), evaluation.Metadata.Name); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}

	status := client.objects[evaluation.Metadata.Name].GetStatus()
	if status.GetPhase() != corev1.EvaluationStatusPhase_EVALUATION_STATUS_PHASE_SUCCEEDED {
		t.Fatalf("phase = %v, want SUCCEEDED", status.GetPhase())
	}
	if status.GetResult() != core.EncodeResult([]byte(rawResult)) {
		t.Fatalf("result = %q, want base64 of raw JSON", status.GetResult())
	}
	if status.GetObservedGeneration() != evaluation.Metadata.Generation {
		t.Fatalf("observed_generation = %d, want %d", status.GetObservedGeneration(), evaluation.Metadata.Generation)
	}
	if status.GetMessage() != "" {
		t.Fatalf("message = %q, want empty", status.GetMessage())
	}
	// Pending write then succeeded write.
	if client.updates != 2 {
		t.Fatalf("updates = %d, want 2 (pending + succeeded)", client.updates)
	}
}

func TestReconcileSkipsAlreadySucceededGeneration(t *testing.T) {
	evaluation := &core.Evaluation{
		Spec: core.EvaluationSpecification{Reference: "ref#server"},
		Metadata: core.Metadata{
			Name:       "server",
			Generation: 4,
		},
		Status: core.EvaluationStatus{
			Phase:              core.EvaluationPhaseSucceeded,
			Result:             core.EncodeResult([]byte(`"/nix/store/kept"`)),
			ObservedGeneration: 4,
		},
	}
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)
	evaluated := false
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		evaluated = true
		return "/nix/store/new", nil
	})

	if err := reconcile(context.Background(), evaluation.Metadata.Name); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
	if evaluated {
		t.Fatal("evaluation ran despite already-succeeded generation")
	}
	if client.shows != 0 || client.updates != 0 {
		t.Fatalf("shows = %d updates = %d, want no API calls", client.shows, client.updates)
	}
}

func TestReconcileRetriesOnSpecChange(t *testing.T) {
	rawResult := `{"storePath":"/nix/store/re-evaluated"}`
	evaluation := &core.Evaluation{
		Spec:     core.EvaluationSpecification{Reference: "ref#server-v2"},
		Metadata: core.Metadata{Name: "server", Generation: 5},
		Status: core.EvaluationStatus{
			Phase:              core.EvaluationPhaseSucceeded,
			Result:             core.EncodeResult([]byte(`"/nix/store/old"`)),
			ObservedGeneration: 4,
		},
	}
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		return rawResult, nil
	})

	if err := reconcile(context.Background(), evaluation.Metadata.Name); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
	status := client.objects[evaluation.Metadata.Name].GetStatus()
	if status.GetObservedGeneration() != 5 {
		t.Fatalf("observed_generation = %d, want 5", status.GetObservedGeneration())
	}
	if status.GetResult() != core.EncodeResult([]byte(rawResult)) {
		t.Fatalf("result = %q, want base64 of raw JSON", status.GetResult())
	}
}

func TestReconcileFailure(t *testing.T) {
	evaluation := &core.Evaluation{
		Spec:     core.EvaluationSpecification{Reference: "ref#broken"},
		Metadata: core.Metadata{Name: "server", Generation: 7},
	}
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)
	evalErr := errors.New("nix eval failed: attribute not found")
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		return "", evalErr
	})

	err := reconcile(context.Background(), evaluation.Metadata.Name)
	if err == nil {
		t.Fatal("reconcile() error = nil, want evaluation error")
	}
	if !errors.Is(err, evalErr) {
		t.Fatalf("error = %v, want wrapped evaluation error", err)
	}

	status := client.objects[evaluation.Metadata.Name].GetStatus()
	if status.GetPhase() != corev1.EvaluationStatusPhase_EVALUATION_STATUS_PHASE_FAILED {
		t.Fatalf("phase = %v, want FAILED", status.GetPhase())
	}
	if !strings.Contains(status.GetMessage(), "attribute not found") {
		t.Fatalf("message = %q, want eval error", status.GetMessage())
	}
	if status.GetObservedGeneration() != evaluation.Metadata.Generation {
		t.Fatalf("observed_generation = %d, want %d", status.GetObservedGeneration(), evaluation.Metadata.Generation)
	}
}

func TestReconcileRepeatedFailuresWriteStatusAtMostTwicePerGeneration(t *testing.T) {
	evaluation := &core.Evaluation{
		Spec:     core.EvaluationSpecification{Reference: "ref#broken"},
		Metadata: core.Metadata{Name: "server", Generation: 7},
	}
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		return "", errors.New("nix eval failed: attribute not found")
	})

	for attempt := 1; attempt <= 5; attempt++ {
		if err := reconcile(context.Background(), evaluation.Metadata.Name); err == nil {
			t.Fatalf("reconcile() error = nil on attempt %d, want failure", attempt)
		}
		// Simulate the watch event echoing the status write back into the cache.
		informer.Add(client.objects[evaluation.Metadata.Name].Convert(nil))
	}
	if client.updates != 2 {
		t.Fatalf("updates = %d, want at most 2 per generation (pending + failed)", client.updates)
	}
}

func TestReconcileMissingKeyReturnsNil(t *testing.T) {
	client := newFakeClient()
	informer := newTestInformer()
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		t.Fatal("evaluation ran for missing key")
		return "", nil
	})
	if err := reconcile(context.Background(), "gone"); err != nil {
		t.Fatalf("reconcile() error = %v, want nil", err)
	}
}

// terminating builds an evaluation that is on its way out.
func terminating(finalizers []string) *core.Evaluation {
	now := time.Now()
	return &core.Evaluation{
		Spec: core.EvaluationSpecification{Reference: "ref#server"},
		Metadata: core.Metadata{
			Name:              "server",
			Generation:        1,
			Finalizers:        finalizers,
			DeletionTimestamp: &now,
		},
	}
}

func reconcileDeletesInsteadOfEvaluating(t *testing.T, client *fakeClient, informer *controller.Informer[*core.Evaluation]) {
	t.Helper()
	reconcile := newReconcile(client, informer, func(ctx context.Context, reference string) (string, error) {
		t.Fatal("evaluation ran for a resource being deleted")
		return "", nil
	})
	if err := reconcile(context.Background(), "server"); err != nil {
		t.Fatalf("reconcile() error = %v", err)
	}
}

func TestReconcileClearsOwnFinalizerInsteadOfDeleting(t *testing.T) {
	evaluation := terminating([]string{controllerFinalizer})
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)

	reconcileDeletesInsteadOfEvaluating(t, client, informer)

	if client.deletes != 0 {
		t.Fatalf("deletes = %d, want 0 while a finalizer still blocks", client.deletes)
	}
	if client.updates != 1 {
		t.Fatalf("updates = %d, want 1 to drop the finalizer", client.updates)
	}
	wantMask := []string{"data.metadata.finalizers"}
	if !slices.Equal(client.masks[0], wantMask) {
		t.Fatalf("update mask = %v, want %v", client.masks[0], wantMask)
	}
	if finalizers := client.objects["server"].GetMetadata().GetFinalizers(); len(finalizers) != 0 {
		t.Fatalf("finalizers = %v, want empty", finalizers)
	}
}

func TestReconcileKeepsForeignFinalizer(t *testing.T) {
	evaluation := terminating([]string{"example.com/cleanup"})
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)

	reconcileDeletesInsteadOfEvaluating(t, client, informer)

	if client.updates != 0 || client.deletes != 0 {
		t.Fatalf("updates = %d deletes = %d, want neither", client.updates, client.deletes)
	}
}

func TestReconcileDeletesOnceFinalizersAreDrained(t *testing.T) {
	evaluation := terminating(nil)
	client := newFakeClient(evaluation)
	informer := newTestInformer(evaluation)

	reconcileDeletesInsteadOfEvaluating(t, client, informer)

	if client.deletes != 1 {
		t.Fatalf("deletes = %d, want 1", client.deletes)
	}
	if client.shows != 0 || client.updates != 0 {
		t.Fatalf("shows = %d updates = %d, want no API traffic before the delete", client.shows, client.updates)
	}
}

func TestWatchApplierReplacesCacheAtSync(t *testing.T) {
	informer := newTestInformer(&core.Evaluation{Metadata: core.Metadata{Name: "stale"}})
	applier := newWatchApplier(informer)

	applier.apply(&corev1.EvaluationServiceWatchResponse{
		Type: corev1.EventType_EVENT_TYPE_PUT,
		Name: "kept",
		Data: &corev1.Evaluation{Metadata: &corev1.Metadata{Name: "kept"}},
	})
	if _, ok := informer.Get("kept"); ok {
		t.Fatal("snapshot item was applied before the sync marker")
	}
	applier.apply(&corev1.EvaluationServiceWatchResponse{Type: corev1.EventType_EVENT_TYPE_SYNC})

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
	applier.apply(&corev1.EvaluationServiceWatchResponse{Type: corev1.EventType_EVENT_TYPE_SYNC})

	applier.apply(&corev1.EvaluationServiceWatchResponse{
		Type: corev1.EventType_EVENT_TYPE_PUT,
		Name: "server",
		Data: &corev1.Evaluation{Metadata: &corev1.Metadata{Name: "server"}},
	})
	if _, ok := informer.Get("server"); !ok {
		t.Fatal("put after the sync marker was not cached")
	}
	applier.apply(&corev1.EvaluationServiceWatchResponse{
		Type: corev1.EventType_EVENT_TYPE_DELETE,
		Name: "server",
	})
	if _, ok := informer.Get("server"); ok {
		t.Fatal("delete event did not evict the resource")
	}
}
