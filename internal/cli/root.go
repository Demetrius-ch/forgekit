package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/app"
	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/projectconfig"
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

func printInitSummaryOpts(projectName, modulePath, targetDir string, httpPort int, dbPort int, cfg projectconfig.ProjectConfig) {
	dbLabel := databaseDisplayName(cfg.Database)
	dockerLabel := "activé"
	if !cfg.Docker {
		dockerLabel = "désactivé"
	}
	authLabel := authDisplayName(cfg.Authentication)
	docsLabel := docsDisplayName(cfg.Documentation)
	testsLabel := testsDisplayName(cfg.Tests)
	ciLabel := ciDisplayName(cfg.CI)
	archLabel := archDisplayName(cfg.Architecture)
	langLabel := languageDisplayName(cfg.Language)

	var dbLine string
	if cfg.Database != projectconfig.DatabaseNone && dbPort > 0 {
		dbLine = fmt.Sprintf("  Base de données : %s (localhost:%d)", dbLabel, dbPort)
	} else if cfg.Database != projectconfig.DatabaseNone {
		dbLine = fmt.Sprintf("  Base de données : %s", dbLabel)
	} else {
		dbLine = "  Base de données : aucune"
	}

	var dockerStep string
	if cfg.Docker {
		dockerStep = "  docker compose -f docker/docker-compose.yml up -d"
	}

	var nextSteps string
	if cfg.IsGo() {
		nextSteps = fmt.Sprintf(`
  cd %s
  cp .env.example .env
%s
  go test ./...
  go run ./cmd/server

  forge doctor
  forge check
  forge analyze`, filepath.Base(targetDir), dockerStep)
	} else if cfg.IsNode() {
		nextSteps = fmt.Sprintf(`
  cd %s
  cp .env.example .env
  npm install
  npm test
  npm run build
  npm start

  forge doctor
  forge check
  forge analyze`, filepath.Base(targetDir))
	}

	fmt.Fprintf(os.Stdout, `
✓ Projet %q créé avec succès

  Répertoire :    %s
  Module Go :     %s
  Port HTTP :     %d
  Langage :       %s
  Architecture :  %s
%s
  Docker :        %s
  Authentification : %s
  Documentation : %s
  Tests :         %s
  CI :            %s

Prochaines étapes recommandées :
%s

Votre projet est prêt pour une première exécution locale et pour les vérifications de qualité.
`, projectName, targetDir, modulePath, httpPort, langLabel, archLabel, dbLine, dockerLabel, authLabel, docsLabel, testsLabel, ciLabel, nextSteps)
}

func databaseDisplayName(db projectconfig.Database) string {
	switch db {
	case projectconfig.DatabasePostgres:
		return "PostgreSQL"
	case projectconfig.DatabaseMySQL:
		return "MySQL"
	case projectconfig.DatabaseSQLite:
		return "SQLite"
	default:
		return "aucune"
	}
}

func authDisplayName(a projectconfig.Authentication) string {
	switch a {
	case projectconfig.AuthenticationJWT:
		return "JWT"
	default:
		return "aucune"
	}
}

func docsDisplayName(d projectconfig.Documentation) string {
	switch d {
	case projectconfig.DocumentationSwagger:
		return "Swagger"
	default:
		return "aucune"
	}
}

func testsDisplayName(t projectconfig.TestStrategy) string {
	switch t {
	case projectconfig.TestStrategyUnitIntegration:
		return "unitaires + intégration"
	default:
		return "unitaires"
	}
}

func ciDisplayName(c projectconfig.CIStrategy) string {
	switch c {
	case projectconfig.CIStrategyGitHub:
		return "GitHub Actions"
	default:
		return "aucune"
	}
}

func archDisplayName(a projectconfig.Architecture) string {
	switch a {
	case projectconfig.ArchitectureHexagonal:
		return "Hexagonal"
	case projectconfig.ArchitectureClean:
		return "Clean Architecture"
	case projectconfig.ArchitectureLayered:
		return "Layered"
	default:
		return string(a)
	}
}

func languageDisplayName(l projectconfig.Language) string {
	switch l {
	case projectconfig.LanguageGo:
		return "Go"
	case projectconfig.LanguageTypeScript:
		return "TypeScript"
	case projectconfig.LanguageJavaScript:
		return "JavaScript"
	case projectconfig.LanguagePython:
		return "Python"
	default:
		return string(l)
	}
}

func runtimeDisplayName(r projectconfig.Runtime) string {
	switch r {
	case projectconfig.RuntimeNode:
		return "Node.js"
	case projectconfig.RuntimeNone:
		return "Aucun"
	default:
		return string(r)
	}
}
