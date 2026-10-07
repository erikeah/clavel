package artifact

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/controller"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/nix"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
	"golang.org/x/time/rate"
)

const (
	apiAddressEnv  = "CLAVEL_API_ADDR"
	defaultAPIAddr = "http://localhost:8080"
	workerCountEnv = "CLAVEL_CONTROLLER_WORKERS"
	defaultWorkers = 4
	watchReconnect = time.Second
)

// pathInfo reports whether a store path exists in a store. An empty store is
// the local store; a path the store does not have is not an error.
type pathInfo func(ctx context.Context, store string, storePath string) (bool, error)

func apiAddress() string {
	if address := os.Getenv(apiAddressEnv); address != "" {
		return address
	}
	return defaultAPIAddr
}

func workerCount() int {
	if raw := os.Getenv(workerCountEnv); raw != "" {
		if count, err := strconv.Atoi(raw); err == nil && count > 0 {
			return count
		}
	}
	return defaultWorkers
}

// startArtifactWatch pumps the watch stream into the informer, reconnecting on
// stream errors. The first request of every stream asks for a snapshot, which
// the applier turns into an informer Replace, so a reconnect also evicts the
// resources deleted while the stream was down.
func startArtifactWatch(ctx context.Context, client corev1connect.ArtifactServiceClient, informer *controller.Informer[*core.Artifact]) {
	watchRequest := &connect.Request[corev1.ArtifactServiceWatchRequest]{
		Msg: &corev1.ArtifactServiceWatchRequest{List: true},
	}
	for {
		watchResponse, err := client.Watch(ctx, watchRequest)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("artifact watch failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(watchReconnect):
			}
			continue
		}
		applier := newWatchApplier(informer)
		for watchResponse.Receive() {
			if streamError := watchResponse.Msg().GetError(); streamError != nil {
				slog.Error(streamError.GetMessage())
				continue
			}
			applier.apply(watchResponse.Msg())
		}
		if err := watchResponse.Err(); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("artifact watch stream ended", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchReconnect):
		}
	}
}

// watchApplier folds a watch stream into an informer. Because the request asked
// for a snapshot (List), resources arriving before the sync marker are the
// state at connect time and are applied as a whole, while everything after it
// is applied as it happens.
type watchApplier struct {
	informer *controller.Informer[*core.Artifact]
	snapshot []*core.Artifact
	syncing  bool
}

func newWatchApplier(informer *controller.Informer[*core.Artifact]) *watchApplier {
	return &watchApplier{informer: informer, syncing: true}
}

func (applier *watchApplier) apply(message *corev1.ArtifactServiceWatchResponse) {
	switch message.GetType() {
	case corev1.EventType_EVENT_TYPE_DELETE:
		applier.informer.Delete(message.GetName())
	case corev1.EventType_EVENT_TYPE_SYNC:
		applier.informer.Replace(applier.snapshot)
		applier.snapshot = nil
		applier.syncing = false
	default:
		data := message.GetData()
		if data == nil {
			return
		}
		if !applier.syncing {
			applier.informer.Add(data.Convert(nil))
			return
		}
		applier.snapshot = append(applier.snapshot, data.Convert(nil))
	}
}

// startEvaluationWatch is the dependency trigger, and the reason the readiness
// gate never polls: an evaluation event is turned straight into the keys
// recorded as depending on it and queued. Nothing is cached here, because this
// stream only exists to wake dependents and the artifact informer already holds
// the resources being reconciled.
func startEvaluationWatch(ctx context.Context, client corev1connect.EvaluationServiceClient, queue controller.WorkQueue, tracker *controller.DependencyTracker) {
	watchRequest := &connect.Request[corev1.EvaluationServiceWatchRequest]{
		Msg: &corev1.EvaluationServiceWatchRequest{List: true},
	}
	for {
		watchResponse, err := client.Watch(ctx, watchRequest)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("evaluation watch failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(watchReconnect):
			}
			continue
		}
		for watchResponse.Receive() {
			message := watchResponse.Msg()
			if streamError := message.GetError(); streamError != nil {
				slog.Error(streamError.GetMessage())
				continue
			}
			// Only a put or a delete carries an identity. The sync marker
			// ends the snapshot and names nothing, so there is no dependent
			// to wake for it; the snapshot's puts still wake theirs.
			name := message.GetName()
			if name == "" {
				continue
			}
			for _, dependent := range tracker.Dependents(name) {
				queue.Add(dependent)
			}
		}
		if err := watchResponse.Err(); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("evaluation watch stream ended", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchReconnect):
		}
	}
}

// trackDependency records an artifact's evalRef as a reverse edge, so a change
// to that Evaluation wakes this artifact rather than leaving it to poll. The
// edge belongs to the artifact and is replaced wholesale, so retargeting the
// evalRef stops the old Evaluation waking it.
func trackDependency(tracker *controller.DependencyTracker) func(key string, artifact *core.Artifact, present bool) []string {
	return func(key string, artifact *core.Artifact, present bool) []string {
		if !present {
			tracker.Untrack(key)
		} else {
			tracker.Track(key, []string{artifact.Spec.StorePath.EvalRef})
		}
		return []string{key}
	}
}

// setStatus writes the artifact status through read-modify-write: Show obtains
// the current resource version, the status is replaced, and Update persists the
// full object (no field mask). A no-op update succeeds silently.
func setStatus(
	ctx context.Context,
	client corev1connect.ArtifactServiceClient,
	name string,
	status core.ArtifactStatus,
) error {
	response, err := client.Show(ctx, connect.NewRequest(&corev1.ArtifactServiceShowRequest{Name: name}))
	if err != nil {
		return err
	}
	current := response.Msg.GetData()
	if current.GetStatus().Convert() == status {
		return nil
	}
	protoStatus := &corev1.ArtifactStatus{}
	protoStatus.Set(&status)
	current.Status = protoStatus
	_, err = client.Update(ctx, connect.NewRequest(&corev1.ArtifactServiceUpdateRequest{
		Name: name,
		Data: current,
	}))
	return err
}

// block parks the artifact on an upstream that is not ready yet. Pending is
// written only for a never-observed or stale generation, so retries of the same
// generation neither re-write status nor re-trigger watch events.
func block(ctx context.Context, client corev1connect.ArtifactServiceClient, key string, artifact *core.Artifact, message string) error {
	if artifact.Status.Phase == core.ArtifactPhasePending &&
		artifact.Status.ObservedGeneration == artifact.Metadata.Generation {
		return nil
	}
	return setStatus(ctx, client, key, core.ArtifactStatus{
		Phase:              core.ArtifactPhasePending,
		ObservedGeneration: artifact.Metadata.Generation,
		Message:            message,
	})
}

func failed(artifact *core.Artifact, storePath string, message string) core.ArtifactStatus {
	return core.ArtifactStatus{
		Phase:              core.ArtifactPhaseFailed,
		StorePath:          storePath,
		ObservedGeneration: artifact.Metadata.Generation,
		Message:            message,
	}
}

// storeName names the store a message should refer to. The empty spec is the
// local store, which is not spelled by a URI anywhere else.
func storeName(store string) string {
	if store == "" {
		return "the local store"
	}
	return store
}

func newReconcile(
	artifacts corev1connect.ArtifactServiceClient,
	evaluations corev1connect.EvaluationServiceClient,
	informer *controller.Informer[*core.Artifact],
	info pathInfo,
) controller.ReconcileFunc {
	return func(ctx context.Context, key string) error {
		artifact, ok := informer.Get(key)
		if !ok {
			// Resource deleted before reconcile. No finalizer is registered,
			// so there is no cleanup the service could not already have done.
			slog.Info("artifact deleted", "name", key)
			return nil
		}
		if artifact.Metadata.DeletionTimestamp != nil {
			// Terminating with none of our finalizers on it. An artifact
			// points at a path clavel does not own, so removal is nothing
			// more than dropping the definition.
			return nil
		}
		if artifact.Status.Phase == core.ArtifactPhaseSucceeded &&
			artifact.Status.ObservedGeneration == artifact.Metadata.Generation {
			return nil
		}
		generation := artifact.Metadata.Generation
		evalRef := artifact.Spec.StorePath.EvalRef

		evaluation, err := evaluations.Show(ctx, connect.NewRequest(&corev1.EvaluationServiceShowRequest{Name: evalRef}))
		if err != nil {
			if connect.CodeOf(err) == connect.CodeNotFound {
				return block(ctx, artifacts, key, artifact, fmt.Sprintf("waiting for evaluation %q", evalRef))
			}
			return errors.Join(err, setStatus(ctx, artifacts, key, failed(artifact, "", err.Error())))
		}
		status := evaluation.Msg.GetData().GetStatus()
		if status.GetPhase() != corev1.EvaluationStatusPhase_EVALUATION_STATUS_PHASE_SUCCEEDED {
			return block(ctx, artifacts, key, artifact, fmt.Sprintf("waiting for evaluation %q to succeed", evalRef))
		}
		storePath, err := core.StorePathFromResult(status.GetResult())
		if err != nil {
			return errors.Join(err, setStatus(ctx, artifacts, key, failed(artifact, "", err.Error())))
		}
		present, err := info(ctx, artifact.Spec.Store, storePath)
		if err != nil {
			return errors.Join(err, setStatus(ctx, artifacts, key, failed(artifact, storePath, err.Error())))
		}
		if !present {
			// Not retrying: the reference resolved to a path the store does
			// not have, and nothing but a change to this artifact or to the
			// evaluation can make that different.
			return setStatus(ctx, artifacts, key, failed(artifact, storePath,
				fmt.Sprintf("store path %s is not present in %s", storePath, storeName(artifact.Spec.Store))))
		}
		return setStatus(ctx, artifacts, key, core.ArtifactStatus{
			Phase:              core.ArtifactPhaseSucceeded,
			StorePath:          storePath,
			ObservedGeneration: generation,
		})
	}
}

// Run starts the artifact controller and blocks until ctx is cancelled.
func Run(ctx context.Context) {
	artifactClient := corev1connect.NewArtifactServiceClient(http.DefaultClient, apiAddress())
	evaluationClient := corev1connect.NewEvaluationServiceClient(http.DefaultClient, apiAddress())
	queue := controller.NewWorkQueue(controller.WithRateLimiter(rate.NewLimiter(rate.Limit(1), 1)))
	tracker := controller.NewDependencyTracker()
	informer := controller.NewInformer[*core.Artifact](queue, func(artifact *core.Artifact) string {
		return artifact.Metadata.Name
	}, controller.WithOnChange(trackDependency(tracker)))
	go startArtifactWatch(ctx, artifactClient, informer)
	go startEvaluationWatch(ctx, evaluationClient, queue, tracker)
	reconcile := newReconcile(artifactClient, evaluationClient, informer, nix.PathInfo)
	controller.NewController(informer, reconcile, workerCount()).Run(ctx)
}
