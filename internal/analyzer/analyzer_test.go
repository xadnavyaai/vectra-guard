package analyzer

import (
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
)

func TestAnalyzeScriptDetectsCriticals(t *testing.T) {
	script := []byte("#!/usr/bin/env bash\nrm -rf /\n")
	findings := AnalyzeScript("danger.sh", script, config.PolicyConfig{})
	if len(findings) == 0 {
		t.Fatalf("expected findings, got none")
	}
}

func TestAllowlistSkipsLine(t *testing.T) {
	script := []byte("sudo echo allowed\n")
	policy := config.PolicyConfig{Allowlist: []string{"sudo echo"}}
	findings := AnalyzeScript("safe.sh", script, policy)
	for _, f := range findings {
		if f.Code == "SUDO_USAGE" {
			t.Fatalf("expected sudo usage to be skipped by allowlist")
		}
	}
}
