package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Demetrius-ch/forgekit/internal/report"
)

func TestJSONFormatHasNoAnsi(t *testing.T) {
	var buf bytes.Buffer
	c := NewConsole(&buf, &buf, FormatJSON, false)
	res := report.Result{Tool: "forge", Version: "0.0", Command: "analyze", Findings: []report.Finding{{ID: "f1", Category: "project", Severity: report.SeverityInfo, Message: "ok"}}}
	if err := c.PrintResult(res); err != nil {
		t.Fatalf("PrintResult error: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
}
