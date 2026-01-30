package main

import (
	"fmt"
	"os"

	"github.com/akrishnanDG/glue-to-ccsr/internal/cli"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	rootCmd := cli.NewRootCmd(Version, BuildTime)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
