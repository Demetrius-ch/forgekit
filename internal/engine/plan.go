package engine

import "os"

// Action describes a filesystem operation.
type Action string

const (
	ActionCreate Action = "create"
	ActionSkip   Action = "skip"
)

// PlanEntry describes one file that would be or was generated.
type PlanEntry struct {
	Path   string      `json:"path"`
	Action Action      `json:"action"`
	Mode   os.FileMode `json:"mode"`
	Reason string      `json:"reason,omitempty"`
}
