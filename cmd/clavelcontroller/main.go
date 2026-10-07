package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	artifactController "github.com/erikeah/clavel/internal/controller/artifact"
	evaluationController "github.com/erikeah/clavel/internal/controller/evaluation"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("clavelcontroller starting")
	var controllers sync.WaitGroup
	controllers.Go(func() { evaluationController.Run(ctx) })
	controllers.Go(func() { artifactController.Run(ctx) })
	controllers.Wait()
	slog.Info("clavelcontroller stopped")
}
