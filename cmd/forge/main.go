// Package forge is the entry point for the ForgeKit CLI.
//
// ForgeKit is a Go CLI tool that generates production-ready REST APIs
// with hexagonal architecture. It scaffolds projects with PostgreSQL,
// Docker, migrations, tests, and provides validation commands.
//
// Commands:
//
//	init     Initialize a new ForgeKit project
//	add      Add features to an existing project (auth, cors, logging, swagger)
//	doctor   Diagnose development environment and project health
//	check    Validate architectural conventions (CI)
//	analyze  Analyze project structure and practices
//	inspect  Inspect ForgeKit project signature
//	config   Manage ForgeKit configuration
//	version  Print version information
//
// Example usage:
//
//	forge init my-api --non-interactive --module github.com/example/my-api
//	forge add auth
//	forge doctor
//	forge check
//
// Documentation: https://github.com/Demetrius-ch/forgekit
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
