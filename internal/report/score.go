package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Score represents per-category scoring and global score.
type Score struct {
	Categories map[string]int `json:"categories"`
	Global     int            `json:"global"`
	Notes      []string       `json:"notes,omitempty"`
}

// ComputeScore computes simple category scores based on findings and project files.
// It uses deterministic rules and avoids magic numbers scattered in code.
func ComputeScore(res Result) Score {
	const (
		penaltyError   = 30
		penaltyWarning = 10
		maxScore       = 100
	)

	// Base categories we expose to the user.
	cats := map[string]int{
		"Architecture":  maxScore,
		"Tests":         maxScore,
		"Security":      maxScore,
		"Configuration": maxScore,
		"Docker":        maxScore,
		"Documentation": maxScore,
	}

	// Tally findings by their category.
	errs := make(map[string]int)
	warns := make(map[string]int)
	for _, f := range res.Findings {
		switch f.Severity {
		case SeverityError, SeverityCritical:
			errs[f.Category]++
		case SeverityWarning:
			warns[f.Category]++
		default:
			// info/pass ignored for penalties
		}
	}

	// Map internal finding categories to exposed categories.
	mapCat := func(in string) string {
		switch in {
		case "architecture":
			return "Architecture"
		case "tests":
			return "Tests"
		case "security":
			return "Security"
		case "docker":
			return "Docker"
		case "project", "environment":
			return "Configuration"
		case "documentation":
			return "Documentation"
		default:
			return "Configuration"
		}
	}

	// Apply penalties based on findings.
	for k, v := range errs {
		out := mapCat(k)
		cats[out] -= v * penaltyError
	}
	for k, v := range warns {
		out := mapCat(k)
		cats[out] -= v * penaltyWarning
	}

	// Basic file-based checks to refine scores: Tests, Docker, Documentation.
	project := res.Project
	if project == "" {
		project = "."
	}

	// Tests: presence of any _test.go files.
	hasTests := false
	_ = filepath.Walk(project, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			hasTests = true
			return filepath.SkipAll
		}
		return nil
	})
	if !hasTests {
		cats["Tests"] = 30
	}

	// Docker: check Dockerfile or docker/docker-compose.yml
	if _, err := os.Stat(filepath.Join(project, "Dockerfile")); os.IsNotExist(err) {
		if _, err2 := os.Stat(filepath.Join(project, "docker", "docker-compose.yml")); os.IsNotExist(err2) {
			cats["Docker"] = 0
		}
	}

	// Documentation: README.md and .env.example
	docs := 0
	if _, err := os.Stat(filepath.Join(project, "README.md")); err == nil {
		docs += 1
	}
	if _, err := os.Stat(filepath.Join(project, ".env.example")); err == nil {
		docs += 1
	}
	if docs == 0 {
		cats["Documentation"] = 20
	} else if docs == 1 {
		cats["Documentation"] = 60
	}

	// Clamp scores to [0,100]
	for k, v := range cats {
		if v < 0 {
			cats[k] = 0
		} else if v > maxScore {
			cats[k] = maxScore
		} else {
			cats[k] = v
		}
	}

	// Global score = average of categories.
	sum := 0
	count := 0
	for _, v := range cats {
		sum += v
		count++
	}
	avg := 0
	if count > 0 {
		avg = sum / count
	}

	notes := []string{}
	// Recommendations: collect top warnings/errors
	for _, f := range res.Findings {
		if f.Severity == SeverityWarning || f.Severity == SeverityError || f.Severity == SeverityCritical {
			notes = append(notes, fmt.Sprintf("%s: %s", strings.ToUpper(string(f.Severity)), f.Message))
		}
	}

	return Score{Categories: cats, Global: avg, Notes: notes}
}
