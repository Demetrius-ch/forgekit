package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/app"
	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/report"
	"github.com/spf13/cobra"
)

const branding = `
  ███████╗ ██████╗ ██████╗  ██████╗ ███████╗
  ██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
  █████╗  ██║   ██║██████╔╝██║  ███╗█████╗
  ██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝
  ██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
  ╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝

  ForgeKit
  Build • Extend • Ship
`

// printBranding prints the ForgeKit branding if conditions are met.
func printBranding(g *globalFlags) {
	console := g.console()

	// Skip branding in JSON or quiet mode
	if console.Format == output.FormatJSON || console.Quiet {
		return
	}

	fmt.Fprint(console.Out, branding)
	fmt.Fprintln(console.Out)
}

// NewRootCommand builds the forge CLI root command.
func NewRootCommand() *cobra.Command {
	g := &globalFlags{}
	root := &cobra.Command{
		Use:           "forge",
		Short:         "ForgeKit — outillage CLI pour backends Go",
		Long:          "ForgeKit crée et maintient des APIs REST Go en architecture hexagonale.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	bindGlobalFlags(root, g)
	root.AddCommand(newInitCommand(g))
	root.AddCommand(newVersionCommand(g))
	root.AddCommand(newAddCommand(g))
	root.AddCommand(newRemoveCommand(g))
	root.AddCommand(newInspectCommand(g))
	root.AddCommand(newDoctorCommand(g))
	root.AddCommand(newAnalyzeCommand(g))
	root.AddCommand(newCheckCommand(g))
	root.AddCommand(newConfigCommand(g))

	// Override help to show branding for 'forge' and 'forge --help' only (not subcommands)
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Only show branding for root command
		if cmd == root {
			printBranding(g)
		}
		cmd.Flags().PrintDefaults()
		fmt.Fprint(cmd.OutOrStdout(), cmd.Long+"\n\n")
		fmt.Fprint(cmd.OutOrStdout(), cmd.UsageString())
	})

	return root
}

func newVersionCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Afficher la version",
		Run: func(cmd *cobra.Command, args []string) {
			console := g.console()

			if console.Format == output.FormatJSON {
				type versionInfo struct {
					SchemaVersion string `json:"schema_version"`
					Name          string `json:"name"`
					Version       string `json:"version"`
					Slogan        string `json:"slogan"`
				}
				_ = console.PrintJSON(versionInfo{
					SchemaVersion: report.JSONSchemaVersion,
					Name:          app.Name,
					Version:       app.Version,
					Slogan:        "Build • Extend • Ship",
				})
				return
			}

			if !console.Quiet {
				fmt.Fprintf(console.Out, "%s version %s\n", app.Name, app.Version)
				fmt.Fprintln(console.Out, "Build • Extend • Ship")
			} else {
				fmt.Fprintf(console.Out, "%s\n", app.Version)
			}
		},
	}
}

func defaultModulePath(projectName string) string {
	user := os.Getenv("USER")
	if user == "" {
		user = "developer"
	}
	return fmt.Sprintf("github.com/%s/%s", user, projectName)
}

func defaultDatabaseName(projectName string) string {
	return strings.ReplaceAll(projectName, "-", "_")
}

func printInitSummaryOpts(projectName, modulePath, targetDir, db string, httpPort, postgresHostPort int) {
	fmt.Fprintf(os.Stdout, `
✓ Projet %q créé avec succès

  Répertoire :  %s
  Module Go :   %s
  Port HTTP :   %d
  PostgreSQL :  localhost:%d

Prochaines étapes recommandées :

  cd %s
  cp .env.example .env
  docker compose -f docker/docker-compose.yml up -d
  go test ./...
  go run ./cmd/server

  forge doctor
  forge check
  forge analyze

Votre projet est prêt pour une première exécution locale et pour les vérifications de qualité.
`, projectName, targetDir, modulePath, httpPort, postgresHostPort, filepath.Base(targetDir))
}
