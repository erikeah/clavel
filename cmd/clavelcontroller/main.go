package main

import (
	"fmt"
	"time"

	controller_source "github.com/erikeah/clavel/cmd/clavelcontroller/source"
	"github.com/erikeah/clavel/internal/genericdispatcher"
	"github.com/erikeah/clavel/internal/source"
)

func main() {
	sourcesChan, sourceErrorsChan := controller_source.SourceWatcher()
	sourceDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[*source.Source]{
			{func(*source.Source) bool { return true }, func(p *source.Source) error { fmt.Println(p); return nil }},
		},
		sourcesChan,
	)
	sourceErrorsDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[error]{},
		sourceErrorsChan,
	)
	sourceDispatcher.Start()
	go sourceErrorsDispatcher.Start()
	for {
		time.Sleep(time.Hour)
	}
}
