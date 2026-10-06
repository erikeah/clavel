package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/erikeah/clavel/cmd/clavelapi/options"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	evaluationStore := core.NewEvaluationStore(cli)
	evaluationService := core.NewEvaluationService(evaluationStore)
	evaluationPath, evaluationHandler := transport.NewEvaluationServiceHandler(evaluationService)
	mux := http.NewServeMux()
	mux.Handle(evaluationPath, evaluationHandler)
	host := ""
	port := options.ServerPort
	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info(fmt.Sprintf("clavelapi binding to %s", addr))
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{
		Addr:      addr,
		Protocols: protocols,
		Handler:   mux,
	}
	err = server.ListenAndServe()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
