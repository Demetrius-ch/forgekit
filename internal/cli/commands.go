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
	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/feature/auth"
	"github.com/Demetrius-ch/forgekit/internal/generator"
	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/prompt"
	"github.com/Demetrius-ch/forgekit/internal/report"
	"github.com/Demetrius-ch/forgekit/internal/rules"
	"github.com/spf13/cobra"
)

func newInitCommand(g *globalFlags) *cobra.Command {
	var (
		modulePath      string
		httpPort        int
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

			opts := generator.InitOptions{
				ProjectName:     projectName,
				TargetDir:       absDir,
				HTTPPort:        httpPort,
				DryRun:          dryRun,
				Author:          config.ResolveAuthor(author, absDir),
				SkipPostprocess: skipPostprocess,
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
	cmd.Flags().BoolVar(&skipPostprocess, "skip-postprocess", false, "Ne pas exécuter gofmt et go test après génération")
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
			registry := feature.NewRegistry(auth.AuthFeature{})

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

	if err := f.Check(ctx, project); err != nil {
		if spinner != nil {
			spinner.Stop("✗ Vérification des prérequis — échec")
		}
		return fmt.Errorf("vérification de la feature %q : %w", name, err)
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
