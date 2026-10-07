package evaluation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/controller"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/erikeah/clavel/internal/nix"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	apiAddressEnv  = "CLAVEL_API_ADDR"
	defaultAPIAddr = "http://localhost:8080"
	workerCountEnv = "CLAVEL_CONTROLLER_WORKERS"
	defaultWorkers = 4
	watchReconnect = time.Second
)

// evaluator evaluates a flake reference and returns the raw JSON result, which
// reconcile base64-encodes before writing it to the status.
type evaluator func(ctx context.Context, reference string) (string, error)

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

// startInformer pumps the watch stream into the informer, reconnecting on
// stream errors. The first request of every stream asks for a snapshot, which
// the applier turns into an informer Replace, so a reconnect also evicts the
// resources deleted while the stream was down.
func startInformer(ctx context.Context, client corev1connect.EvaluationServiceClient, informer *controller.Informer[*core.Evaluation]) {
	watchRequest := &connect.Request[corev1.EvaluationServiceWatchRequest]{
		Msg: &corev1.EvaluationServiceWatchRequest{List: true},
	}
	for {
		watchResponse, err := client.Watch(ctx, watchRequest)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("watch failed", "error", err)
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
			slog.Error("watch stream ended", "error", err)
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
	informer *controller.Informer[*core.Evaluation]
	snapshot []*core.Evaluation
	syncing  bool
}

func newWatchApplier(informer *controller.Informer[*core.Evaluation]) *watchApplier {
	return &watchApplier{informer: informer, syncing: true}
}

func (applier *watchApplier) apply(message *corev1.EvaluationServiceWatchResponse) {
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

// setStatus writes the evaluation status through read-modify-write: Show
// obtains the current resource version, the status is replaced, and Update
// persists the full object (no field mask). A no-op update succeeds silently.
func setStatus(
	ctx context.Context,
	client corev1connect.EvaluationServiceClient,
	name string,
	status core.EvaluationStatus,
) error {
	response, err := client.Show(ctx, connect.NewRequest(&corev1.EvaluationServiceShowRequest{Name: name}))
	if err != nil {
		return err
	}
	current := response.Msg.GetData()
	if current.GetStatus().Convert() == status {
		return nil
	}
	protoStatus := &corev1.EvaluationStatus{}
	protoStatus.Set(&status)
	current.Status = protoStatus
	_, err = client.Update(ctx, connect.NewRequest(&corev1.EvaluationServiceUpdateRequest{
		Name: name,
		Data: current,
	}))
	return err
}

// controllerFinalizer is this controller's finalizer on a resource. It is
// dropped during finalization; registering it is deferred until there is
// cleanup to perform on deletion.
const controllerFinalizer = "clavel.core/evaluation"

// finalize drives a terminating resource to removal. The resource is deleted
// here rather than by the service: first this controller's finalizer goes, then
// once no finalizer blocks it anymore the delete is issued. Finalizers owned by
// someone else are left alone, and the resource waits for their owner.
func finalize(
	ctx context.Context,
	client corev1connect.EvaluationServiceClient,
	name string,
	evaluation *core.Evaluation,
) error {
	finalizers := evaluation.Metadata.Finalizers
	if slices.Contains(finalizers, controllerFinalizer) {
		return updateFinalizers(ctx, client, name, withoutFinalizer(finalizers, controllerFinalizer))
	}
	if len(finalizers) > 0 {
		return nil
	}
	_, err := client.Delete(ctx, connect.NewRequest(&corev1.EvaluationServiceDeleteRequest{Name: name}))
	return err
}

func withoutFinalizer(finalizers []string, drop string) []string {
	remaining := make([]string, 0, len(finalizers))
	for _, finalizer := range finalizers {
		if finalizer != drop {
			remaining = append(remaining, finalizer)
		}
	}
	return remaining
}

// updateFinalizers rewrites metadata.finalizers of the current object. The
// update has to be masked: an unmasked Update leaves slice fields untouched.
func updateFinalizers(ctx context.Context, client corev1connect.EvaluationServiceClient, name string, finalizers []string) error {
	response, err := client.Show(ctx, connect.NewRequest(&corev1.EvaluationServiceShowRequest{Name: name}))
	if err != nil {
		return err
	}
	current := response.Msg.GetData()
	if current == nil {
		return exceptions.InternalFailure
	}
	if current.Metadata == nil {
		current.Metadata = &corev1.Metadata{}
	}
	current.Metadata.Finalizers = finalizers
	_, err = client.Update(ctx, connect.NewRequest(&corev1.EvaluationServiceUpdateRequest{
		Name:       name,
		Data:       current,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"data.metadata.finalizers"}},
	}))
	return err
}

func newReconcile(
	client corev1connect.EvaluationServiceClient,
	informer *controller.Informer[*core.Evaluation],
	evaluate evaluator,
) controller.ReconcileFunc {
	return func(ctx context.Context, key string) error {
		evaluation, ok := informer.Get(key)
		if !ok {
			// Resource deleted before reconcile; cleanup/finalizers are a follow-up.
			slog.Info("evaluation deleted", "name", key)
			return nil
		}
		if evaluation.Metadata.DeletionTimestamp != nil {
			return finalize(ctx, client, key, evaluation)
		}
		if evaluation.Status.Phase == core.EvaluationPhaseSucceeded &&
			evaluation.Status.ObservedGeneration == evaluation.Metadata.Generation {
			return nil
		}
		// Write Pending once per generation: only for a never-observed or stale
		// generation, so retries of the same generation don't re-write status and
		// re-trigger watch events.
		if evaluation.Status.Phase == core.EvaluationPhaseUnspecified ||
			evaluation.Status.ObservedGeneration != evaluation.Metadata.Generation {
			pending := core.EvaluationStatus{
				Phase:              core.EvaluationPhasePending,
				Message:            "evaluating",
				Result:             evaluation.Status.Result,
				ObservedGeneration: evaluation.Metadata.Generation,
			}
			if err := setStatus(ctx, client, key, pending); err != nil {
				return err
			}
		}
		result, err := evaluate(ctx, evaluation.Spec.Reference)
		if err != nil {
			failed := core.EvaluationStatus{
				Phase:              core.EvaluationPhaseFailed,
				Message:            err.Error(),
				ObservedGeneration: evaluation.Metadata.Generation,
			}
			return errors.Join(err, setStatus(ctx, client, key, failed))
		}
		succeeded := core.EvaluationStatus{
			Phase:              core.EvaluationPhaseSucceeded,
			Result:             core.EncodeResult([]byte(result)),
			ObservedGeneration: evaluation.Metadata.Generation,
		}
		return setStatus(ctx, client, key, succeeded)
	}
}

// Run starts the evaluation controller and blocks until ctx is cancelled.
func Run(ctx context.Context) {
	client := corev1connect.NewEvaluationServiceClient(http.DefaultClient, apiAddress())
	queue := controller.NewWorkQueue(controller.WithRateLimiter(rate.NewLimiter(rate.Limit(1), 1)))
	informer := controller.NewInformer[*core.Evaluation](queue, func(evaluation *core.Evaluation) string {
		return evaluation.Metadata.Name
	})
	go startInformer(ctx, client, informer)
	reconcile := newReconcile(client, informer, nix.Eval)
	controller.NewController(informer, reconcile, workerCount()).Run(ctx)
}
