package diff

import ()

// diffEngine compares three states of project files: BASE, CURRENT, and NEW (template).
type diffEngine struct {
	base    map[string]string
	current map[string]string
	new     map[string]string
}

// NewDiffEngine creates a new diff engine with the three file states.
func NewDiffEngine(base, current, new map[string]string) *diffEngine {
	return &diffEngine{
		base:    base,
		current: current,
		new:     new,
	}
}

// Compare compares the three states and returns a FileChanges result.
func (e *diffEngine) Compare() FileChanges {
	var changes FileChanges
	allPaths := e.collectAllPaths()

	for _, path := range allPaths {
		change := e.compareFile(path)
		changes = append(changes, change)
	}

	return changes
}

// collectAllPaths returns all unique file paths across the three states.
func (e *diffEngine) collectAllPaths() []string {
	paths := make(map[string]bool)
	for p := range e.base {
		paths[p] = true
	}
	for p := range e.current {
		paths[p] = true
	}
	for p := range e.new {
		paths[p] = true
	}
	result := make([]string, 0, len(paths))
	for p := range paths {
		result = append(result, p)
	}
	return result
}

// compareFile compares a single file across the three states and returns a FileChange.
func (e *diffEngine) compareFile(path string) FileChange {
	inBase := e.base[path] != ""
	inCurrent := e.current[path] != ""
	inNew := e.new[path] != ""

	baseContent := e.base[path]
	currentContent := e.current[path]
	newContent := e.new[path]

	userChanged := false
	templateChanged := false
	conflict := false
	status := UNCHANGED

	// Case 1: ADDED_BY_TEMPLATE - BASE absent, CURRENT absent, NEW present
	if !inBase && !inCurrent && inNew {
		status = ADDED_BY_TEMPLATE
		templateChanged = true
	}

	// Case 2: USER_ADDED - BASE absent, CURRENT present
	if !inBase && inCurrent {
		status = USER_ADDED
		userChanged = true
	}

	// Case 3: DELETED_BY_USER - BASE present, CURRENT absent
	if inBase && !inCurrent {
		status = DELETED_BY_USER
		userChanged = true
	}

	// Case 4: DELETED_BY_TEMPLATE - BASE present, NEW absent
	if inBase && !inNew {
		status = DELETED_BY_TEMPLATE
		templateChanged = true
	}

	// Case 5: UNCHANGED - BASE == CURRENT == NEW
	if baseContent == currentContent && currentContent == newContent {
		status = UNCHANGED
	}

	// Case 6: USER_CHANGED - BASE != CURRENT but BASE == NEW
	if baseContent != currentContent && baseContent == newContent {
		status = USER_CHANGED
		userChanged = true
	}

	// Case 7: TEMPLATE_CHANGED - BASE == CURRENT but BASE != NEW
	if baseContent == currentContent && baseContent != newContent {
		status = TEMPLATE_CHANGED
		templateChanged = true
	}

	// Case 8: BOTH_CHANGED - BASE != CURRENT and BASE != NEW and CURRENT != NEW
	if baseContent != currentContent && baseContent != newContent && currentContent != newContent {
		status = BOTH_CHANGED
		conflict = true
		userChanged = true
		templateChanged = true
	}

	// Case 9: Partial differences - determine which changed
	if status == UNCHANGED {
		if baseContent != currentContent && baseContent != newContent {
			status = BOTH_CHANGED
			conflict = true
			userChanged = true
			templateChanged = true
		} else if baseContent != currentContent {
			status = USER_CHANGED
			userChanged = true
		} else if baseContent != newContent {
			status = TEMPLATE_CHANGED
			templateChanged = true
		}
	}

	return FileChange{
		Path:            path,
		Status:          status,
		UserChanged:     userChanged,
		TemplateChanged: templateChanged,
		Conflict:        conflict,
	}
}

// CompareFilesystems compares file contents from three filesystem snapshots.
// base, current, and new are maps of relative paths to file content (byte slices).
func CompareFilesystems(base, current, new map[string][]byte) (FileChanges, error) {
	contentMaps := map[string]map[string]string{
		"base":    byteSliceMapToStringMap(base),
		"current": byteSliceMapToStringMap(current),
		"new":     byteSliceMapToStringMap(new),
	}

	engine := NewDiffEngine(contentMaps["base"], contentMaps["current"], contentMaps["new"])
	return engine.Compare(), nil
}

// byteSliceMapToStringMap converts a map of path -> []byte to map of path -> string.
func byteSliceMapToStringMap(m map[string][]byte) map[string]string {
	result := make(map[string]string, len(m))
	for p, content := range m {
		result[p] = string(content)
	}
	return result
}

// ComparePaths compares three sets of file paths (baseline, current, template)
// and returns FileChanges results.
func ComparePaths(base, current, new []string) (FileChanges, error) {
	baseMap := make(map[string]string)
	currentMap := make(map[string]string)
	newMap := make(map[string]string)

	for _, p := range base {
		baseMap[p] = "baseline"
	}
	for _, p := range current {
		currentMap[p] = "current"
	}
	for _, p := range new {
		newMap[p] = "template"
	}

	engine := NewDiffEngine(baseMap, currentMap, newMap)
	return engine.Compare(), nil
}
