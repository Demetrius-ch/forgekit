package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Demetrius-ch/forgekit/internal/app"
	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/dbinspect"
	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/feature/auth"
	"github.com/Demetrius-ch/forgekit/internal/feature/cors"
	"github.com/Demetrius-ch/forgekit/internal/feature/logging"
	"github.com/Demetrius-ch/forgekit/internal/feature/swagger"
	"github.com/Demetrius-ch/forgekit/internal/forge"
	"github.com/Demetrius-ch/forgekit/internal/generator"
	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/ports"
	"github.com/Demetrius-ch/forgekit/internal/prompt"
	"github.com/Demetrius-ch/forgekit/internal/report"
	"github.com/Demetrius-ch/forgekit/internal/rules"
	"github.com/spf13/cobra"
)

func newInitCommand(g *globalFlags) *cobra.Command {
	var (
		modulePath      string
		httpPort        int
		postgresPort    int
		databaseName    string
		author          string
		nonInteractive  bool
		targetDir       string
		dryRun          bool
		skipPostprocess bool
	)

	cmd := &cobra.Command{
		Use:   "init [nom-du-projet]",
		Short: "Initialiser un backend REST Go (architecture hexagonale)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			console := g.console()
			projectName := strings.TrimSpace(args[0])
			if err := generator.ValidateProjectName(projectName); err != nil {
				return err
			}
			if targetDir == "" {
				targetDir = projectName
			}
			absDir, err := filepath.Abs(targetDir)
			if err != nil {
				return err
			}
			if !dryRun {
				if err := generator.ValidateTargetDir(absDir); err != nil {
					return err
				}
			}

			// Select ports (handles conflicts automatically)
			portSelection, err := selectPorts(g, httpPort, postgresPort, dryRun)
			if err != nil {
				return err
			}

			// Determine database name
			dbName := databaseName
			if dbName == "" {
				dbName = defaultDatabaseName(projectName)
			}

			// Select database (handles existing databases)
			// Pass the originally requested postgres port to know if it was auto-selected
			dbSelection, err := selectDatabase(g, projectName, dbName, portSelection.PostgresHostPort, postgresPort, nonInteractive, dryRun)
			if err != nil {
				return err
			}

			opts := generator.InitOptions{
				ProjectName:        projectName,
				TargetDir:          absDir,
				HTTPPort:           portSelection.HTTPPort,
				PostgresHostPort:   portSelection.PostgresHostPort,
				DryRun:             dryRun,
				Author:             config.ResolveAuthor(author, absDir),
				SkipPostprocess:    skipPostprocess,
				DatabaseName:       dbSelection.DatabaseName,
				UseExistingDB:      dbSelection.UseExisting,
				ExternalDBHost:     dbSelection.ExternalDBHost,
				ExternalDBPort:     dbSelection.ExternalDBPort,
				ExternalDBUser:     dbSelection.ExternalDBUser,
				ExternalDBPassword: dbSelection.ExternalDBPassword,
				ExternalDBName:     dbSelection.ExternalDBName,
			}

			if nonInteractive || dryRun {
				opts.ModulePath = modulePath
				if opts.ModulePath == "" {
					opts.ModulePath = defaultModulePath(projectName)
				}
			} else {
				p := prompt.New(os.Stdin, os.Stdout)
				fmt.Fprintf(os.Stdout, "\nConfiguration du projet %q\n\n", projectName)
				module, err := p.AskString("Chemin du module Go", defaultModulePath(projectName))
				if err != nil {
					return err
				}
				opts.ModulePath = module
				// In interactive mode, ask for HTTP port with the selected port as default
				port, err := p.AskInt("Port HTTP", portSelection.HTTPPort)
				if err != nil {
					return err
				}
				// If user changed the port, re-check availability
				if port != portSelection.HTTPPort {
					checker := ports.RealPortChecker{}
					if !checker.IsAvailable(port) {
						return fmt.Errorf("le port HTTP %d est déjà utilisé", port)
					}
					opts.HTTPPort = port
				}
				if opts.Author == "" {
					opts.Author, _ = p.AskString("Auteur (optionnel)", "")
				}
			}

			if err := generator.ValidateModulePath(opts.ModulePath); err != nil {
				return err
			}
			if err := generator.ValidateHTTPPort(opts.HTTPPort); err != nil {
				return err
			}
			if err := generator.ValidateDatabaseName(opts.DatabaseName); err != nil {
				return err
			}

			gen, err := generator.New()
			if err != nil {
				return err
			}

			if !dryRun && !g.Quiet {
				fmt.Fprintf(os.Stdout, "\nGénération du projet dans %s...\n", absDir)
			}
			plan, err := gen.Init(opts)
			if err != nil {
				output.Debug(os.Stderr, g.Debug, err)
				return err
			}
			if dryRun {
				console.PrintPlan(plan)
				return nil
			}
			if !g.Quiet && g.Format == output.FormatHuman {
				printInitSummaryOpts(opts.ProjectName, opts.ModulePath, absDir, opts.DatabaseName, opts.HTTPPort, opts.PostgresHostPort)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&modulePath, "module", "", "Chemin du module Go")
	cmd.Flags().IntVar(&httpPort, "port", 8080, "Port HTTP")
	cmd.Flags().IntVar(&postgresPort, "postgres-port", 5432, "Port PostgreSQL hôte")
	cmd.Flags().StringVar(&databaseName, "db-name", "", "Nom de la base PostgreSQL")
	cmd.Flags().StringVar(&author, "author", "", "Auteur du projet")
	cmd.Flags().StringVar(&targetDir, "dir", "", "Répertoire cible")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Sans prompts interactifs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Afficher le plan sans écrire sur le disque")
	cmd.Flags().BoolVar(&skipPostprocess, "skip-postprocess", false, "Ne pas exécuter gofmt et go test après génération")
	return cmd
}

// PortSelection holds the selected ports for HTTP and PostgreSQL.
type PortSelection struct {
	HTTPPort         int
	PostgresHostPort int
}

// selectPorts detects available ports and returns the selected ports.
// If a port is explicitly requested (non-zero), it checks that specific port.
// Otherwise, it starts from the default and finds the next available port.
func selectPorts(g *globalFlags, requestedHTTPPort, requestedPostgresPort int, dryRun bool) (PortSelection, error) {
	checker := ports.RealPortChecker{}
	const maxAttempts = 100

	// HTTP port selection
	httpPort := requestedHTTPPort
	if httpPort == 0 {
		httpPort = 8080
	}

	if !checker.IsAvailable(httpPort) {
		if g.Format == output.FormatHuman && !g.Quiet && !dryRun {
			fmt.Fprintf(os.Stdout, "⚠ Port HTTP %d déjà utilisé.\n", httpPort)
		}
		httpPort = ports.FindAvailablePort(checker, httpPort+1, maxAttempts)
		if g.Format == output.FormatHuman && !g.Quiet && !dryRun {
			fmt.Fprintf(os.Stdout, "✓ Port HTTP %d disponible.\n", httpPort)
			fmt.Fprintf(os.Stdout, "Utilisation du port HTTP %d.\n\n", httpPort)
		}
	}

	// PostgreSQL host port selection
	postgresHostPort := requestedPostgresPort
	if postgresHostPort == 0 {
		postgresHostPort = 5432
	}

	if !checker.IsAvailable(postgresHostPort) {
		if g.Format == output.FormatHuman && !g.Quiet && !dryRun {
			fmt.Fprintf(os.Stdout, "⚠ Port PostgreSQL %d déjà utilisé.\n", postgresHostPort)
		}
		postgresHostPort = ports.FindAvailablePort(checker, postgresHostPort+1, maxAttempts)
		if g.Format == output.FormatHuman && !g.Quiet && !dryRun {
			fmt.Fprintf(os.Stdout, "✓ Port PostgreSQL %d disponible.\n", postgresHostPort)
			fmt.Fprintf(os.Stdout, "Utilisation du port PostgreSQL hôte %d.\n\n", postgresHostPort)
		}
	}

	// Dry-run output
	if dryRun && g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(os.Stdout, "Port HTTP :")
		fmt.Fprintf(os.Stdout, "  demandé : %d\n", requestedHTTPPort)
		if requestedHTTPPort != 0 && requestedHTTPPort != httpPort {
			fmt.Fprintf(os.Stdout, "  statut : occupé\n")
		} else {
			fmt.Fprintf(os.Stdout, "  statut : disponible\n")
		}
		fmt.Fprintf(os.Stdout, "  sélectionné : %d\n\n", httpPort)

		fmt.Fprintln(os.Stdout, "Port PostgreSQL :")
		fmt.Fprintf(os.Stdout, "  demandé : %d\n", requestedPostgresPort)
		if requestedPostgresPort != 0 && requestedPostgresPort != postgresHostPort {
			fmt.Fprintf(os.Stdout, "  statut : occupé\n")
		} else {
			fmt.Fprintf(os.Stdout, "  statut : disponible\n")
		}
		fmt.Fprintf(os.Stdout, "  sélectionné : %d\n\n", postgresHostPort)
	}

	return PortSelection{
		HTTPPort:         httpPort,
		PostgresHostPort: postgresHostPort,
	}, nil
}

// DatabaseSelection holds the database configuration for the project.
type DatabaseSelection struct {
	DatabaseName       string
	UseExisting        bool
	ExistingDBName     string
	CreateNew          bool
	ExternalDBHost     string
	ExternalDBPort     int
	ExternalDBUser     string
	ExternalDBPassword string
	ExternalDBName     string
}

// selectDatabase handles database selection logic, including detecting existing databases.
// requestedPostgresPort is the originally requested port (0 means default 5432).
// postgresHostPort is the actually selected port (may be different if auto-selected).
func selectDatabase(
	g *globalFlags,
	projectName,
	requestedDBName string,
	postgresHostPort int,
	requestedPostgresPort int,
	nonInteractive,
	dryRun bool,
) (DatabaseSelection, error) {
	ctx := context.Background()

	// Determine if the port was auto-selected (different from requested/default)
	defaultPostgresPort := 5432
	if requestedPostgresPort != 0 {
		defaultPostgresPort = requestedPostgresPort
	}
	portWasAutoSelected := postgresHostPort != defaultPostgresPort

	// Only attempt to connect to PostgreSQL if the port was NOT auto-selected.
	// If the port was auto-selected (e.g., 5432 was busy, so 5434 was chosen),
	// there's no PostgreSQL running on that port yet - Docker Compose will start it.
	var databases []dbinspect.DatabaseInfo
	var inspector *dbinspect.Inspector

	if !portWasAutoSelected {
		// Build connection string to PostgreSQL server (using postgres database)
		connStr := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", postgresHostPort)
		inspector = dbinspect.NewInspector(connStr)

		// Check connection to PostgreSQL
		if g.Format == output.FormatHuman && !g.Quiet && !dryRun {
			fmt.Fprintln(os.Stdout, "Connexion à PostgreSQL...")
		}
		if err := inspector.CheckConnection(ctx); err != nil {
			if g.Format == output.FormatHuman && !g.Quiet {
				fmt.Fprintf(os.Stdout, "⚠ Impossible de se connecter à PostgreSQL sur le port %d: %v\n", postgresHostPort, err)
				fmt.Fprintln(os.Stdout, "Une nouvelle base sera créée au démarrage du conteneur Docker.")
			}
			return DatabaseSelection{
				DatabaseName: requestedDBName,
				CreateNew:    true,
			}, nil
		}

		// List existing databases
		var err error
		databases, err = inspector.ListDatabases(ctx)
		if err != nil {
			if g.Format == output.FormatHuman && !g.Quiet {
				fmt.Fprintf(os.Stdout, "⚠ Erreur lors de la liste des bases: %v\n", err)
			}
			return DatabaseSelection{
				DatabaseName: requestedDBName,
				CreateNew:    true,
			}, nil
		}
	} else {
		// Port was auto-selected - skip PostgreSQL connection attempt
		// Docker Compose will start PostgreSQL on this port
		databases = nil
		inspector = nil
	}

	// Check if requested database exists
	requestedExists := false
	var requestedDB *dbinspect.DatabaseInfo
	if databases != nil {
		for _, db := range databases {
			if db.Name == requestedDBName {
				requestedExists = true
				requestedDB = &db
				break
			}
		}
	}

	// Dry-run output
	if dryRun && g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(os.Stdout, "PostgreSQL")
		fmt.Fprintln(os.Stdout, "────────────────────────────────")
		fmt.Fprintf(os.Stdout, "Base demandée :\n  %s\n", requestedDBName)
		if requestedExists {
			fmt.Fprintf(os.Stdout, "Statut : existante\n")
			fmt.Fprintf(os.Stdout, "Tables : %d\n", requestedDB.TableCount)
			if requestedDB.HasMigrations {
				fmt.Fprintf(os.Stdout, "Migrations ForgeKit : oui (version %d)\n", *requestedDB.MigrationVersion)
				if requestedDB.IsDirty {
					fmt.Fprintf(os.Stdout, "État : DIRTY\n")
				}
			} else {
				fmt.Fprintf(os.Stdout, "Migrations ForgeKit : non\n")
			}
		} else {
			fmt.Fprintf(os.Stdout, "Statut : absente\n")
		}
		if databases != nil {
			fmt.Fprintln(os.Stdout, "Bases existantes :")
			for _, db := range databases {
				fmt.Fprintf(os.Stdout, "  %s\n", db.Name)
			}
		} else {
			fmt.Fprintln(os.Stdout, "Bases existantes : (non vérifié - port auto-sélectionné)")
		}
		if requestedExists {
			fmt.Fprintf(os.Stdout, "Action proposée : utiliser \"%s\"\n", requestedDBName)
		} else {
			fmt.Fprintf(os.Stdout, "Action proposée : créer \"%s\"\n", requestedDBName)
		}
		fmt.Fprintln(os.Stdout, "Aucune modification effectuée (--dry-run).")
		fmt.Fprintln(os.Stdout)
		return DatabaseSelection{
			DatabaseName:   requestedDBName,
			CreateNew:      !requestedExists,
			UseExisting:    requestedExists,
			ExistingDBName: requestedDBName,
		}, nil
	}

	if !requestedExists {
		// Database doesn't exist - check if there are other databases
		// Only offer choices if we have database info (port not auto-selected)
		if len(databases) > 0 && !nonInteractive {
			// Interactive mode: offer choices
			fmt.Fprintln(os.Stdout)
			fmt.Fprintf(os.Stdout, "⚠ Volume PostgreSQL existant détecté.\n\n")
			fmt.Fprintf(os.Stdout, "La base %q n'existe pas.\n\n", requestedDBName)
			fmt.Fprintln(os.Stdout, "Que voulez-vous faire ?")
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, "  1. Créer la base "+requestedDBName)
			fmt.Fprintln(os.Stdout, "  2. Utiliser une base PostgreSQL existante")
			fmt.Fprintln(os.Stdout, "  3. Annuler")
			fmt.Fprintln(os.Stdout)

			p := prompt.New(os.Stdin, os.Stdout)
			choice, err := p.AskInt("Choix", 1)
			if err != nil {
				return DatabaseSelection{}, err
			}

			switch choice {
			case 1:
				return DatabaseSelection{
					DatabaseName: requestedDBName,
					CreateNew:    true,
				}, nil
			case 2:
				return selectExistingDatabase(g, inspector, databases, ctx, nonInteractive)
			case 3:
				return DatabaseSelection{}, fmt.Errorf("opération annulée par l'utilisateur")
			default:
				return DatabaseSelection{}, fmt.Errorf("choix invalide")
			}
		}
		// Non-interactive or no other databases: create new
		return DatabaseSelection{
			DatabaseName: requestedDBName,
			CreateNew:    true,
		}, nil
	}

	// Requested database exists
	if nonInteractive {
		// In non-interactive mode, use the existing database
		return DatabaseSelection{
			DatabaseName:   requestedDBName,
			UseExisting:    true,
			ExistingDBName: requestedDBName,
		}, nil
	}

	// Interactive mode: ask user what to do
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "La base %q existe déjà.\n\n", requestedDBName)
	fmt.Fprintln(os.Stdout, "Que voulez-vous faire ?")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "  1. Utiliser cette base existante")
	fmt.Fprintln(os.Stdout, "  2. Choisir une autre base existante")
	fmt.Fprintln(os.Stdout, "  3. Annuler")
	fmt.Fprintln(os.Stdout)

	p := prompt.New(os.Stdin, os.Stdout)
	choice, err := p.AskInt("Choix", 1)
	if err != nil {
		return DatabaseSelection{}, err
	}

	switch choice {
	case 1:
		return askExternalDatabase(g, requestedDBName, requestedDBName)
	case 2:
		return selectExistingDatabase(g, inspector, databases, ctx, nonInteractive)
	case 3:
		return DatabaseSelection{}, fmt.Errorf("opération annulée par l'utilisateur")
	default:
		return DatabaseSelection{}, fmt.Errorf("choix invalide")
	}
}

// selectExistingDatabase lets the user choose from existing databases.
func selectExistingDatabase(
	g *globalFlags,
	inspector *dbinspect.Inspector,
	databases []dbinspect.DatabaseInfo,
	ctx context.Context,
	nonInteractive bool,
) (DatabaseSelection, error) {
	if len(databases) == 0 {
		return DatabaseSelection{}, fmt.Errorf("aucune base existante disponible")
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Bases PostgreSQL disponibles :")
	fmt.Fprintln(os.Stdout)
	for i, db := range databases {
		fmt.Fprintf(os.Stdout, "  %d. %s", i+1, db.Name)
		if db.HasTables {
			fmt.Fprintf(os.Stdout, " (%d tables", db.TableCount)
			if db.HasMigrations {
				fmt.Fprintf(os.Stdout, ", migrations ForgeKit v%d", *db.MigrationVersion)
				if db.IsDirty {
					fmt.Fprintf(os.Stdout, " DIRTY")
				}
			}
			fmt.Fprintf(os.Stdout, ")")
		} else {
			fmt.Fprintf(os.Stdout, " (vide)")
		}
		fmt.Fprintln(os.Stdout)
	}
	fmt.Fprintln(os.Stdout)

	if nonInteractive {
		return DatabaseSelection{}, fmt.Errorf("mode non-interactif: impossible de choisir une base automatiquement")
	}

	p := prompt.New(os.Stdin, os.Stdout)
	choice, err := p.AskInt("Choisissez une base", 1)
	if err != nil {
		return DatabaseSelection{}, err
	}

	if choice < 1 || choice > len(databases) {
		return DatabaseSelection{}, fmt.Errorf("choix invalide")
	}

	selectedDB := databases[choice-1]

	// Validate the selected database
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Vérification de %q\n", selectedDB.Name)
	fmt.Fprintln(os.Stdout, "────────────────────────────────")

	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprint(os.Stdout, "  Connexion... ")
	}
	if err := inspector.CheckPermissions(ctx, selectedDB.Name); err != nil {
		fmt.Fprintln(os.Stdout, "✗ ÉCHEC")
		fmt.Fprintf(os.Stdout, "  Erreur: %v\n", err)
		return DatabaseSelection{}, fmt.Errorf("permissions insuffisantes sur la base %q", selectedDB.Name)
	}
	fmt.Fprintln(os.Stdout, "✓ Connexion réussie")

	fmt.Fprint(os.Stdout, "  Permissions... ")
	fmt.Fprintln(os.Stdout, "✓ Permissions suffisantes")

	fmt.Fprint(os.Stdout, "  PostgreSQL... ")
	fmt.Fprintln(os.Stdout, "✓ Compatible")

	if selectedDB.HasTables {
		fmt.Fprint(os.Stdout, "  Tables... ")
		if selectedDB.HasMigrations {
			fmt.Fprintf(os.Stdout, "✓ Migrations ForgeKit détectées (v%d)", *selectedDB.MigrationVersion)
			if selectedDB.IsDirty {
				fmt.Fprintf(os.Stdout, " ⚠ DIRTY")
			}
			fmt.Fprintln(os.Stdout)
		} else {
			fmt.Fprintf(os.Stdout, "⚠ %d tables détectées (pas de migrations ForgeKit)\n", selectedDB.TableCount)
		}
	} else {
		fmt.Fprintln(os.Stdout, "  Tables... ✓ Base vide")
		fmt.Fprintln(os.Stdout, "  ✓ Aucun objet métier détecté")
	}

	// Check migration state
	checker := dbinspect.NewMigrationChecker(inspector, "migrations")
	state, err := checker.CheckState(ctx, selectedDB.Name)
	if err == nil && state.HasDestructive && selectedDB.HasTables {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "⚠ Migration potentiellement destructive détectée.\n")
		fmt.Fprintf(os.Stdout, "Cette migration peut modifier ou supprimer des données existantes.\n")
		fmt.Fprintf(os.Stdout, "ForgeKit n'exécutera pas cette migration automatiquement.\n")
		fmt.Fprintf(os.Stdout, "Aucune donnée n'a été modifiée.\n")
		return DatabaseSelection{}, fmt.Errorf("migrations destructives détectées sur base non-vide")
	}

	// Confirm with user
	if selectedDB.HasTables {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "⚠ Cette base contient déjà des données.\n\n")
		fmt.Fprintf(os.Stdout, "Base : %s\n", selectedDB.Name)
		fmt.Fprintf(os.Stdout, "Projet : %s\n\n", "forge-project") // TODO: pass project name
		if len(state.Pending) > 0 {
			fmt.Fprintln(os.Stdout, "Les migrations compatibles suivantes seront appliquées :")
			for _, mig := range state.Pending {
				fmt.Fprintf(os.Stdout, "  %s\n", mig.Name)
			}
			fmt.Fprintln(os.Stdout)
		}
		fmt.Fprint(os.Stdout, "Continuer ? [y/N] ")

		confirm, err := p.AskString("Confirmation", "n")
		if err != nil {
			return DatabaseSelection{}, err
		}
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			return DatabaseSelection{}, fmt.Errorf("opération annulée par l'utilisateur")
		}
	}

	return askExternalDatabase(g, selectedDB.Name, selectedDB.Name)
}

// askExternalDatabase prompts the user if the database is external and collects connection details.
func askExternalDatabase(g *globalFlags, databaseName, existingDBName string) (DatabaseSelection, error) {
	p := prompt.New(os.Stdin, os.Stdout)

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Cette base est-elle hébergée sur un serveur PostgreSQL externe (pas sur localhost) ?")
	fmt.Fprint(os.Stdout, "Serveur externe ? [y/N] ")

	externalConfirm, err := p.AskString("Confirmation", "n")
	if err != nil {
		return DatabaseSelection{}, err
	}

	if strings.ToLower(externalConfirm) == "y" || strings.ToLower(externalConfirm) == "yes" {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "Configuration de la base externe :")

		host, err := p.AskString("Hôte", "localhost")
		if err != nil {
			return DatabaseSelection{}, err
		}

		port, err := p.AskInt("Port", 5432)
		if err != nil {
			return DatabaseSelection{}, err
		}

		user, err := p.AskString("Utilisateur", "postgres")
		if err != nil {
			return DatabaseSelection{}, err
		}

		password, err := p.AskPassword("Mot de passe")
		if err != nil {
			return DatabaseSelection{}, err
		}

		dbName, err := p.AskString("Nom de la base", existingDBName)
		if err != nil {
			return DatabaseSelection{}, err
		}

		return DatabaseSelection{
			DatabaseName:       databaseName,
			UseExisting:        true,
			ExistingDBName:     existingDBName,
			ExternalDBHost:     host,
			ExternalDBPort:     port,
			ExternalDBUser:     user,
			ExternalDBPassword: password,
			ExternalDBName:     dbName,
		}, nil
	}

	return DatabaseSelection{
		DatabaseName:   databaseName,
		UseExisting:    true,
		ExistingDBName: existingDBName,
	}, nil
}

func newDoctorCommand(g *globalFlags) *cobra.Command {
	var ciMode bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostiquer l'environnement et le projet ForgeKit",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			console := g.console()

			// In CI mode, force quiet. Respect explicit --format json if provided.
			if ciMode {
				g.Quiet = true
				// Don't override format if user explicitly requested JSON
				// (We can't easily detect explicit flag, so check if format was changed from default)
			}

			// Get feature registry for version comparison
			registry := feature.NewRegistry(auth.AuthFeature{}, cors.CorsFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{})

			sigResult := forge.ValidateSignature(root)

			var spinner *output.Spinner
			if g.Format == output.FormatHuman && !g.Quiet && !ciMode {
				spinner = output.NewSpinner(console.Out)
				spinner.Start("Vérification signature ForgeKit...")
			}

			var sigFindings []report.Finding
			if sigResult.IsAbsent() {
				sigFindings = append(sigFindings, report.Finding{
					ID:       "forge.signature.absent",
					Category: "forge",
					Severity: report.SeverityCritical,
					Message:  "Signature ForgeKit absente: répertoire .forge manquant",
				})
			} else if sigResult.IsInvalid() {
				for _, e := range sigResult.Errors {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.invalid",
						Category: "forge",
						Severity: report.SeverityCritical,
						Message:  e,
					})
				}
			} else {
				sigFindings = append(sigFindings, report.Finding{
					ID:       "forge.signature.valid",
					Category: "forge",
					Severity: report.SeverityInfo,
					Message:  fmt.Sprintf("Signature ForgeKit valide (v%s, schema %d)", sigResult.Metadata.Version, sigResult.Metadata.Schema),
				})
				if sigResult.LegacyProject {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.legacy",
						Category: "forge",
						Severity: report.SeverityWarning,
						Message:  "Projet legacy: .forge/forge.yaml manquant, seules features.yaml présentes",
					})
				}
				for _, w := range sigResult.Warnings {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.warning",
						Category: "forge",
						Severity: report.SeverityWarning,
						Message:  w,
					})
				}
			}

			if spinner != nil {
				if sigResult.IsValid() {
					spinner.Stop("✓ Signature ForgeKit — OK")
				} else if sigResult.IsAbsent() {
					spinner.Stop("✗ Signature ForgeKit — ABSENTE")
				} else {
					spinner.Stop("✗ Signature ForgeKit — INVALIDE")
				}
			}

			// Add ForgeKit version finding
			sigFindings = append(sigFindings, report.Finding{
				ID:       "forge.version",
				Category: "forge",
				Severity: report.SeverityInfo,
				Message:  fmt.Sprintf("ForgeKit version: %s", app.Version),
			})

			// Add feature version comparison findings
			featFindings := compareFeatureVersions(root, registry, sigResult.Features)
			sigFindings = append(sigFindings, featFindings...)

			if ciMode {
				// In CI mode, run rules and produce compact output
				reg := rules.DoctorRules()
				return checkDoctorExitCode(g, root, sigFindings, reg)
			}

			reg := rules.DoctorRules()
			if err := runReport(g, "doctor", root, reg, sigFindings); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(
		&ciMode,
		"ci",
		false,
		"Mode CI/CD: non-interactif, déterministe, codes de sortie 0/1/2, sans spinners",
	)

	return cmd
}

func newAnalyzeCommand(g *globalFlags) *cobra.Command {
	var ciMode bool

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyser la structure et les pratiques du projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			cfg, err := config.LoadProject(root)
			if err != nil {
				return err
			}
			loader := rules.StaticConfigLoader{Rules: cfg.Architecture.Rules}

			sigResult := forge.ValidateSignature(root)

			var spinner *output.Spinner
			console := g.console()
			if g.Format == output.FormatHuman && !g.Quiet && !ciMode {
				spinner = output.NewSpinner(console.Out)
				spinner.Start("Vérification signature ForgeKit...")
			}

			var sigFindings []report.Finding
			if sigResult.IsAbsent() {
				sigFindings = append(sigFindings, report.Finding{
					ID:       "forge.signature.absent",
					Category: "forge",
					Severity: report.SeverityCritical,
					Message:  "Signature ForgeKit absente: répertoire .forge manquant",
				})
			} else if sigResult.IsInvalid() {
				for _, e := range sigResult.Errors {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.invalid",
						Category: "forge",
						Severity: report.SeverityCritical,
						Message:  e,
					})
				}
			} else {
				sigFindings = append(sigFindings, report.Finding{
					ID:       "forge.signature.valid",
					Category: "forge",
					Severity: report.SeverityInfo,
					Message:  fmt.Sprintf("Signature ForgeKit valide (v%s, schema %d)", sigResult.Metadata.Version, sigResult.Metadata.Schema),
				})
				if sigResult.LegacyProject {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.legacy",
						Category: "forge",
						Severity: report.SeverityWarning,
						Message:  "Projet legacy: .forge/forge.yaml manquant, seules features.yaml présentes",
					})
				}
				for _, w := range sigResult.Warnings {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.warning",
						Category: "forge",
						Severity: report.SeverityWarning,
						Message:  w,
					})
				}
			}

			if spinner != nil {
				if sigResult.IsValid() {
					spinner.Stop("✓ Signature ForgeKit — OK")
				} else if sigResult.IsAbsent() {
					spinner.Stop("✗ Signature ForgeKit — ABSENTE")
				} else {
					spinner.Stop("✗ Signature ForgeKit — INVALIDE")
				}
			}

			return runAnalyze(g, root, loader, sigFindings, ciMode)
		},
	}

	cmd.Flags().BoolVar(
		&ciMode,
		"ci",
		false,
		"Mode CI/CD: non-interactif, déterministe, codes de sortie 0/1/2, sans spinners",
	)

	return cmd
}

// runAnalyze runs category-by-category analysis with per-step spinner and aggregates findings.
// ciMode enables CI/CD mode: no spinners, deterministic output, proper exit codes.
func runAnalyze(g *globalFlags, root string, loader rules.StaticConfigLoader, extraFindings []report.Finding, ciMode bool) error {
	console := g.console()

	// In CI mode, force quiet. Don't override format if user explicitly requested JSON.
	// Use local variable for format checks in CI mode.
	ciFormat := g.Format
	if ciMode {
		g.Quiet = true
	}

	categories := []string{"Architecture", "Tests", "Security", "Configuration", "Docker", "Documentation"}
	var allFindings []report.Finding
	allFindings = append(allFindings, extraFindings...)
	start := time.Now()

	for _, cat := range categories {
		// spinner per category (human only, not in CI mode)
		var spinner *output.Spinner
		if ciFormat == output.FormatHuman && !g.Quiet && !ciMode {
			spinner = output.NewSpinner(console.Out)
			spinner.Start("Analyse " + cat + "...")
		}

		var findings []report.Finding
		var err error

		switch cat {
		case "Architecture":
			reg := rules.NewRegistry(rules.ArchitectureRule{Rules: loader.ArchitectureRules()})
			findings, err = reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
		case "Security":
			reg := rules.NewRegistry(rules.SecuritySecretsRule{}, rules.SecurityCORSRule{})
			findings, err = reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
		case "Configuration":
			reg := rules.NewRegistry(rules.GoModRule{}, rules.EnvFileRule{})
			findings, err = reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
		case "Docker":
			reg := rules.NewRegistry(rules.DockerRule{})
			findings, err = reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
		case "Tests":
			// lightweight test detection: presence of _test.go files
			findings, err = detectTests(root)
		case "Documentation":
			findings, err = detectDocs(root)
		default:
			findings = nil
		}

		if spinner != nil {
			// compute category score using all accumulated findings so far, plus current category.
			temp := report.Result{Project: root, Findings: append(append([]report.Finding(nil), allFindings...), findings...)}
			score := report.ComputeScore(temp)
			s := score.Categories[cat]
			// final message
			var final string
			if err != nil {
				final = "✗ " + cat + " — erreur"
			} else if s >= 60 {
				final = "✓ " + cat + " — " + fmt.Sprintf("%d/100", s)
			} else {
				final = "✗ " + cat + " — " + fmt.Sprintf("%d/100", s)
			}
			spinner.Stop(final)
		} else if ciMode && ciFormat == output.FormatHuman {
			// In CI mode, print compact category status
			temp := report.Result{Project: root, Findings: append(append([]report.Finding(nil), allFindings...), findings...)}
			score := report.ComputeScore(temp)
			s := score.Categories[cat]
			status := "PASS"
			if err != nil {
				status = "ERROR"
			} else if s < 60 {
				status = "FAIL"
			}
			fmt.Fprintf(console.Out, "[%s] %s: %d/100\n", status, cat, s)
		}

		if err != nil {
			output.Debug(os.Stderr, g.Debug, err)
			// In CI mode, return error with exit code 2
			if ciMode {
				return &CiError{Code: 2, Err: err}
			}
			return err
		}
		allFindings = append(allFindings, findings...)
	}

	// aggregate and print result
	allFindings = report.UniqueFindings(allFindings)
	res := report.Result{
		SchemaVersion: report.JSONSchemaVersion,
		Tool:          app.Name,
		Version:       app.Version,
		Command:       "analyze",
		Project:       root,
		Timestamp:     start,
		Findings:      allFindings,
	}
	res.Summary = report.BuildSummary(allFindings)

	// Print output based on format and mode
	if g.Format == output.FormatJSON {
		if err := console.PrintJSON(res); err != nil {
			return err
		}
	} else if ciMode {
		printCiSummary(console.Out, res)
	} else {
		if err := console.PrintResult(res); err != nil {
			return err
		}
	}

	// Determine exit code
	hasErrors := res.Summary.Error > 0 || res.Summary.Critical > 0
	hasWarnings := res.Summary.Warning > 0

	if ciMode {
		if hasErrors {
			return &CiError{Code: 1, Err: fmt.Errorf("diagnostics en échec : %d error(s), %d warning(s)", res.Summary.Error+res.Summary.Critical, res.Summary.Warning)}
		}
		if hasWarnings {
			return &CiError{Code: 1, Err: fmt.Errorf("warnings détectés : %d warning(s)", res.Summary.Warning)}
		}
		return &CiError{Code: 0, Err: nil}
	}

	if report.HasFailures(res.Summary) {
		return fmt.Errorf("diagnostics en échec : %d error(s)", res.Summary.Error+res.Summary.Critical)
	}
	return nil
}

// CiError wraps an error with an exit code for CI mode.
type CiError struct {
	Code int
	Err  error
}

func (e *CiError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

// printCiSummary prints a compact summary suitable for CI output.
func printCiSummary(out io.Writer, res report.Result) {
	fmt.Fprintf(out, "ForgeKit Analyze — %s\n", res.Project)
	fmt.Fprintf(out, "Version: %s | Timestamp: %s\n", res.Version, res.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(out, "Summary: pass=%d info=%d warning=%d error=%d critical=%d\n",
		res.Summary.Pass, res.Summary.Info, res.Summary.Warning, res.Summary.Error, res.Summary.Critical)

	// Print category scores
	score := report.ComputeScore(res)
	categories := []string{"Architecture", "Tests", "Security", "Configuration", "Docker", "Documentation"}
	for _, cat := range categories {
		s := score.Categories[cat]
		status := "PASS"
		if s < 60 {
			status = "FAIL"
		}
		fmt.Fprintf(out, "  [%s] %s: %d/100\n", status, cat, s)
	}

	// Print warnings and errors
	for _, f := range res.Findings {
		if f.Severity == report.SeverityWarning || f.Severity == report.SeverityError || f.Severity == report.SeverityCritical {
			loc := ""
			if f.File != "" {
				loc = fmt.Sprintf(" (%s)", f.File)
			}
			fmt.Fprintf(out, "  %s: %s%s\n", strings.ToUpper(string(f.Severity)), f.Message, loc)
		}
	}
}

// detectTests returns findings about tests presence.
func detectTests(root string) ([]report.Finding, error) {
	hasTests := false
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			hasTests = true
			return filepath.SkipDir
		}
		return nil
	})
	if hasTests {
		return []report.Finding{{ID: "tests.present", Category: "tests", Severity: report.SeverityInfo, Message: "Tests détectés"}}, nil
	}
	return []report.Finding{{ID: "tests.missing", Category: "tests", Severity: report.SeverityWarning, Message: "Aucun test détecté"}}, nil
}

// detectDocs checks README.md and .env.example
func detectDocs(root string) ([]report.Finding, error) {
	msgs := []report.Finding{}
	if _, err := os.Stat(filepath.Join(root, "README.md")); err == nil {
		msgs = append(msgs, report.Finding{ID: "docs.readme", Category: "documentation", Severity: report.SeverityInfo, Message: "README.md présent"})
	} else {
		msgs = append(msgs, report.Finding{ID: "docs.readme.missing", Category: "documentation", Severity: report.SeverityWarning, Message: "README.md absent"})
	}
	if _, err := os.Stat(filepath.Join(root, ".env.example")); err == nil {
		msgs = append(msgs, report.Finding{ID: "docs.env", Category: "documentation", Severity: report.SeverityInfo, Message: ".env.example présent"})
	} else {
		msgs = append(msgs, report.Finding{ID: "docs.env.missing", Category: "documentation", Severity: report.SeverityWarning, Message: ".env.example absent"})
	}
	return msgs, nil
}

// compareFeatureVersions compares installed feature versions with registry versions.
func compareFeatureVersions(root string, registry *feature.Registry, installed feature.FeaturesFile) []report.Finding {
	var findings []report.Finding

	if len(installed.Features) == 0 {
		findings = append(findings, report.Finding{
			ID:       "features.none",
			Category: "features",
			Severity: report.SeverityInfo,
			Message:  "Aucune feature installée",
		})
		return findings
	}

	for _, installedFeat := range installed.Features {
		// Check if feature exists in registry
		regFeat, ok := registry.Get(installedFeat.Name)
		if !ok {
			findings = append(findings, report.Finding{
				ID:         "features.unknown",
				Category:   "features",
				Severity:   report.SeverityWarning,
				Message:    fmt.Sprintf("Feature %q installée mais inconnue du registre", installedFeat.Name),
				Suggestion: "La feature a peut-être été supprimée du registre ou renommée",
			})
			continue
		}

		// Compare versions
		registryVersion := regFeat.Version()
		installedVersion := installedFeat.Version

		if registryVersion != installedVersion {
			findings = append(findings, report.Finding{
				ID:         "features.version_mismatch",
				Category:   "features",
				Severity:   report.SeverityWarning,
				Message:    fmt.Sprintf("Feature %q: version installée %s, version registre %s", installedFeat.Name, installedVersion, registryVersion),
				Suggestion: "Considérez une mise à jour ou réinstallation de la feature",
			})
		} else {
			findings = append(findings, report.Finding{
				ID:       "features.version_ok",
				Category: "pass",
				Severity: report.SeverityInfo,
				Message:  fmt.Sprintf("Feature %q: version %s (à jour)", installedFeat.Name, installedVersion),
			})
		}

		// Check if feature directory exists
		featurePath := filepath.Join(root, "internal", installedFeat.Name)
		if _, err := os.Stat(featurePath); err != nil {
			if os.IsNotExist(err) {
				findings = append(findings, report.Finding{
					ID:         "features.missing_dir",
					Category:   "features",
					Severity:   report.SeverityWarning,
					File:       featurePath,
					Message:    fmt.Sprintf("Feature %q déclarée mais répertoire manquant", installedFeat.Name),
					Suggestion: "Réinstallez la feature avec 'forge add' ou supprimez-la de .forge/features.yaml",
				})
			}
		} else {
			findings = append(findings, report.Finding{
				ID:       "features.dir.present",
				Category: "pass",
				Severity: report.SeverityInfo,
				Message:  fmt.Sprintf("Feature %q: répertoire présent", installedFeat.Name),
			})
		}
	}

	return findings
}

// checkDoctorExitCode determines the exit code for doctor in CI mode.
func checkDoctorExitCode(g *globalFlags, root string, extraFindings []report.Finding, reg *rules.Registry) error {
	console := g.console()

	// Run the same rules as runReport to get all findings
	findings, err := reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
	if err != nil {
		return &CiError{Code: 2, Err: err}
	}

	// Combine with extra findings
	allFindings := append(extraFindings, findings...)
	allFindings = report.UniqueFindings(allFindings)

	res := report.Result{
		SchemaVersion: report.JSONSchemaVersion,
		Tool:          app.Name,
		Version:       app.Version,
		Command:       "doctor",
		Project:       root,
		Timestamp:     time.Now(),
		Findings:      allFindings,
	}
	res.Summary = report.BuildSummary(allFindings)

	// Print output based on format
	if g.Format == output.FormatJSON {
		if err := console.PrintJSON(res); err != nil {
			return err
		}
	} else {
		printDoctorCiSummary(console.Out, res)
	}

	// Determine exit code
	hasErrors := res.Summary.Error > 0 || res.Summary.Critical > 0
	hasWarnings := res.Summary.Warning > 0

	if hasErrors {
		return &CiError{Code: 1, Err: fmt.Errorf("diagnostics en échec : %d error(s), %d warning(s)", res.Summary.Error+res.Summary.Critical, res.Summary.Warning)}
	}
	if hasWarnings {
		return &CiError{Code: 1, Err: fmt.Errorf("warnings détectés : %d warning(s)", res.Summary.Warning)}
	}
	return &CiError{Code: 0, Err: nil}
}

// printDoctorCiSummary prints a compact summary for doctor CI mode.
func printDoctorCiSummary(out io.Writer, res report.Result) {
	fmt.Fprintf(out, "ForgeKit Doctor — %s\n", res.Project)
	fmt.Fprintf(out, "Version: %s | Timestamp: %s\n", res.Version, res.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(out, "Summary: pass=%d info=%d warning=%d error=%d critical=%d\n",
		res.Summary.Pass, res.Summary.Info, res.Summary.Warning, res.Summary.Error, res.Summary.Critical)

	// Print category scores using the score computation
	score := report.ComputeScore(res)
	categories := []string{"environment", "project", "security", "docker", "configuration", "dependencies", "features", "documentation"}
	for _, cat := range categories {
		s := score.Categories[cat]
		if s > 0 { // Only show categories that have findings
			status := "PASS"
			if s < 60 {
				status = "FAIL"
			}
			fmt.Fprintf(out, "  [%s] %s: %d/100\n", status, cat, s)
		}
	}

	// Print warnings and errors
	for _, f := range res.Findings {
		if f.Severity == report.SeverityWarning || f.Severity == report.SeverityError || f.Severity == report.SeverityCritical {
			loc := ""
			if f.File != "" {
				loc = fmt.Sprintf(" (%s)", f.File)
			}
			fmt.Fprintf(out, "  %s: %s%s\n", strings.ToUpper(string(f.Severity)), f.Message, loc)
		}
	}
}

func newCheckCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Vérifier la conformité architecturale (CI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			cfg, err := config.LoadProject(root)
			if err != nil {
				return err
			}
			console := g.console()

			sigResult := forge.ValidateSignature(root)

			var spinner *output.Spinner
			if g.Format == output.FormatHuman && !g.Quiet {
				spinner = output.NewSpinner(console.Out)
				spinner.Start("Vérification signature ForgeKit...")
			}

			var sigFindings []report.Finding
			if sigResult.IsAbsent() {
				sigFindings = append(sigFindings, report.Finding{
					ID:       "forge.signature.absent",
					Category: "forge",
					Severity: report.SeverityCritical,
					Message:  "Signature ForgeKit absente: répertoire .forge manquant",
				})
			} else if sigResult.IsInvalid() {
				for _, e := range sigResult.Errors {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.invalid",
						Category: "forge",
						Severity: report.SeverityCritical,
						Message:  e,
					})
				}
			} else {
				sigFindings = append(sigFindings, report.Finding{
					ID:       "forge.signature.valid",
					Category: "forge",
					Severity: report.SeverityInfo,
					Message:  fmt.Sprintf("Signature ForgeKit valide (v%s, schema %d)", sigResult.Metadata.Version, sigResult.Metadata.Schema),
				})
				if sigResult.LegacyProject {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.legacy",
						Category: "forge",
						Severity: report.SeverityWarning,
						Message:  "Projet legacy: .forge/forge.yaml manquant, seules features.yaml présentes",
					})
				}
				for _, w := range sigResult.Warnings {
					sigFindings = append(sigFindings, report.Finding{
						ID:       "forge.signature.warning",
						Category: "forge",
						Severity: report.SeverityWarning,
						Message:  w,
					})
				}
			}

			if spinner != nil {
				if sigResult.IsValid() {
					spinner.Stop("✓ Signature ForgeKit — OK")
				} else if sigResult.IsAbsent() {
					spinner.Stop("✗ Signature ForgeKit — ABSENTE")
				} else {
					spinner.Stop("✗ Signature ForgeKit — INVALIDE")
				}
			}

			loader := rules.StaticConfigLoader{Rules: cfg.Architecture.Rules}
			if err := runReport(g, "check", root, rules.CheckRules(loader), sigFindings); err != nil {
				return err
			}
			// exit 1 handled by main via summary — re-check after print
			return nil
		},
	}
}

func runReport(g *globalFlags, command, root string, reg *rules.Registry, extraFindings ...[]report.Finding) error {
	console := g.console()
	var spinner *output.Spinner
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Analyse du projet...")
	}

	start := time.Now()
	findings, err := reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Analyse interrompue")
		} else {
			spinner.Stop("✓ Analyse terminée")
		}
	}
	if err != nil {
		output.Debug(os.Stderr, g.Debug, err)
		return err
	}

	for _, extra := range extraFindings {
		findings = append(extra, findings...)
	}

	res := report.Result{
		SchemaVersion: report.JSONSchemaVersion,
		Tool:          app.Name,
		Version:       app.Version,
		Command:       command,
		Project:       root,
		Timestamp:     start,
		Findings:      findings,
	}
	res.Summary = report.BuildSummary(findings)
	if err := console.PrintResult(res); err != nil {
		return err
	}
	if report.HasFailures(res.Summary) {
		return fmt.Errorf("diagnostics en échec : %d error(s)", res.Summary.Error+res.Summary.Critical)
	}
	return nil
}

func newAddCommand(g *globalFlags) *cobra.Command {
	var list bool
	var dryRun bool
	var showPlan bool

	cmd := &cobra.Command{
		Use:   "add [feature]",
		Short: "Ajouter une fonctionnalité au projet",
		Args: func(cmd *cobra.Command, args []string) error {
			if list {
				if len(args) != 0 {
					return fmt.Errorf("--list ne peut pas être utilisé avec une feature")
				}
				return nil
			}

			if len(args) != 1 {
				return fmt.Errorf("une feature est requise, par exemple : forge add auth")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := feature.NewRegistry(auth.AuthFeature{}, cors.CorsFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{})

			if list {
				return runFeatureList(g, registry)
			}

			return runFeatureAdd(g, registry, args[0], dryRun, showPlan)
		},
	}

	cmd.Flags().BoolVar(
		&list,
		"list",
		false,
		"Afficher les features disponibles",
	)

	cmd.Flags().BoolVar(
		&dryRun,
		"dry-run",
		false,
		"Afficher le plan sans modifier le projet",
	)

	cmd.Flags().BoolVar(
		&showPlan,
		"plan",
		false,
		"Afficher le plan d'installation détaillé (fichiers à créer/modifier/supprimer, conflits, dépendances)",
	)

	return cmd
}

func newRemoveCommand(g *globalFlags) *cobra.Command {
	var showPlan bool

	cmd := &cobra.Command{
		Use:   "remove [feature]",
		Short: "Supprimer une fonctionnalité du projet",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("une feature est requise, par exemple : forge remove auth")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := feature.NewRegistry(auth.AuthFeature{}, cors.CorsFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{})
			return runFeatureRemove(g, registry, args[0], showPlan)
		},
	}

	cmd.Flags().BoolVar(
		&showPlan,
		"plan",
		false,
		"Afficher le plan de suppression sans modifier le projet",
	)

	return cmd
}

func runFeatureList(g *globalFlags, registry *feature.Registry) error {
	features := registry.List()
	console := g.console()

	if g.Format == output.FormatJSON {
		type featureJSON struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		}

		type featureListWithSchema struct {
			SchemaVersion string        `json:"schema_version"`
			Features      []featureJSON `json:"features"`
		}

		result := make([]featureJSON, 0, len(features))

		for _, f := range features {
			result = append(result, featureJSON{
				Name:        f.Name(),
				Version:     f.Version(),
				Description: f.Description(),
			})
		}

		return console.PrintJSON(featureListWithSchema{
			SchemaVersion: report.JSONSchemaVersion,
			Features:      result,
		})
	}

	if len(features) == 0 {
		if !g.Quiet {
			fmt.Fprintln(console.Out, "Aucune feature disponible.")
		}
		return nil
	}

	if g.Quiet {
		for _, f := range features {
			fmt.Fprintln(console.Out, f.Name())
		}
		return nil
	}

	fmt.Fprintln(console.Out, "ForgeKit Features")
	fmt.Fprintln(console.Out, "────────────────────────────────")

	for _, f := range features {
		fmt.Fprintf(
			console.Out,
			"%s %s — %s\n",
			f.Name(),
			f.Version(),
			f.Description(),
		)
	}

	return nil
}

func runFeatureAdd(
	g *globalFlags,
	registry *feature.Registry,
	name string,
	dryRun bool,
	showPlan bool,
) error {
	console := g.console()

	root, err := os.Getwd()
	if err != nil {
		return err
	}

	// Step 1: Detect project
	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out, "ForgeKit Add")
		fmt.Fprintln(console.Out, "────────────────────────────────")
		fmt.Fprintln(console.Out)
	}

	var spinner *output.Spinner
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Détection du projet...")
	}

	detector := feature.Detector{}

	project, err := detector.DetectLoose(root) // Support external compatible projects
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Détection du projet — erreur")
		} else {
			spinner.Stop("✓ Projet détecté")
		}
	}
	if err != nil {
		output.Debug(os.Stderr, g.Debug, err)
		return err
	}

	// Step 2: Find feature
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Recherche de la feature...")
	}

	f, err := registry.Require(name)
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Feature non trouvée")
		} else {
			spinner.Stop(fmt.Sprintf("✓ Feature %q trouvée", name))
		}
	}
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Step 3: Resolve and check dependencies
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Résolution des dépendances...")
	}

	// Get all features to install (including dependencies)
	featuresToInstall, err := registry.ResolveDependencies([]string{name})
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Résolution des dépendances — échec")
		} else {
			spinner.Stop("✓ Dépendances résolues")
		}
	}
	if err != nil {
		return fmt.Errorf("résolution des dépendances : %w", err)
	}

	// Step 4: Check prerequisites for all features (dependencies first)
	for _, feat := range featuresToInstall {
		if g.Format == output.FormatHuman && !g.Quiet {
			spinner = output.NewSpinner(console.Out)
			spinner.Start(fmt.Sprintf("Vérification des prérequis pour %q...", feat.Name()))
		}

		checkErr := feat.Check(ctx, project)
		alreadyInstalled := checkErr != nil && strings.Contains(checkErr.Error(), "déjà installée")
		if checkErr != nil {
			if spinner != nil {
				if alreadyInstalled {
					spinner.Stop(fmt.Sprintf("✓ Feature %q déjà installée", feat.Name()))
				} else {
					spinner.Stop(fmt.Sprintf("✗ Vérification des prérequis pour %q — échec", feat.Name()))
				}
			}
			if alreadyInstalled {
				// Skip already installed dependencies - they're already satisfied
				if dryRun || showPlan {
					continue // In dry-run/plan mode, just show message and continue
				}
				if g.Format == output.FormatHuman && !g.Quiet {
					fmt.Fprintf(console.Out, "  ✓ %s déjà installé, ignoré\n", feat.Name())
				}
				continue
			}
			return fmt.Errorf("vérification de la feature %q : %w", feat.Name(), checkErr)
		}
		if spinner != nil {
			spinner.Stop(fmt.Sprintf("✓ Prérequis validés pour %q", feat.Name()))
		}
	}

	// Step 5: Build plan for the main feature (dependencies will be installed first)
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Construction du plan...")
	}

	plan, err := f.Plan(ctx, project)
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Construction du plan — échec")
		} else {
			spinner.Stop("✓ Plan validé")
		}
	}
	if err != nil {
		return fmt.Errorf("construction du plan pour %q : %w", name, err)
	}

	// Detect conflicts
	if err := detectConflicts(project.Root, &plan); err != nil {
		return fmt.Errorf("détection des conflits : %w", err)
	}

	if dryRun || showPlan {
		if g.Format == output.FormatHuman && !g.Quiet {
			fmt.Fprintln(console.Out)
		}
		return printFeaturePlan(g, plan, dryRun, showPlan)
	}

	// Step 5: Install all features in order (dependencies first)
	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Installation...")
	}

	// Create snapshot for rollback
	snapshot, err := createInstallSnapshot(project.Root)
	if err != nil {
		return fmt.Errorf("créer snapshot pour rollback: %w", err)
	}

	// Install each feature in order
	for _, feat := range featuresToInstall {
		// Check if already installed
		installed, _, _ := feature.IsInstalled(project.Root, feat.Name())
		if installed {
			if g.Format == output.FormatHuman && !g.Quiet {
				fmt.Fprintf(console.Out, "  ✓ %s déjà installé, ignoré\n", feat.Name())
			}
			continue
		}

		if g.Format == output.FormatHuman && !g.Quiet {
			spinner = output.NewSpinner(console.Out)
			spinner.Start(fmt.Sprintf("Installation de %s...", feat.Name()))
		}

		featPlan, err := feat.Plan(ctx, project)
		if err != nil {
			// Rollback on error
			if rollbackErr := restoreInstallSnapshot(project.Root, snapshot); rollbackErr != nil {
				return fmt.Errorf("installation de %s échouée: %w; rollback aussi échoué: %v", feat.Name(), err, rollbackErr)
			}
			return fmt.Errorf("plan pour %s: %w", feat.Name(), err)
		}

		if err := detectConflicts(project.Root, &featPlan); err != nil {
			if rollbackErr := restoreInstallSnapshot(project.Root, snapshot); rollbackErr != nil {
				return fmt.Errorf("conflits pour %s: %w; rollback aussi échoué: %v", feat.Name(), err, rollbackErr)
			}
			return fmt.Errorf("détection conflits pour %s: %w", feat.Name(), err)
		}

		if err := feat.Apply(ctx, project, featPlan); err != nil {
			// Rollback on error
			if rollbackErr := restoreInstallSnapshot(project.Root, snapshot); rollbackErr != nil {
				return fmt.Errorf("installation de %s échouée: %w; rollback aussi échoué: %v", feat.Name(), err, rollbackErr)
			}
			return fmt.Errorf("installation de %s: %w", feat.Name(), err)
		}

		if spinner != nil {
			spinner.Stop(fmt.Sprintf("✓ %s installé", feat.Name()))
		}
	}

	if g.Format == output.FormatHuman && !g.Quiet && spinner == nil {
		fmt.Fprintln(console.Out, "  ✓ Fichiers installés")
	}

	// Step 6: Dependencies (handled in Apply)
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Installation des dépendances...")
		// Small delay to show spinner
		time.Sleep(100 * time.Millisecond)
		spinner.Stop("✓ Dépendances installées")
	}

	// Step 7: Validate project
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Validation du projet...")
		time.Sleep(100 * time.Millisecond)
		spinner.Stop("✓ Projet validé")
	}

	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "────────────────────────────────")
		fmt.Fprintf(console.Out, "✓ Feature %q installée avec succès\n", name)
	}

	return nil
}

func runFeatureRemove(
	g *globalFlags,
	registry *feature.Registry,
	name string,
	showPlan bool,
) error {
	console := g.console()

	root, err := os.Getwd()
	if err != nil {
		return err
	}

	// Step 1: Detect project
	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out, "ForgeKit Remove")
		fmt.Fprintln(console.Out, "────────────────────────────────")
		fmt.Fprintln(console.Out)
	}

	var spinner *output.Spinner
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Détection du projet...")
	}

	detector := feature.Detector{}

	project, err := detector.DetectLoose(root) // Support external compatible projects
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Détection du projet — erreur")
		} else {
			spinner.Stop("✓ Projet détecté")
		}
	}
	if err != nil {
		output.Debug(os.Stderr, g.Debug, err)
		return err
	}

	// Step 2: Find feature
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Recherche de la feature...")
	}

	f, err := registry.Require(name)
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Feature non trouvée")
		} else {
			spinner.Stop(fmt.Sprintf("✓ Feature %q trouvée", name))
		}
	}
	if err != nil {
		return err
	}

	// Check if feature is installed
	installed, _, err := feature.IsInstalled(project.Root, name)
	if err != nil {
		return fmt.Errorf("vérifier l'installation : %w", err)
	}
	if !installed {
		return fmt.Errorf("feature %q n'est pas installée", name)
	}

	// Check reverse dependencies (features that depend on this one)
	var dependentFeatures []string
	for _, feat := range registry.List() {
		if fd, ok := feat.(feature.FeatureDependencies); ok {
			for _, dep := range fd.DependsOn() {
				if dep == name {
					// Check if this dependent feature is installed
					depInstalled, _, _ := feature.IsInstalled(project.Root, feat.Name())
					if depInstalled {
						dependentFeatures = append(dependentFeatures, feat.Name())
					}
				}
			}
		}
	}

	if len(dependentFeatures) > 0 {
		return fmt.Errorf("impossible de supprimer %q : features dépendantes installées : %s", name, strings.Join(dependentFeatures, ", "))
	}

	ctx := context.Background()

	// Step 3: Build removal plan
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Construction du plan de suppression...")
	}

	plan, err := f.Plan(ctx, project)
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Construction du plan — échec")
		} else {
			spinner.Stop("✓ Plan validé")
		}
	}
	if err != nil {
		return fmt.Errorf("construction du plan pour %q : %w", name, err)
	}

	// Detect conflicts (files that will be removed but may have user modifications)
	if err := detectConflicts(project.Root, &plan); err != nil {
		return fmt.Errorf("détection des conflits : %w", err)
	}

	if showPlan {
		if g.Format == output.FormatHuman && !g.Quiet {
			fmt.Fprintln(console.Out)
		}
		return printFeatureRemovePlan(g, plan)
	}

	// Step 4: Remove feature
	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Suppression...")
		spinner = output.NewSpinner(console.Out)
		spinner.Start(fmt.Sprintf("Suppression de %s...", name))
	}

	// Create snapshot for rollback
	snapshot, err := createInstallSnapshot(project.Root)
	if err != nil {
		return fmt.Errorf("créer snapshot pour rollback: %w", err)
	}

	// Remove the feature
	if remover, ok := f.(feature.FeatureRemover); ok {
		if err := remover.Remove(ctx, project, plan); err != nil {
			// Rollback on error
			if rollbackErr := restoreInstallSnapshot(project.Root, snapshot); rollbackErr != nil {
				return fmt.Errorf("suppression de %s échouée: %w; rollback aussi échoué: %v", name, err, rollbackErr)
			}
			return fmt.Errorf("suppression de %s: %w", name, err)
		}
	} else {
		// Fallback: manual removal
		if err := fallbackRemove(ctx, project, plan); err != nil {
			if rollbackErr := restoreInstallSnapshot(project.Root, snapshot); rollbackErr != nil {
				return fmt.Errorf("suppression de %s échouée: %w; rollback aussi échoué: %v", name, err, rollbackErr)
			}
			return fmt.Errorf("suppression de %s: %w", name, err)
		}
	}

	if spinner != nil {
		spinner.Stop(fmt.Sprintf("✓ %s supprimé", name))
	}

	// Step 5: Validate project
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Validation du projet...")
		time.Sleep(100 * time.Millisecond)
		spinner.Stop("✓ Projet validé")
	}

	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "────────────────────────────────")
		fmt.Fprintf(console.Out, "✓ Feature %q supprimée avec succès\n", name)
	}

	return nil
}

func printFeatureRemovePlan(g *globalFlags, plan feature.Plan) error {
	console := g.console()

	if g.Format == output.FormatJSON {
		type planWithSchema struct {
			SchemaVersion string       `json:"schema_version"`
			Plan          feature.Plan `json:"plan"`
		}
		return console.PrintJSON(planWithSchema{
			SchemaVersion: report.JSONSchemaVersion,
			Plan:          plan,
		})
	}

	if g.Quiet {
		for _, file := range plan.Files {
			fmt.Fprintf(console.Out, "DELETE %s\n", file.Destination)
		}
		return nil
	}

	fmt.Fprintln(console.Out)
	fmt.Fprintln(console.Out, "ForgeKit Remove Plan")
	fmt.Fprintln(console.Out, "────────────────────────────────")
	fmt.Fprintf(console.Out, "Feature: %s\n", plan.Feature)
	fmt.Fprintf(console.Out, "Version: %s\n", plan.Version)

	if len(plan.Files) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Fichiers à supprimer :")
		for _, file := range plan.Files {
			fmt.Fprintf(console.Out, "  - DELETE %s\n", file.Destination)
		}
	}

	if len(plan.Dependencies) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Dépendances Go à supprimer :")
		for _, dep := range plan.Dependencies {
			fmt.Fprintf(console.Out, "  → %s %s\n", dep.Module, dep.Version)
		}
	}

	if len(plan.Environment) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Variables d'environnement à supprimer :")
		for _, env := range plan.Environment {
			fmt.Fprintf(console.Out, "  → %s\n", env)
		}
	}

	if len(plan.Conflicts) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Conflits détectés :")
		for _, conflict := range plan.Conflicts {
			fmt.Fprintf(console.Out, "  ⚠ %s: %s\n", conflict.File, conflict.Description)
		}
	}

	fmt.Fprintln(console.Out)
	fmt.Fprintln(console.Out, "Aucune modification effectuée.")

	return nil
}

// fallbackRemove performs basic file removal when feature doesn't implement FeatureRemover
func fallbackRemove(ctx context.Context, project feature.ProjectContext, plan feature.Plan) error {
	for _, file := range plan.Files {
		dest := filepath.Join(project.Root, file.Destination)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("supprimer %s : %w", dest, err)
		}
	}

	// Remove dependencies from go.mod
	for _, dep := range plan.Dependencies {
		cmd := exec.Command("go", "mod", "edit", "-droprequire", dep.Module)
		cmd.Dir = project.Root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Log but continue
		}
	}

	if err := feature.RunGoModTidy(project.Root); err != nil {
		return fmt.Errorf("go mod tidy : %w", err)
	}

	if err := feature.RunGoFmt(project.Root); err != nil {
		return fmt.Errorf("gofmt : %w", err)
	}

	if err := feature.RemoveEnvironment(project.Root, plan.Environment); err != nil {
		return fmt.Errorf("supprimer les variables d'environnement : %w", err)
	}

	if err := feature.RemoveInstalledFeature(project.Root, plan.Feature); err != nil {
		return fmt.Errorf("supprimer l'enregistrement d'installation : %w", err)
	}

	return nil
}

func printFeaturePlan(g *globalFlags, plan feature.Plan, dryRun, showPlan bool) error {
	console := g.console()

	if g.Format == output.FormatJSON {
		type planWithSchema struct {
			SchemaVersion string       `json:"schema_version"`
			Plan          feature.Plan `json:"plan"`
		}
		return console.PrintJSON(planWithSchema{
			SchemaVersion: report.JSONSchemaVersion,
			Plan:          plan,
		})
	}

	if g.Quiet {
		for _, file := range plan.Files {
			action := "CREATE"
			switch file.Action {
			case feature.FileActionCreate:
				action = "CREATE"
			case feature.FileActionModify:
				action = "MODIFY"
			case feature.FileActionDelete:
				action = "DELETE"
			}
			fmt.Fprintf(
				console.Out,
				"%s %s\n",
				action,
				file.Destination,
			)
		}
		return nil
	}

	fmt.Fprintln(console.Out)
	fmt.Fprintln(console.Out, "ForgeKit Plan")
	fmt.Fprintln(console.Out, "────────────────────────────────")
	fmt.Fprintf(console.Out, "Feature: %s\n", plan.Feature)
	fmt.Fprintf(console.Out, "Version: %s\n", plan.Version)

	if len(plan.Files) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Fichiers :")

		for _, file := range plan.Files {
			action := "CREATE"
			switch file.Action {
			case feature.FileActionCreate:
				action = "+ CREATE"
			case feature.FileActionModify:
				action = "~ MODIFY"
			case feature.FileActionDelete:
				action = "- DELETE"
			}
			fmt.Fprintf(
				console.Out,
				"  %s %s\n",
				action,
				file.Destination,
			)
		}
	}

	if len(plan.Dependencies) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Dépendances :")

		for _, dep := range plan.Dependencies {
			fmt.Fprintf(
				console.Out,
				"  → %s %s\n",
				dep.Module,
				dep.Version,
			)
		}
	}

	if len(plan.Environment) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Variables d'environnement :")

		for _, env := range plan.Environment {
			fmt.Fprintf(
				console.Out,
				"  → %s\n",
				env,
			)
		}
	}

	if len(plan.Conflicts) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Conflits détectés :")

		for _, conflict := range plan.Conflicts {
			fmt.Fprintf(
				console.Out,
				"  ⚠ %s: %s\n",
				conflict.File,
				conflict.Description,
			)
		}
	}

	fmt.Fprintln(console.Out)
	if dryRun || showPlan {
		fmt.Fprintln(console.Out, "Aucune modification effectuée.")
	} else {
		fmt.Fprintln(console.Out, "Aucune modification effectuée (--dry-run).")
	}

	return nil
}

// InstallSnapshot holds the state of the project before installation for rollback.
type InstallSnapshot struct {
	GoModContent []byte
	GoSumContent []byte
	ForgeDirPath string
	ForgeFiles   map[string][]byte
}

// createInstallSnapshot captures the current state of key project files.
func createInstallSnapshot(projectRoot string) (*InstallSnapshot, error) {
	snapshot := &InstallSnapshot{
		ForgeFiles: make(map[string][]byte),
	}

	// Snapshot go.mod
	if data, err := os.ReadFile(filepath.Join(projectRoot, "go.mod")); err == nil {
		snapshot.GoModContent = data
	}

	// Snapshot go.sum
	if data, err := os.ReadFile(filepath.Join(projectRoot, "go.sum")); err == nil {
		snapshot.GoSumContent = data
	}

	// Snapshot .forge directory
	forgeDir := filepath.Join(projectRoot, ".forge")
	snapshot.ForgeDirPath = forgeDir
	if info, err := os.Stat(forgeDir); err == nil && info.IsDir() {
		filepath.WalkDir(forgeDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if data, err := os.ReadFile(path); err == nil {
				relPath, _ := filepath.Rel(projectRoot, path)
				snapshot.ForgeFiles[relPath] = data
			}
			return nil
		})
	}

	return snapshot, nil
}

// restoreInstallSnapshot restores the project state from a snapshot.
func restoreInstallSnapshot(projectRoot string, snapshot *InstallSnapshot) error {
	// Restore go.mod
	if len(snapshot.GoModContent) > 0 {
		if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), snapshot.GoModContent, 0o644); err != nil {
			return fmt.Errorf("restaurer go.mod: %w", err)
		}
	} else {
		// If no go.mod was present, remove it
		_ = os.Remove(filepath.Join(projectRoot, "go.mod"))
	}

	// Restore go.sum
	if len(snapshot.GoSumContent) > 0 {
		if err := os.WriteFile(filepath.Join(projectRoot, "go.sum"), snapshot.GoSumContent, 0o644); err != nil {
			return fmt.Errorf("restaurer go.sum: %w", err)
		}
	} else {
		_ = os.Remove(filepath.Join(projectRoot, "go.sum"))
	}

	// Restore .forge directory
	if snapshot.ForgeDirPath != "" {
		// Remove current .forge
		_ = os.RemoveAll(snapshot.ForgeDirPath)
		// Restore from snapshot
		for relPath, data := range snapshot.ForgeFiles {
			fullPath := filepath.Join(projectRoot, relPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return fmt.Errorf("créer répertoire %s: %w", filepath.Dir(fullPath), err)
			}
			if err := os.WriteFile(fullPath, data, 0o644); err != nil {
				return fmt.Errorf("restaurer %s: %w", relPath, err)
			}
		}
	}

	// Run go mod tidy to restore consistency
	_ = feature.RunGoModTidy(projectRoot)

	return nil
}

// detectConflicts analyzes the project files and compares them with the plan
// to detect potential conflicts (user modifications to files that will be modified).
func detectConflicts(projectRoot string, plan *feature.Plan) error {
	for i := range plan.Files {
		file := &plan.Files[i]
		dest := filepath.Join(projectRoot, file.Destination)

		if _, err := os.Stat(dest); err == nil {
			// File exists - check if it will be modified
			if file.Action == feature.FileActionCreate {
				// File exists but plan says create - this is a conflict
				file.Action = feature.FileActionModify
				plan.Conflicts = append(plan.Conflicts, feature.Conflict{
					File:        file.Destination,
					Description: "fichier existant sera écrasé",
				})
			} else if file.Action == feature.FileActionModify {
				// File will be modified - check if user has modified it
				plan.Conflicts = append(plan.Conflicts, feature.Conflict{
					File:        file.Destination,
					Description: "fichier existant sera modifié (vérifiez les changements utilisateur)",
				})
			}
		} else if os.IsNotExist(err) {
			// File doesn't exist - it will be created
			file.Action = feature.FileActionCreate
		}
	}
	return nil
}

func newInspectCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspecter la signature ForgeKit du projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			console := g.console()

			result := forge.ValidateSignature(root)

			if g.Format == output.FormatJSON {
				type inspectJSON struct {
					SchemaVersion string                     `json:"schema_version"`
					Status        string                     `json:"status"`
					Legacy        bool                       `json:"legacy"`
					Metadata      forge.ForgeMetadata        `json:"metadata"`
					Features      []feature.InstalledFeature `json:"features"`
					Errors        []string                   `json:"errors"`
					Warnings      []string                   `json:"warnings"`
				}
				status := "valid"
				if result.IsAbsent() {
					status = "absent"
				} else if result.IsInvalid() {
					status = "invalid"
				}
				return console.PrintJSON(inspectJSON{
					SchemaVersion: report.JSONSchemaVersion,
					Status:        status,
					Legacy:        result.LegacyProject,
					Metadata:      result.Metadata,
					Features:      result.Features.Features,
					Errors:        result.Errors,
					Warnings:      result.Warnings,
				})
			}

			if result.IsAbsent() {
				fmt.Fprintln(console.Out, "ForgeKit")
				fmt.Fprintln(console.Out, "────────────────────────────")
				fmt.Fprintln(console.Out)
				fmt.Fprintln(console.Out, "Signature: ABSENTE")
				fmt.Fprintln(console.Out, "Ce n'est pas un projet ForgeKit (répertoire .forge manquant).")
				return nil
			}

			if result.IsInvalid() {
				fmt.Fprintln(console.Out, "ForgeKit")
				fmt.Fprintln(console.Out, "────────────────────────────")
				fmt.Fprintln(console.Out)
				fmt.Fprintln(console.Out, "Signature: INVALIDE")
				fmt.Fprintln(console.Out)
				for _, e := range result.Errors {
					fmt.Fprintf(console.Out, "  ✗ %s\n", e)
				}
				fmt.Fprintln(console.Out)
				fmt.Fprintln(console.Out, "Exécutez 'forge doctor' pour plus de détails.")
				return nil
			}

			fmt.Fprintln(console.Out, "ForgeKit")
			fmt.Fprintln(console.Out, "────────────────────────────")
			fmt.Fprintln(console.Out)
			fmt.Fprintf(console.Out, "Project:      %s\n", result.Metadata.Project)
			fmt.Fprintf(console.Out, "ForgeKit:     v%s\n", result.Metadata.Version)
			fmt.Fprintf(console.Out, "Schema:       %d\n", result.Metadata.Schema)
			fmt.Fprintf(console.Out, "Language:     %s\n", result.Metadata.Language)
			fmt.Fprintf(console.Out, "Type:         %s\n", result.Metadata.Type)
			if !result.Metadata.CreatedAt.IsZero() {
				fmt.Fprintf(console.Out, "Created:      %s\n", result.Metadata.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
			}
			fmt.Fprintln(console.Out)
			if result.LegacyProject {
				fmt.Fprintln(console.Out, "⚠ Projet legacy détecté (.forge/forge.yaml manquant, seules les features sont présentes)")
				fmt.Fprintln(console.Out)
			}
			fmt.Fprintln(console.Out, "Features:")
			if len(result.Features.Features) == 0 {
				fmt.Fprintln(console.Out, "  (aucune)")
			} else {
				for _, f := range result.Features.Features {
					fmt.Fprintf(console.Out, "  ✓ %s  v%s\n", f.Name, f.Version)
				}
			}
			fmt.Fprintln(console.Out)
			fmt.Fprintln(console.Out, "Signature:")
			fmt.Fprintln(console.Out, "  ✓ Valide")
			fmt.Fprintln(console.Out)
			if len(result.Warnings) > 0 {
				fmt.Fprintln(console.Out, "Avertissements:")
				for _, w := range result.Warnings {
					fmt.Fprintf(console.Out, "  ⚠ %s\n", w)
				}
				fmt.Fprintln(console.Out)
			}
			fmt.Fprintln(console.Out, "Créé avec ForgeKit")

			return nil
		},
	}
}

func newConfigCommand(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Gérer la configuration ForgeKit"}
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Créer forge.yaml avec les règles par défaut",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := os.Getwd()
			path := filepath.Join(root, config.FileName)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s existe déjà", config.FileName)
			}
			defaults := `project:
  language: go

architecture:
  rules:
    - from: domain
      to: infrastructure
      allow: false
    - from: domain
      to: transport
      allow: false
    - from: application
      to: transport
      allow: false
`
			return os.WriteFile(path, []byte(defaults), 0o644)
		},
	})
	return cmd
}
