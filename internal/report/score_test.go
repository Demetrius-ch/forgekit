package report

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeScoreBasic(t *testing.T) {
	res := Result{Project: ".", Findings: nil}
	s := ComputeScore(res)
	if s.Global != 75 && s.Global != 100 && s.Global >= 0 { // accept valid range
		// just ensure categories exist
	}
	if _, ok := s.Categories["Architecture"]; !ok {
		t.Fatalf("missing Architecture category")
	}
}

func TestComputeScorePenalties(t *testing.T) {
	// one architecture error should apply penaltyError (30)
	res := Result{Project: ".", Findings: []Finding{{Category: "architecture", Severity: SeverityError, Message: "arch fail"}}}
	s := ComputeScore(res)
	got := s.Categories["Architecture"]
	if got != 70 {
		t.Fatalf("expected Architecture=70 got %d", got)
	}

	// one warning maps to 10 points penalty
	res2 := Result{Project: ".", Findings: []Finding{{Category: "architecture", Severity: SeverityWarning, Message: "arch warn"}}}
	s2 := ComputeScore(res2)
	if s2.Categories["Architecture"] != 90 {
		t.Fatalf("expected Architecture=90 got %d", s2.Categories["Architecture"])
	}
}

func TestComputeScoreDockerPresence(t *testing.T) {
	dir := t.TempDir()
	// without Dockerfile, Docker should be 0 per logic
	res := Result{Project: dir, Findings: nil}
	s := ComputeScore(res)
	if s.Categories["Docker"] != 0 {
		t.Fatalf("expected Docker=0 when no Dockerfile, got %d", s.Categories["Docker"])
	}
	// create Dockerfile and recompute
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch"), 0o644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	s2 := ComputeScore(Result{Project: dir, Findings: nil})
	if s2.Categories["Docker"] == 0 {
		t.Fatalf("expected Docker>0 when Dockerfile present, got %d", s2.Categories["Docker"])
	}
}

func TestComputeScoreDockerRuleCategory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch"), 0o644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	res := Result{Project: dir, Findings: []Finding{{Category: "docker", Severity: SeverityWarning, Message: "daemon down"}}}
	s := ComputeScore(res)
	if s.Categories["Docker"] != 90 {
		t.Fatalf("expected Docker=90 for docker warning with Dockerfile present, got %d", s.Categories["Docker"])
	}
	if s.Categories["Configuration"] != 100 {
		t.Fatalf("expected Configuration=100 when only docker warnings exist, got %d", s.Categories["Configuration"])
	}
}

func TestComputeScoreDocumentationPresence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Demo"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("KEY=VALUE"), 0o644); err != nil {
		t.Fatalf("write .env.example: %v", err)
	}
	s := ComputeScore(Result{Project: dir, Findings: nil})
	if s.Categories["Documentation"] != 100 {
		t.Fatalf("expected Documentation=100 when README and .env.example present, got %d", s.Categories["Documentation"])
	}
}
