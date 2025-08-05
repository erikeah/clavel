package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/erikeah/clavel/cmd/clavelapi/core"
	"github.com/erikeah/clavel/cmd/clavelapi/transport"
	"github.com/erikeah/clavel/internal/repository"
	"github.com/erikeah/clavel/internal/service"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	options := core.GetOptions()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer cli.Close()
	projectRepository := repository.NewProjectRepository(cli)
	projectService := service.NewProjectService(projectRepository)
	projectPath, projectHandler := transport.NewProjectServiceHandler(projectService)
	mux := http.NewServeMux()
	mux.Handle(projectPath, projectHandler)
	host := ""
	port := options.ServerPort
	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info(fmt.Sprintf("clavelapi binded to %s", addr))
	err = http.ListenAndServe(addr, h2c.NewHandler(mux, &http2.Server{}))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
