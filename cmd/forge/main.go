package main

import (
	"fmt"
	"os"

	"github.com/forgekit/forgekit/internal/cli"
	"github.com/forgekit/forgekit/internal/errs"
	"github.com/forgekit/forgekit/internal/output"
)

func main() {
	root := cli.NewRootCommand()
	if err := root.Execute(); err != nil {
		debug, _ := root.PersistentFlags().GetBool("debug")
		fmt.Fprintf(os.Stderr, "erreur : %v\n", err)
		output.Debug(os.Stderr, debug, err)
		os.Exit(errs.ExitCode(err))
	}
}
