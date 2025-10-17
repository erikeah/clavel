package genericdispatcher

import (
	"log/slog"
)

type Dispatcher[R any] struct {
	rules   []Rule[R]
	channel <-chan R
}

func (d *Dispatcher[R]) dispatch(resource R) {
	for _, rule := range d.rules {
		if rule.Test(resource) {
			if err := rule.Execute(resource); err != nil {
				// TODO: Error handling
				slog.Error(err.Error())
			}
			break
		}
	}
}

func (d *Dispatcher[R]) Start() {
	for resource := range d.channel {
		d.dispatch(resource)
	}
}

func NewDispatcher[R any](rules []Rule[R], channel <-chan R) *Dispatcher[R] {
	return &Dispatcher[R]{
		rules:   rules,
		channel: channel,
	}
}
