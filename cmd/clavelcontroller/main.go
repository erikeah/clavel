package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	evaluationController "github.com/erikeah/clavel/internal/controller/evaluation"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("clavelcontroller starting")
	evaluationController.Run(ctx)
	slog.Info("clavelcontroller stopped")
}
