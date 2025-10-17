package main

import (
	"fmt"
	"time"

	controller_project "github.com/erikeah/clavel/cmd/clavelcontroller/project"
	"github.com/erikeah/clavel/internal/genericdispatcher"
	"github.com/erikeah/clavel/internal/project"
)

func main() {
	projectsChan, projectErrorsChan := controller_project.ProjectWatcher()
	projectDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[*project.Project]{
			{func(*project.Project) bool { return true }, func(p *project.Project) error { fmt.Println(p); return nil }},
		},
		projectsChan,
	)
	projectErrorsDispatcher := genericdispatcher.NewDispatcher(
		[]genericdispatcher.Rule[error]{},
		projectErrorsChan,
	)
	projectDispatcher.Start()
	go projectErrorsDispatcher.Start()
	for {
		time.Sleep(time.Hour)
	}
}
