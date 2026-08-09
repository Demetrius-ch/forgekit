package main

import (
	"fmt"
	"os"

	"github.com/Demetrius-ch/forgekit/internal/cli"
	"github.com/Demetrius-ch/forgekit/internal/errs"
	"github.com/Demetrius-ch/forgekit/internal/output"
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
