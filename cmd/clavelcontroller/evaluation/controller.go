package evaluation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/genericdispatcher"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

func watcher() (<-chan *core.Evaluation, <-chan error) {
	watchUpdates := make(chan *core.Evaluation)
	watchErrors := make(chan error)
	go func() {
		client := corev1connect.NewEvaluationServiceClient(http.DefaultClient, "http://localhost:8080")
		watchResp, err := client.Watch(context.TODO(), &connect.Request[corev1.EvaluationServiceWatchRequest]{
			Msg: &corev1.EvaluationServiceWatchRequest{List: true},
		})
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		defer watchResp.Close()
		defer close(watchUpdates)
		defer close(watchErrors)
		for {
			if !watchResp.Receive() {
				if watchResp.Err() != nil {
					slog.Error(watchResp.Err().Error())
					os.Exit(1)
				}
				os.Exit(0)
			}
			if data := watchResp.Msg().GetData(); data != nil {
				watchUpdates <- data.Convert(nil)
			}
			if err := watchResp.Msg().GetError(); err != nil {
				watchErrors <- errors.New(err.GetMessage())
			}
		}
	}()
	return watchUpdates, watchErrors
}

func Start() {
	evaluationsChan, evaluationErrorsChan := watcher()
	evaluationDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[*core.Evaluation]{
			{Test: func(*core.Evaluation) bool { return true },
				Execute: func(p *core.Evaluation) error { fmt.Println(p); return nil }},
		},
		evaluationsChan,
	)
	evaluationErrorsDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[error]{},
		evaluationErrorsChan,
	)
	go evaluationErrorsDispatcher.Start()
	evaluationDispatcher.Start()
}
