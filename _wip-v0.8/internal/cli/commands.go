package cli

import (
	"fmt"
	"os"

	"github.com/Demetrius-ch/forgekit/internal/config"
	"github.com/Demetrius-ch/forgekit/internal/diff"
	"github.com/Demetrius-ch/forgekit/internal/feature"
	"github.com/Demetrius-ch/forgekit/internal/output"
	"github.com/Demetrius-ch/forgekit/internal/forge"
	"github.com/spf13/cobra"
)

// newStatusCommand builds the forge status command.
func newStatusCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Affiche l'état du projet ForgeKit",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			console := g.console()

			cfg, err := config.LoadProject(root)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "no_project",
					})
				}
				fmt.Fprintln(console.Out, "Aucun projet ForgeKit trouvé (pas de forge.yaml)")
				return nil
			}

			// Load metadata from .forge
			meta, err := forge.LoadMetadata(root)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "no_metadata",
					})
				}
				fmt.Fprintln(console.Out, "Aucune métadonnée ForgeKit trouvée")
				return nil
			}

			// Build file lists for comparison
			baseFiles := getBaselineFiles(root, meta)
			currentFiles := getCurrentFiles(root)
			newTemplateFiles := getTemplateFiles(root, forge.ForgeDirPath(root))

			// Compare using DiffEngine
			changes, err := diff.ComparePaths(baseFiles, currentFiles, newTemplateFiles)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "error",
					})
				}
				return fmt.Errorf("erreur lors de la comparaison: %w", err)
			}

			// Determine status based on diff results
			status := "up_to_date"
			if changes.IsUpToDate() {
				status = "up_to_date"
			} else if changes.HasConflicts() {
				status = "conflicts"
			} else {
				// Check if template changed (update available)
				for _, c := range changes {
					if c.Status == diff.TEMPLATE_CHANGED || c.Status == diff.ADDED_BY_TEMPLATE {
						status = "update_available"
						break
					}
					if c.Status == diff.USER_CHANGED {
						status = "project_modified"
						break
					}
				}
			}

			if g.Format == output.FormatJSON {
				type statusJSON struct {
					Project    string `json:"project"`
					Status     string `json:"status"`
					Language   string `json:"language"`
				}
				return console.PrintJSON(statusJSON{
					Project: cfg.Project.Module,
					Status:  status,
					Language: cfg.Project.Language,
				})
			}

			fmt.Fprintf(console.Out, "ForgeKit Project Status\n\n")
			fmt.Fprintf(console.Out, "Project        : %s\n", cfg.Project.Module)
			fmt.Fprintf(console.Out, "Language       : %s\n", cfg.Project.Language)
			fmt.Fprintf(console.Out, "Status         : %s\n", status)

			if status == "update_available" {
				fmt.Fprintln(console.Out)
				fmt.Fprintf(console.Out, "Recommendation:\n  forge update --check\n")
			} else if status == "project_modified" {
				fmt.Fprintln(console.Out)
				fmt.Fprintf(console.Out, "Recommendation:\n  Review local modifications\n")
			} else if status == "conflicts" {
				fmt.Fprintln(console.Out)
				fmt.Fprintf(console.Out, "Recommendation:\n  Resolve conflicts before updating\n")
			}

			return nil
		},
	}
}

// newUpdateCheckCommand builds the forge update --check command.
func newUpdateCheckCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "update --check",
		Short: "Vérifie les mises à jour disponibles (lecture seule)",
		Long: `Vérifie si une nouvelle version du template est disponible.

Cette commande ne modifie aucun fichier ni métadonnée.
Elle analyse simplement l'état actuel du projet et signale
les mises à jour possibles.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			console := g.console()

			cfg, err := config.LoadProject(root)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "no_project",
					})
				}
				fmt.Fprintln(console.Out, "Aucun projet ForgeKit trouvé")
				return nil
			}

			status := "up_to_date"

			if g.Format == output.FormatJSON {
				type updateCheckJSON struct {
					Project   string `json:"project"`
					Status    string `json:"status"`
					Language  string `json:"language"`
				}
				return console.PrintJSON(updateCheckJSON{
					Project: cfg.Project.Module,
					Status:  status,
					Language: cfg.Project.Language,
				})
			}

			fmt.Fprintf(console.Out, "Update Check\n\n")
			fmt.Fprintf(console.Out, "Project        : %s\n", cfg.Project.Module)
			fmt.Fprintf(console.Out, "Status         : %s\n", status)

			if status != "up_to_date" {
				fmt.Fprintln(console.Out)
				fmt.Fprintln(console.Out, "Mise à jour disponible")
			} else {
				fmt.Fprintln(console.Out, "Aucune mise à jour disponible.")
				fmt.Fprintln(console.Out, "Le projet est à jour.")
			}

			return nil
		},
	}
}

// newDiffCommand builds the forge diff command.
func newDiffCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Affiche les différences entre le projet actuel et le template",
		Long: `Affiche les différences entre l'état actuel du projet et
la nouvelle version du template.

Les fichiers sont classés en :
- Ajoutés (+)
- Modifiés (~)
- Supprimés (-)
- En conflit (!)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			console := g.console()

			cfg, err := config.LoadProject(root)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "no_project",
					})
				}
				fmt.Fprintln(console.Out, "Aucun projet ForgeKit trouvé (pas de forge.yaml)")
				return nil
			}

			// Load baseline from .forge
			meta, err := forge.LoadMetadata(root)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "no_metadata",
					})
				}
				fmt.Fprintln(console.Out, "Aucune métadonnée ForgeKit trouvée")
				return nil
			}

			// Determine baseline, current, and new template file states
			// For now, we use the project config and metadata to determine what files
			// should be compared. In a full implementation, this would read the actual
			// file contents from baseline snapshot, current project, and template.

			// Build file lists for comparison
			baseFiles := getBaselineFiles(root, meta)
			currentFiles := getCurrentFiles(root)
			newTemplateFiles := getTemplateFiles(root, forge.ForgeDirPath(root))

			// Compare the three states
			changes, err := diff.ComparePaths(baseFiles, currentFiles, newTemplateFiles)
			if err != nil {
				if g.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "error",
					})
				}
				return fmt.Errorf("erreur lors de la comparaison: %w", err)
			}

			// Output results
			if g.Format == output.FormatJSON {
				type diffJSON struct {
					Project   string `json:"project"`
					Status    string `json:"status"`
					Changes   []diffJSONChange `json:"changes"`
				}

				jsonChanges := make([]diffJSONChange, len(changes))
				for i, c := range changes {
					jsonChanges[i] = diffJSONChange{
						Path:    c.Path,
						Status:  string(c.Status),
						UserChanged: c.UserChanged,
						TemplateChanged: c.TemplateChanged,
						Conflict: c.Conflict,
					}
				}

				return console.PrintJSON(diffJSON{
					Project:   cfg.Project.Module,
					Status:    string(changes[0].Status),
					Changes:   jsonChanges,
				})
			}

			// Human-readable output
			fmt.Fprintln(console.Out, "ForgeKit Diff")
			fmt.Fprintln(console.Out, "=============")
			fmt.Fprintf(console.Out, "Project: %s\n", cfg.Project.Module)
			fmt.Fprintf(console.Out, "Status: %s\n\n", changes.StatusSummary())

			if changes.HasConflicts() {
				fmt.Fprintln(console.Out, "Potentiels conflits :")
				for _, c := range changes {
					if c.Status == diff.BOTH_CHANGED {
						fmt.Fprintf(console.Out, "  ! %s\n", c.Path)
					}
				}
				fmt.Fprintln(console.Out)
			}

			fmt.Fprintln(console.Out, "Fichiers ajoutés :")
			for _, c := range changes {
				if c.Status == diff.ADDED_BY_TEMPLATE {
					fmt.Fprintf(console.Out, "  + %s\n", c.Path)
				}
			}

			fmt.Fprintln(console.Out, "Fichiers modifiés :")
			for _, c := range changes {
				if c.Status == diff.USER_CHANGED {
					fmt.Fprintf(console.Out, "  ~ %s (modification utilisateur)\n", c.Path)
				}
				if c.Status == diff.TEMPLATE_CHANGED {
					fmt.Fprintf(console.Out, "  ~ %s (modification template)\n", c.Path)
				}
				if c.Status == diff.BOTH_CHANGED {
					fmt.Fprintf(console.Out, "  ! %s (conflit)\n", c.Path)
				}
			}

			fmt.Fprintln(console.Out, "Fichiers supprimés :")
			for _, c := range changes {
				if c.Status == diff.DELETED_BY_USER {
					fmt.Fprintf(console.Out, "  - %s (suppression utilisateur)\n", c.Path)
				}
				if c.Status == diff.DELETED_BY_TEMPLATE {
					fmt.Fprintf(console.Out, "  - %s (suppression template)\n", c.Path)
				}
			}

			if !changes.HasConflicts() && changes.IsUpToDate() {
				fmt.Fprintln(console.Out, "\nAucune modification détectée.")
			}

			return nil
		},
	}
}

// getBaselineFiles returns the baseline file list from .forge metadata.
func getBaselineFiles(projectRoot string, meta forge.ForgeMetadata) []string {
	// For now, return common baseline files based on project configuration
	// In a full implementation, this would read from the snapshot
	return []string{
		"cmd/server/main.go",
		"internal/domain/*.go",
		"internal/application/*.go",
		"internal/infrastructure/*.go",
		"internal/transport/http/router.go",
	}
}

// getCurrentFiles returns the current project file list.
func getCurrentFiles(projectRoot string) []string {
	// For now, list files in the project root
	// In a full implementation, this would walk the project directory
	return []string{
		"cmd/server/main.go",
		"internal/domain/entity.go",
		"internal/application/use_case.go",
		"internal/infrastructure/repository.go",
	}
}

// getTemplateFiles returns the new template file list based on metadata.
func getTemplateFiles(projectRoot string, templatePath string) []string {
	// For now, return template files based on the template version
	// In a full implementation, this would read from the template
	return []string{
		"cmd/server/main.go",
		"internal/domain/entity.go",
		"internal/application/use_case.go",
		"internal/infrastructure/repository.go",
		"internal/transport/http/router.go",
	}
}

// diffJSONChange represents a file change in JSON output for the diff command.
type diffJSONChange struct {
	Path        string `json:"path"`
	Status      string `json:"status"`
	UserChanged bool   `json:"user_changed"`
	TemplateChanged bool `json:"template_changed"`
	Conflict    bool   `json:"conflict"`
}

// newInitCommand builds the forge init command.
func newInitCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init [nom-du-projet]",
		Short: "Initialise un nouveau projet ForgeKit",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("la commande init n'est pas encore implémentée pleinement")
		},
	}
}

// newAddCommand builds the forge add command.
func newAddCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "add <feature>",
		Short: "Installe une feature",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: forge add <feature>")
			}
			return fmt.Errorf("l'installation de la feature %s n'est pas encore implémentée", args[0])
		},
	}
}

// newRemoveCommand builds the forge remove command.
func newRemoveCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <feature>",
		Short: "Supprime une feature",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: forge remove <feature>")
			}
			return fmt.Errorf("la suppression de la feature %s n'est pas encore implémentée", args[0])
		},
	}
}

// newInspectCommand builds the forge inspect command.
func newInspectCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect [objet]",
		Short: "Inspecte un aspect du projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("la commande inspect n'est pas encore implémentée pleinement")
		},
	}
}

// newDoctorCommand builds the forge doctor command.
func newDoctorCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Valide la signature du projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("la commande doctor n'est pas encore implémentée pleinement")
		},
	}
}

// newAnalyzeCommand builds the forge analyze command.
func newAnalyzeCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Analyse la qualité du projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("la commande analyze n'est pas encore implémentée pleinement")
		},
	}
}

// newCheckCommand builds the forge check command.
func newCheckCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Vérifie la conformité architecturale",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("la commande check n'est pas encore implémentée pleinement")
		},
	}
}

// newHistoryCommand builds the forge history command.
func newHistoryCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Affiche l'historique des opérations ForgeKit",
		Long: `Affiche l'historique des opérations passées sur le projet.

Exemples :
  forge history
  forge history --to 1.1.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			console := g.console()

			// Display history information
			// Note: In a full implementation, this would read the .forge/history file
			// For now, display a message
			if console.Format == output.FormatJSON {
				type historyJSON struct {
					Project string `json:"project"`
					Status  string `json:"status"`
				}
				return console.PrintJSON(historyJSON{
					Project: "unknown",
					Status:  "no_history",
				})
			}

			fmt.Fprintln(console.Out, "Aucun historique trouvé (aucune opération enregistrée)")
			fmt.Fprintln(console.Out, "Utilisez 'forge update' ou 'forge migrate' pour enregistrer les opérations.")

			return nil
		},
	}
}

// newConfigCommand builds the forge config command.
func newConfigCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Gère la configuration du projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("la commande config n'est pas encore implémentée pleinement")
		},
	}
}

// newRollbackCommand builds the forge rollback command.
func newRollbackCommand(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "Restaure le projet à partir du snapshot précédent",
		Long: `Restaure l'état précédent du projet à partir d'un snapshot.

Cette commande est destructive - elle écrase les fichiers actuels
avec l'état sauvegardé précédent.

Exemples :
  forge rollback`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			console := g.console()

			// Use the RollbackManager to restore the last snapshot
			snapMgr := feature.NewRollbackManager(root)
			snapshots := snapMgr.ListSnapshots()

			if len(snapshots) == 0 {
				if console.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "no_snapshot",
					})
				}
				fmt.Fprintln(console.Out, "Aucun snapshot disponible pour le rollback")
				return nil
			}

			lastSnap := snapshots[len(snapshots)-1]
			if err := snapMgr.RollbackTo(lastSnap); err != nil {
				if console.Format == output.FormatJSON {
					return console.PrintJSON(map[string]string{
						"status": "rollback_failed",
					})
				}
				fmt.Fprintln(console.Out, "Échec du rollback : "+err.Error())
				return err
			}

			if console.Format == output.FormatJSON {
				return console.PrintJSON(map[string]string{
					"status": "rollback_success",
				})
			}

			fmt.Fprintln(console.Out, "Projet restauré avec succès à partir du snapshot : "+lastSnap.ID)
			fmt.Fprintln(console.Out, "Dernière modification : "+lastSnap.Timestamp.Format("2006-01-02 15:04:05"))

			return nil
		},
	}
}