package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Demetrius-ch/forgekit/internal/app"
	"github.com/spf13/cobra"
)

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
	root.AddCommand(newVersionCommand())
	root.AddCommand(newAddCommand(g))
	root.AddCommand(newDoctorCommand(g))
	root.AddCommand(newAnalyzeCommand(g))
	root.AddCommand(newCheckCommand(g))
	root.AddCommand(newConfigCommand(g))

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Afficher la version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s\n", app.Name, app.Version)
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

func printInitSummaryOpts(projectName, modulePath, targetDir, db string, port int) {
	fmt.Fprintf(os.Stdout, `
✓ Projet %q créé avec succès

  Répertoire :  %s
  Module Go :   %s
  Port HTTP :   %d
  Base PostgreSQL : %s

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
`, projectName, targetDir, modulePath, port, db, filepath.Base(targetDir))
}
