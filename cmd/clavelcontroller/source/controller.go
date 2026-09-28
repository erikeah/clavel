package source

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/genericdispatcher"
	"github.com/erikeah/clavel/internal/source"
	sourcev1 "github.com/erikeah/clavel/pkg/api/source/v1"
	"github.com/erikeah/clavel/pkg/api/source/v1/sourcev1connect"
)

func watcher() (<-chan *source.Source, <-chan error) {
	watchUpdates := make(chan *source.Source)
	watchErrors := make(chan error)
	go func() {
		client := sourcev1connect.NewSourceServiceClient(http.DefaultClient, "http://localhost:8080")
		watchResp, err := client.Watch(context.TODO(), &connect.Request[sourcev1.SourceServiceWatchRequest]{
			Msg: &sourcev1.SourceServiceWatchRequest{List: true},
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
	sourcesChan, sourceErrorsChan := watcher()
	sourceDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[*source.Source]{
			{Test: func(*source.Source) bool { return true },
				Execute: func(p *source.Source) error { fmt.Println(p); return nil }},
		},
		sourcesChan,
	)
	sourceErrorsDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[error]{},
		sourceErrorsChan,
	)
	go sourceErrorsDispatcher.Start()
	sourceDispatcher.Start()
}
