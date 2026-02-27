package seed

import (
	"strings"
	"testing"
)

func TestMergeMarkedSection_EmptyExisting(t *testing.T) {
	content := []byte("# VectraGuard Instructions\nSome content here.\n")
	result, status := MergeMarkedSection(nil, content)

	if status != "written" {
		t.Errorf("expected status 'written', got %q", status)
	}
	if !strings.Contains(string(result), MarkerBegin) {
		t.Error("result should contain begin marker")
	}
	if !strings.Contains(string(result), MarkerEnd) {
		t.Error("result should contain end marker")
	}
	if !strings.Contains(string(result), "# VectraGuard Instructions") {
		t.Error("result should contain template content")
	}
}

func TestMergeMarkedSection_ExistingWithoutMarkers(t *testing.T) {
	existing := []byte("# My Custom Instructions\n\nDo not delete this.\n")
	content := []byte("# VectraGuard Instructions\nNew content.\n")

	result, status := MergeMarkedSection(existing, content)

	if status != "merged" {
		t.Errorf("expected status 'merged', got %q", status)
	}

	resultStr := string(result)

	// Original content should be preserved
	if !strings.Contains(resultStr, "# My Custom Instructions") {
		t.Error("result should preserve original content")
	}
	if !strings.Contains(resultStr, "Do not delete this.") {
		t.Error("result should preserve original content")
	}

	// New content should be appended with markers
	if !strings.Contains(resultStr, MarkerBegin) {
		t.Error("result should contain begin marker")
	}
	if !strings.Contains(resultStr, MarkerEnd) {
		t.Error("result should contain end marker")
	}
	if !strings.Contains(resultStr, "# VectraGuard Instructions") {
		t.Error("result should contain new content")
	}

	// Original content should come before markers
	beginIdx := strings.Index(resultStr, MarkerBegin)
	customIdx := strings.Index(resultStr, "# My Custom Instructions")
	if customIdx >= beginIdx {
		t.Error("original content should come before the begin marker")
	}
}

func TestMergeMarkedSection_ExistingWithMarkers(t *testing.T) {
	existing := []byte("# My Custom Instructions\n\nKeep this.\n\n" +
		MarkerBegin + "\n" +
		"# Old VectraGuard Content\nOld stuff.\n" +
		MarkerEnd + "\n\n" +
		"# Footer\nKeep this too.\n")

	content := []byte("# VectraGuard Instructions\nUpdated content.\n")

	result, status := MergeMarkedSection(existing, content)

	if status != "updated" {
		t.Errorf("expected status 'updated', got %q", status)
	}

	resultStr := string(result)

	// Original surrounding content preserved
	if !strings.Contains(resultStr, "# My Custom Instructions") {
		t.Error("result should preserve content before markers")
	}
	if !strings.Contains(resultStr, "Keep this.") {
		t.Error("result should preserve content before markers")
	}
	if !strings.Contains(resultStr, "# Footer") {
		t.Error("result should preserve content after markers")
	}
	if !strings.Contains(resultStr, "Keep this too.") {
		t.Error("result should preserve content after markers")
	}

	// Old content should be replaced
	if strings.Contains(resultStr, "Old VectraGuard Content") {
		t.Error("result should not contain old marked content")
	}
	if strings.Contains(resultStr, "Old stuff.") {
		t.Error("result should not contain old marked content")
	}

	// New content should be present
	if !strings.Contains(resultStr, "Updated content.") {
		t.Error("result should contain updated content")
	}
}

func TestMergeMarkedSection_BeginWithoutEnd(t *testing.T) {
	// Begin marker without end marker — treated as no markers (append)
	existing := []byte("# My Instructions\n\n" +
		MarkerBegin + "\n" +
		"# Orphaned section\n")

	content := []byte("# VectraGuard Instructions\nNew content.\n")

	result, status := MergeMarkedSection(existing, content)

	if status != "merged" {
		t.Errorf("expected status 'merged' for begin-only marker, got %q", status)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "# My Instructions") {
		t.Error("result should preserve original content")
	}
	// Should have two begin markers now (original orphaned + new)
	count := strings.Count(resultStr, MarkerBegin)
	if count < 2 {
		t.Errorf("expected at least 2 begin markers (orphaned + new), got %d", count)
	}
}

func TestMergeMarkedSection_Idempotency(t *testing.T) {
	content := []byte("# VectraGuard Instructions\nSome content.\n")

	// First merge into empty
	first, status1 := MergeMarkedSection(nil, content)
	if status1 != "written" {
		t.Errorf("first merge: expected 'written', got %q", status1)
	}

	// Second merge (update) should produce same result
	second, status2 := MergeMarkedSection(first, content)
	if status2 != "updated" {
		t.Errorf("second merge: expected 'updated', got %q", status2)
	}

	// Third merge should also produce same result as second
	third, status3 := MergeMarkedSection(second, content)
	if status3 != "updated" {
		t.Errorf("third merge: expected 'updated', got %q", status3)
	}

	if string(second) != string(third) {
		t.Error("merge should be idempotent: second and third results differ")
	}
}

func TestMergeMarkedSection_EndBeforeBegin(t *testing.T) {
	// End marker before begin marker — treated as no valid pair (append)
	existing := []byte("# My Instructions\n" +
		MarkerEnd + "\n" +
		"Some text\n" +
		MarkerBegin + "\n")

	content := []byte("# VectraGuard\n")

	_, status := MergeMarkedSection(existing, content)

	if status != "merged" {
		t.Errorf("expected status 'merged' for reversed markers, got %q", status)
	}
}

func TestMergeMarkedSection_EmptyNewContent(t *testing.T) {
	existing := []byte("# Existing\n")
	content := []byte("")

	result, status := MergeMarkedSection(existing, content)

	if status != "merged" {
		t.Errorf("expected status 'merged', got %q", status)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "# Existing") {
		t.Error("result should preserve existing content")
	}
	if !strings.Contains(resultStr, MarkerBegin) {
		t.Error("result should contain markers even with empty content")
	}
}
