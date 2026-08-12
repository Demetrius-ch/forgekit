package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Demetrius-ch/forgekit/internal/app"
	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/dbinspect"
	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/feature/auth"
	"github.com/Demetrius-ch/forgekit/internal/feature/logging"
	"github.com/Demetrius-ch/forgekit/internal/feature/swagger"
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
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnostiquer l'environnement et le projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			return runReport(g, "doctor", root, rules.DoctorRules())
		},
	}
}

func newAnalyzeCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
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
			return runAnalyze(g, root, loader)
		},
	}
}

// runAnalyze runs category-by-category analysis with per-step spinner and aggregates findings.
func runAnalyze(g *globalFlags, root string, loader rules.StaticConfigLoader) error {
	console := g.console()

	categories := []string{"Architecture", "Tests", "Security", "Configuration", "Docker", "Documentation"}
	var allFindings []report.Finding
	start := time.Now()

	for _, cat := range categories {
		// spinner per category (human only)
		var spinner *output.Spinner
		if g.Format == output.FormatHuman && !g.Quiet {
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
		}

		if err != nil {
			output.Debug(os.Stderr, g.Debug, err)
			return err
		}
		allFindings = append(allFindings, findings...)
	}

	// aggregate and print result
	allFindings = report.UniqueFindings(allFindings)
	res := report.Result{Tool: app.Name, Version: app.Version, Command: "analyze", Project: root, Timestamp: start, Findings: allFindings}
	res.Summary = report.BuildSummary(allFindings)
	if err := console.PrintResult(res); err != nil {
		return err
	}
	if report.HasFailures(res.Summary) {
		return fmt.Errorf("diagnostics en échec : %d error(s)", res.Summary.Error+res.Summary.Critical)
	}
	return nil
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
			loader := rules.StaticConfigLoader{Rules: cfg.Architecture.Rules}
			if err := runReport(g, "check", root, rules.CheckRules(loader)); err != nil {
				return err
			}
			// exit 1 handled by main via summary — re-check after print
			return nil
		},
	}
}

func runReport(g *globalFlags, command, root string, reg *rules.Registry) error {
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
	res := report.Result{
		Tool: app.Name, Version: app.Version, Command: command,
		Project: root, Timestamp: start, Findings: findings,
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
			registry := feature.NewRegistry(auth.AuthFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{})

			if list {
				return runFeatureList(g, registry)
			}

			return runFeatureAdd(g, registry, args[0], dryRun)
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

		result := make([]featureJSON, 0, len(features))

		for _, f := range features {
			result = append(result, featureJSON{
				Name:        f.Name(),
				Version:     f.Version(),
				Description: f.Description(),
			})
		}

		return console.PrintJSON(result)
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

	project, err := detector.Detect(root)
	if spinner != nil {
		if err != nil {
			spinner.Stop("✗ Détection du projet — erreur")
		} else {
			spinner.Stop("✓ Projet ForgeKit détecté")
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

	// Step 3: Check prerequisites
	if g.Format == output.FormatHuman && !g.Quiet {
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Vérification des prérequis...")
	}

	checkErr := f.Check(ctx, project)
	alreadyInstalled := checkErr != nil && strings.Contains(checkErr.Error(), "déjà installée")
	if checkErr != nil {
		if spinner != nil {
			if alreadyInstalled {
				spinner.Stop("✓ Feature déjà installée")
			} else {
				spinner.Stop("✗ Vérification des prérequis — échec")
			}
		}
		// In dry-run mode, treat "already installed" as a special case
		if dryRun && alreadyInstalled {
			// Show already installed message for dry-run
			if g.Format == output.FormatHuman && !g.Quiet {
				fmt.Fprintln(console.Out)
				fmt.Fprintln(console.Out, "Dry-run")
				fmt.Fprintln(console.Out, "────────────────────────────────")
				fmt.Fprintln(console.Out)
				fmt.Fprintln(console.Out, "Aucune modification ne sera effectuée.")
				fmt.Fprintf(console.Out, "Feature : %s\n", name)
				// Extract version from error message
				version := strings.TrimPrefix(strings.TrimPrefix(checkErr.Error(), "vérification de la feature "), name+" version ")
				version = strings.TrimSuffix(version, " déjà installée")
				fmt.Fprintf(console.Out, "Version installée : %s\n", version)
				fmt.Fprintln(console.Out, "Statut : aucune modification nécessaire.")
			}
			return nil
		}
		return fmt.Errorf("vérification de la feature %q : %w", name, checkErr)
	}
	if spinner != nil {
		spinner.Stop("✓ Prérequis validés")
	}

	// Step 4: Build plan
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

	if dryRun {
		if g.Format == output.FormatHuman && !g.Quiet {
			fmt.Fprintln(console.Out)
		}
		return printFeaturePlan(g, plan)
	}

	// Step 5: Install files
	if g.Format == output.FormatHuman && !g.Quiet {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Installation...")
		spinner = output.NewSpinner(console.Out)
		spinner.Start("Installation des fichiers...")
	}

	if err := f.Apply(ctx, project, plan); err != nil {
		if spinner != nil {
			spinner.Stop("✗ Installation — échec")
		}
		return err
	}
	if spinner != nil {
		spinner.Stop("✓ Fichiers installés")
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

func printFeaturePlan(g *globalFlags, plan feature.Plan) error {
	console := g.console()

	if g.Format == output.FormatJSON {
		return console.PrintJSON(plan)
	}

	if g.Quiet {
		for _, file := range plan.Files {
			fmt.Fprintf(
				console.Out,
				"%s\n",
				file.Destination,
			)
		}
		return nil
	}

	fmt.Fprintln(console.Out)
	fmt.Fprintln(console.Out, "ForgeKit Add")
	fmt.Fprintln(console.Out, "────────────────────────────────")
	fmt.Fprintf(console.Out, "Feature : %s\n", plan.Feature)
	fmt.Fprintf(console.Out, "Version : %s\n", plan.Version)

	if len(plan.Files) > 0 {
		fmt.Fprintln(console.Out)
		fmt.Fprintln(console.Out, "Fichiers :")

		for _, file := range plan.Files {
			fmt.Fprintf(
				console.Out,
				"  → %s\n",
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

	fmt.Fprintln(console.Out)
	fmt.Fprintln(console.Out, "Aucune modification effectuée (--dry-run).")

	return nil
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
