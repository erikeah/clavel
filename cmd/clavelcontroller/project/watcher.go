package project

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/project"
	projectv1 "github.com/erikeah/clavel/pkg/api/project/v1"
	"github.com/erikeah/clavel/pkg/api/project/v1/projectv1connect"
)

func ProjectWatcher() (<-chan *project.Project, <-chan error) {
	watchUpdates := make(chan *project.Project)
	watchErrors := make(chan error)
	go func() {
		client := projectv1connect.NewProjectServiceClient(http.DefaultClient, "http://localhost:8080")
		watchResp, err := client.Watch(context.TODO(), &connect.Request[projectv1.ProjectServiceWatchRequest]{
			Msg: &projectv1.ProjectServiceWatchRequest{List: true},
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
