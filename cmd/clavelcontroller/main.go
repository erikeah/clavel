package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"time"

	"connectrpc.com/connect"
	projectv1 "github.com/erikeah/clavel/pkg/pb/project/v1"
	"github.com/erikeah/clavel/pkg/pb/project/v1/projectv1connect"
)

func main() {
	client := projectv1connect.NewProjectServiceClient(http.DefaultClient, "http://localhost:8080")
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
}
