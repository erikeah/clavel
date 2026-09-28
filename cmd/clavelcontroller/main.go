package main

import (
	"time"

	sourceController "github.com/erikeah/clavel/cmd/clavelcontroller/source"
)

func main() {
	go sourceController.Start()
	for {
		time.Sleep(time.Hour)
	}
}
