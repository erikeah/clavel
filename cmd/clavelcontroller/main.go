package main

import (
	"time"

	evaluationController "github.com/erikeah/clavel/cmd/clavelcontroller/evaluation"
)

func main() {
	go evaluationController.Start()
	for {
		time.Sleep(time.Hour)
	}
}
