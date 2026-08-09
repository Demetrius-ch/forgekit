package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forgekit/forgekit/internal/app"
	"github.com/forgekit/forgekit/internal/config"
	"github.com/forgekit/forgekit/internal/generator"
	"github.com/forgekit/forgekit/internal/output"
	"github.com/forgekit/forgekit/internal/prompt"
	"github.com/forgekit/forgekit/internal/report"
	"github.com/forgekit/forgekit/internal/rules"
	"github.com/spf13/cobra"
)

func newInitCommand(g *globalFlags) *cobra.Command {
	var (
		modulePath     string
		httpPort       int
		databaseName   string
		author         string
		nonInteractive bool
		targetDir      string
		dryRun         bool
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

			opts := generator.InitOptions{
				ProjectName: projectName,
				TargetDir:   absDir,
				HTTPPort:    httpPort,
				DryRun:      dryRun,
				Author:      config.ResolveAuthor(author, absDir),
			}

			if nonInteractive || dryRun {
				opts.ModulePath = modulePath
				opts.DatabaseName = databaseName
				if opts.ModulePath == "" {
					opts.ModulePath = defaultModulePath(projectName)
				}
				if opts.DatabaseName == "" {
					opts.DatabaseName = defaultDatabaseName(projectName)
				}
			} else {
				p := prompt.New(os.Stdin, os.Stdout)
				fmt.Fprintf(os.Stdout, "\nConfiguration du projet %q\n\n", projectName)
				module, err := p.AskString("Chemin du module Go", defaultModulePath(projectName))
				if err != nil {
					return err
				}
				opts.ModulePath = module
				port, err := p.AskInt("Port HTTP", httpPort)
				if err != nil {
					return err
				}
				opts.HTTPPort = port
				db, err := p.AskString("Nom PostgreSQL", defaultDatabaseName(projectName))
				if err != nil {
					return err
				}
				opts.DatabaseName = db
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
				printInitSummaryOpts(opts.ProjectName, opts.ModulePath, absDir, opts.DatabaseName, opts.HTTPPort)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&modulePath, "module", "", "Chemin du module Go")
	cmd.Flags().IntVar(&httpPort, "port", 8080, "Port HTTP")
	cmd.Flags().StringVar(&databaseName, "db-name", "", "Nom de la base PostgreSQL")
	cmd.Flags().StringVar(&author, "author", "", "Auteur du projet")
	cmd.Flags().StringVar(&targetDir, "dir", "", "Répertoire cible")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Sans prompts interactifs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Afficher le plan sans écrire sur le disque")
	return cmd
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
			return runReport(g, "analyze", root, rules.AnalyzeRules(loader))
		},
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
	findings, err := reg.Run(context.Background(), rules.Context{ProjectRoot: root, Language: "go"})
	if err != nil {
		output.Debug(os.Stderr, g.Debug, err)
		return err
	}
	res := report.Result{
		Tool: app.Name, Version: app.Version, Command: command,
		Project: root, Timestamp: time.Now(), Findings: findings,
	}
	res.Summary = report.BuildSummary(findings)
	console := g.console()
	if err := console.PrintResult(res); err != nil {
		return err
	}
	if report.HasFailures(res.Summary) {
		return fmt.Errorf("diagnostics en échec : %d error(s)", res.Summary.Error+res.Summary.Critical)
	}
	return nil
}

func newAddCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add [feature]",
		Short: "Ajouter une fonctionnalité (auth, redis, swagger…)",
		Long:  "V0.2+ — architecture extensible via manifestes de features.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("forge add sera disponible en V0.2 (manifestes : auth, postgres, redis, swagger, docker)")
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
