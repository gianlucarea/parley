package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gianlucarea/parley/internal/collection"
	"github.com/gianlucarea/parley/internal/env"
	"github.com/gianlucarea/parley/internal/tui"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	collections, err := collection.ParseDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading collections:", err)
		os.Exit(1)
	}

	// Load .parley.env if present; missing file is not an error.
	envPath := ".parley.env"
	envVars, err := env.Load(envPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "error loading .parley.env:", err)
		os.Exit(1)
	}
	if envVars == nil {
		envVars = make(map[string]string)
	}

	if err := tui.Run(collections, envVars); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
