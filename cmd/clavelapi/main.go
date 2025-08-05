package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/erikeah/clavel/cmd/clavelapi/core"
	"github.com/erikeah/clavel/cmd/clavelapi/transport"
	"github.com/erikeah/clavel/internal/repository"
	"github.com/erikeah/clavel/internal/service"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	options := core.GetOptions()
	mux := http.NewServeMux()
	projectRepository := repository.NewProjectRepository() 
	projectService := service.NewProjectService(projectRepository)
	projectPath, projectHandler := transport.NewProjectServiceHandler(projectService)
	mux.Handle(projectPath, projectHandler)
	host := ""
	port := options.ServerPort
	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info(fmt.Sprintf("Service located at %s", addr))
	err := http.ListenAndServe(addr, h2c.NewHandler(mux, &http2.Server{}))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
