// Package corpus provides a small, curated benchmark corpus of malicious and
// benign prompts used to measure the precision/recall of the promptfw detector
// suite over time.
//
// The corpus is intentionally small (50 malicious + 50 benign) so it can be
// checked into the repo, reviewed by humans, and diffed across PRs. It is NOT
// a replacement for a full evaluation harness — it is a regression safety net.
//
// Sources (all public):
//   - OWASP LLM Top 10 (2024) — injection / jailbreak examples
//   - promptfoo open benchmark (public prompt samples)
//   - Anthropic red-team disclosures
//   - Hand-written benign prompts reflecting typical developer use
package corpus

import (
	_ "embed"
	"strings"
)

//go:embed corpus_malicious.txt
var maliciousRaw string

//go:embed corpus_benign.txt
var benignRaw string

// Entry is a single corpus prompt.
type Entry struct {
	Label  string // "malicious" or "benign"
	Prompt string
}

// Malicious returns the bundled malicious prompt corpus.
func Malicious() []Entry {
	return parse(maliciousRaw, "malicious")
}

// Benign returns the bundled benign prompt corpus.
func Benign() []Entry {
	return parse(benignRaw, "benign")
}

// All returns the concatenation of Malicious() then Benign().
func All() []Entry {
	return append(Malicious(), Benign()...)
}

func parse(raw, label string) []Entry {
	var out []Entry
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, Entry{Label: label, Prompt: line})
	}
	return out
}
