package main

import "log/slog"

func main() {
	slog.Info("clavelcontroller starting")
	controller := newController()
	controller.Start()
}
