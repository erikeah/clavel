package evaluation

import (
	"context"
	"errors"
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

// startInformer pumps the watch stream into informer.Add, reconnecting on
// stream errors. Initial sync comes from the server via the List request flag.
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
		for watchResponse.Receive() {
			if data := watchResponse.Msg().GetData(); data != nil {
				informer.Add(data.Convert(nil))
			}
			if streamError := watchResponse.Msg().GetError(); streamError != nil {
				slog.Error(streamError.GetMessage())
			}
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
			return nil
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
		return evaluation.Name
	})
	go startInformer(ctx, client, informer)
	reconcile := newReconcile(client, informer, nix.Eval)
	controller.NewController(informer, reconcile, workerCount()).Run(ctx)
}
