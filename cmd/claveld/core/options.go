package core

import (
	"log/slog"
	"os"
	"strconv"
	"sync"
)

var claveldOptionsInstance *ClaveldOptions = &ClaveldOptions{}

type ClaveldOptions struct {
	Source     string
	ServerPort int
}

var setOptionsFromEnvVar = sync.OnceFunc(func() {
	if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil {
		claveldOptionsInstance.ServerPort = port
	} else {
		claveldOptionsInstance.ServerPort = 80
	}
})

var setOptionsFromArgs = sync.OnceFunc(func() {
	args := os.Args
	if len(args) != 2 {
		slog.Info("usage: claveld <source>")
		os.Exit(1)
	}
	claveldOptionsInstance.Source = args[1]
})

func GetOptions() *ClaveldOptions {
	setOptionsFromArgs()
	setOptionsFromEnvVar()
	return claveldOptionsInstance
}
