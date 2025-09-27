package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	projectv1 "github.com/erikeah/clavel/pkg/api/project/v1"
	"github.com/erikeah/clavel/pkg/api/project/v1/projectv1connect"
)

func main() {
	client := projectv1connect.NewProjectServiceClient(http.DefaultClient, "http://localhost:8080")
	watchResp, err := client.Watch(context.TODO(), &connect.Request[projectv1.ProjectServiceWatchRequest]{
		Msg: &projectv1.ProjectServiceWatchRequest{List: true},
	})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer watchResp.Close()
	for {
		if !watchResp.Receive() {
			if watchResp.Err() != nil {
				slog.Error(watchResp.Err().Error())
				os.Exit(1)
			}
			os.Exit(0)
		}
		if data := watchResp.Msg().GetData(); data != nil {
			slog.Info(fmt.Sprint(data))
		}
		if err := watchResp.Msg().GetError(); err != nil {
			slog.Info(fmt.Sprint(err))
		}

	}
	/*
		for {
			ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
			response, err := client.List(ctx, &connect.Request[projectv1.ProjectServiceListRequest]{})
			if err != nil {
				slog.Error(err.Error())
				break
			}
			cancel()
			for _, project := range response.Msg.GetData() {
				flakeref := project.GetSpec().GetFlakeref()
				cmd := exec.Command("nix", "flake", "prefetch", "--json", flakeref)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				if err := cmd.Run(); err != nil {
					slog.Error(err.Error())
					break
				}
				var result struct{ Hash string }
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
					slog.Error(err.Error())
					break
				}
				if result.Hash != project.GetStatus().GetSource().GetHash() {
					slog.Info(fmt.Sprintf("Starting evaluation of %s", project.Name))
				}
			}
			time.Sleep(time.Second * 30)
		}
	*/
}
