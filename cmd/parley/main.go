package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gianlucarea/parley/internal/collection"
	"github.com/gianlucarea/parley/internal/env"
	"github.com/gianlucarea/parley/internal/tui"
)

func main() {
	var envName string
	flag.StringVar(&envName, "env", "", "named environment to use (loads .parley.<name>.env over .parley.env)")
	flag.StringVar(&envName, "e", "", "shorthand for --env")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: parley [--env name] [dir]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	collections, err := collection.ParseDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading collections:", err)
		os.Exit(1)
	}

	// Load .parley.env and any .parley.<name>.env files; missing files are not an error.
	envs, err := env.Discover(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading environment:", err)
		os.Exit(1)
	}
	if _, err := envs.Resolve(envName); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := tui.Run(collections, envs, envName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
