// Package main is the entry point for the poros CLI application.
package main

import (
	"fmt"
	"os"
)

// Version information (set via ldflags during build)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version info for CLI
	SetVersion(version, commit, date)

	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
