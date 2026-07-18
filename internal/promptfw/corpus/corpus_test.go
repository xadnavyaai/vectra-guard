package corpus

import (
	"strings"
	"testing"
)

func TestMalicious_NonEmpty(t *testing.T) {
	entries := Malicious()
	if len(entries) == 0 {
		t.Fatal("Malicious() returned empty")
	}
	if len(entries) < 30 {
		t.Errorf("expected at least 30 malicious entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.Label != "malicious" {
			t.Errorf("entries[%d].Label = %q, want malicious", i, e.Label)
		}
		if e.Prompt == "" {
			t.Errorf("entries[%d].Prompt empty", i)
		}
		if strings.HasPrefix(e.Prompt, "#") {
			t.Errorf("entries[%d] looks like a comment: %q", i, e.Prompt)
		}
	}
}

func TestBenign_NonEmpty(t *testing.T) {
	entries := Benign()
	if len(entries) == 0 {
		t.Fatal("Benign() returned empty")
	}
	if len(entries) < 30 {
		t.Errorf("expected at least 30 benign entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.Label != "benign" {
			t.Errorf("entries[%d].Label = %q, want benign", i, e.Label)
		}
		if e.Prompt == "" {
			t.Errorf("entries[%d].Prompt empty", i)
		}
	}
}

func TestAll_ConcatenatesMaliciousThenBenign(t *testing.T) {
	mal := Malicious()
	ben := Benign()
	all := All()
	if len(all) != len(mal)+len(ben) {
		t.Errorf("All() len = %d, want %d", len(all), len(mal)+len(ben))
	}
	// First len(mal) must be malicious, rest must be benign.
	for i := 0; i < len(mal); i++ {
		if all[i].Label != "malicious" {
			t.Errorf("all[%d].Label = %q, want malicious", i, all[i].Label)
		}
	}
	for i := len(mal); i < len(all); i++ {
		if all[i].Label != "benign" {
			t.Errorf("all[%d].Label = %q, want benign", i, all[i].Label)
		}
	}
}

func TestParse_StripsCommentsAndBlanks(t *testing.T) {
	raw := `# this is a comment

first prompt
# another comment
second prompt

	`
	got := parse(raw, "malicious")
	if len(got) != 2 {
		t.Fatalf("parse returned %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Prompt != "first prompt" {
		t.Errorf("got[0] = %q, want 'first prompt'", got[0].Prompt)
	}
	if got[1].Prompt != "second prompt" {
		t.Errorf("got[1] = %q, want 'second prompt'", got[1].Prompt)
	}
	for _, e := range got {
		if e.Label != "malicious" {
			t.Errorf("wrong label: %+v", e)
		}
	}
}

func TestParse_EmptyInput(t *testing.T) {
	got := parse("", "benign")
	if len(got) != 0 {
		t.Errorf("parse(empty) = %+v, want empty", got)
	}
}

func TestParse_OnlyCommentsAndBlanks(t *testing.T) {
	raw := "# only comment\n\n  \n# another\n"
	got := parse(raw, "benign")
	if len(got) != 0 {
		t.Errorf("expected empty, got %+v", got)
	}
}

func TestCorpus_NoDuplicateMaliciousPrompts(t *testing.T) {
	seen := map[string]int{}
	for _, e := range Malicious() {
		seen[e.Prompt]++
	}
	for prompt, count := range seen {
		if count > 1 {
			t.Errorf("duplicate malicious prompt (%dx): %q", count, prompt)
		}
	}
}

func TestCorpus_NoDuplicateBenignPrompts(t *testing.T) {
	seen := map[string]int{}
	for _, e := range Benign() {
		seen[e.Prompt]++
	}
	for prompt, count := range seen {
		if count > 1 {
			t.Errorf("duplicate benign prompt (%dx): %q", count, prompt)
		}
	}
}

func TestCorpus_DeterministicAcrossCalls(t *testing.T) {
	// Calling Malicious() twice should return equal slices (content).
	a := Malicious()
	b := Malicious()
	if len(a) != len(b) {
		t.Fatalf("non-deterministic length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("non-deterministic entry at %d: %+v vs %+v", i, a[i], b[i])
		}
	}
}
