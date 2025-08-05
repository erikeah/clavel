package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/erikeah/clavel/cmd/claveld/core"
	"github.com/erikeah/clavel/cmd/claveld/server"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	options := core.GetOptions()
	mux := http.NewServeMux()
	path, handler := server.NewAgentServiceHandler()
	mux.Handle(path, handler)
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
