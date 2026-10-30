package diff

import "fmt"

// FileChange represents the comparison result for a single file between
// the baseline, current project state, and new template state.
type FileChange struct {
	Path            string
	Status          Status
	UserChanged     bool
	TemplateChanged bool
	Conflict        bool
}

// Status represents the change status of a file relative to baseline/template.
type Status string

const (
	// UNCHANGED means the file is the same in baseline, current, and template.
	UNCHANGED Status = "unchanged"

	// USER_CHANGED means the user modified the file, but the template did not.
	USER_CHANGED Status = "user_changed"

	// TEMPLATE_CHANGED means the template modified the file, but the user did not.
	TEMPLATE_CHANGED Status = "template_changed"

	// BOTH_CHANGED means both the user and template modified the same file, causing a conflict.
	BOTH_CHANGED Status = "both_changed"

	// ADDED_BY_TEMPLATE means the file was added by the new template (doesn't exist in current project).
	ADDED_BY_TEMPLATE Status = "added_by_template"

	// USER_ADDED means the user added a file that doesn't exist in the template.
	USER_ADDED Status = "user_added"

	// DELETED_BY_USER means the user deleted a file that exists in the template.
	DELETED_BY_USER Status = "deleted_by_user"

	// DELETED_BY_TEMPLATE means the template removed a file that exists in the current project.
	DELETED_BY_TEMPLATE Status = "deleted_by_template"
)

// FileChanges is a collection of FileChange results.
type FileChanges []FileChange

// HasConflicts returns true if any file has a conflict status.
func (fc FileChanges) HasConflicts() bool {
	for _, f := range fc {
		if f.Status == BOTH_CHANGED {
			return true
		}
	}
	return false
}

// CountByStatus counts files by status.
func (fc FileChanges) CountByStatus() map[Status]int {
	counts := make(map[Status]int)
	for _, f := range fc {
		counts[f.Status]++
	}
	return counts
}

// StatusSummary returns a human-readable summary of the overall status.
func (fc FileChanges) StatusSummary() string {
	if len(fc) == 0 {
		return "aucun fichier"
	}

	counts := fc.CountByStatus()
	var parts []string

	if counts[ADDED_BY_TEMPLATE] > 0 {
		parts = append(parts, fmt.Sprintf("%d fichier(s) ajouté(s) par le template", counts[ADDED_BY_TEMPLATE]))
	}
	if counts[USER_CHANGED] > 0 {
		parts = append(parts, fmt.Sprintf("%d fichier(s) modifié(s) par l'utilisateur", counts[USER_CHANGED]))
	}
	if counts[TEMPLATE_CHANGED] > 0 {
		parts = append(parts, fmt.Sprintf("%d fichier(s) modifié(s) par le template", counts[TEMPLATE_CHANGED]))
	}
	if counts[BOTH_CHANGED] > 0 {
		parts = append(parts, fmt.Sprintf("%d fichier(s) en conflit", counts[BOTH_CHANGED]))
	}
	if counts[DELETED_BY_USER] > 0 {
		parts = append(parts, fmt.Sprintf("%d fichier(s) supprimé(s) par l'utilisateur", counts[DELETED_BY_USER]))
	}
	if counts[DELETED_BY_TEMPLATE] > 0 {
		parts = append(parts, fmt.Sprintf("%d fichier(s) supprimé(s) par le template", counts[DELETED_BY_TEMPLATE]))
	}
	if counts[UNCHANGED] > 0 && len(parts) == 0 {
		return "aucun changement"
	}

	return fmt.Sprintf("(%s)", parts)
}

// IsUpToDate returns true if no changes were detected (project is up to date).
func (fc FileChanges) IsUpToDate() bool {
	for _, f := range fc {
		if f.Status != UNCHANGED {
			return false
		}
	}
	return true
}
