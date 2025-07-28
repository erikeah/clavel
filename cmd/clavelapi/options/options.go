package options

import (
	"os"
	"strconv"
)

type ClavelapiConfig struct {
	ServerPort int
}

func setOptionsFromEnvVar(config *ClavelapiConfig) {
	if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil {
		config.ServerPort = port
	} else {
		config.ServerPort = 80
	}
}

/*
func setOptionsFromArgs() {
	args := os.Args
	if len(args) != 2 {
		slog.Info("usage: clavelapi <source>")
		os.Exit(1)
	}
}
*/

func GetOptions() ClavelapiConfig {
	//setOptionsFromArgs()
	config := &ClavelapiConfig{}
	setOptionsFromEnvVar(config)
	return *config
}
