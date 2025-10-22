package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/erikeah/clavel/cmd/clavelapi/options"
	apiserversource "github.com/erikeah/clavel/cmd/clavelapi/source"
	sourceapi "github.com/erikeah/clavel/internal/source/api"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	options := options.GetOptions()
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer cli.Close()
	sourceStore := sourceapi.NewSourceStore(cli)
	sourceService := sourceapi.NewSourceService(sourceStore)
	sourcePath, sourceHandler := apiserversource.NewSourceServiceHandler(sourceService)
	mux := http.NewServeMux()
	mux.Handle(sourcePath, sourceHandler)
	host := ""
	port := options.ServerPort
	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info(fmt.Sprintf("clavelapi binding to %s", addr))
	err = http.ListenAndServe(addr, h2c.NewHandler(mux, &http2.Server{}))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
