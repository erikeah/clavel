package main

import (
	"fmt"
	"os"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: clavelctl <command> [flags]\n\ncommands:\n  apply <flake-ref>   evaluate a clavel configuration and push it to clavelapi\n")
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "apply":
		applyCommand(os.Args[2:])
	default:
		usage()
	}
}

func applyCommand(args []string) {
	address := "http://localhost:8080"
	config := "default"
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--address" || arg == "-address":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--address requires a value")
				os.Exit(2)
			}
			i++
			address = args[i]
		case strings.HasPrefix(arg, "--address="):
			address = strings.TrimPrefix(arg, "--address=")
		case arg == "--config" || arg == "-config":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--config requires a value")
				os.Exit(2)
			}
			i++
			config = args[i]
		case strings.HasPrefix(arg, "--config="):
			config = strings.TrimPrefix(arg, "--config=")
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			os.Exit(2)
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 1 {
		fmt.Fprintf(os.Stderr, "usage: clavelctl apply [flags] <flake-ref>\n")
		os.Exit(2)
	}
	if err := apply(positional[0], config, address); err != nil {
		fmt.Fprintln(os.Stderr, "apply failed:", err)
		os.Exit(1)
	}
}
