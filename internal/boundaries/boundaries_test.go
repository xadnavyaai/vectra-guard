package boundaries

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestScanPath_Testdata(t *testing.T) {
	findings, err := ScanPath("testdata", Options{})
	if err != nil {
		t.Fatalf("ScanPath testdata failed: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected boundary findings in testdata, got 0")
	}

	// Verify at least one finding for every supported language.
	wantLangs := []string{"rust", "go", "python", "node", "java", "c"}
	seen := map[string]bool{}
	for _, f := range findings {
		seen[f.Language] = true
	}
	for _, lang := range wantLangs {
		if !seen[lang] {
			t.Errorf("expected at least one %s finding, got none. findings=%+v", lang, findings)
		}
	}
}

func TestScanPath_RustUnsafeFindings(t *testing.T) {
	findings, err := ScanPath("testdata/sample.rs", Options{})
	if err != nil {
		t.Fatalf("ScanPath rust failed: %v", err)
	}
	want := map[string]bool{
		"RUST_UNSAFE_BLOCK": false,
		"RUST_UNSAFE_FN":    false,
		"RUST_UNSAFE_IMPL":  false,
		"RUST_UNSAFE_TRAIT": false,
	}
	for _, f := range findings {
		if _, ok := want[f.Code]; ok {
			want[f.Code] = true
		}
	}
	for code, hit := range want {
		if !hit {
			t.Errorf("expected %s in rust findings, not found", code)
		}
	}
}

func TestScanPath_GoCgoAndUnsafePointer(t *testing.T) {
	findings, err := ScanPath("testdata/sample_cgo.go", Options{})
	if err != nil {
		t.Fatalf("ScanPath go failed: %v", err)
	}
	codes := map[string]int{}
	for _, f := range findings {
		codes[f.Code]++
	}
	if codes["GO_CGO_IMPORT"] == 0 {
		t.Errorf("expected GO_CGO_IMPORT, got %+v", codes)
	}
	if codes["GO_UNSAFE_POINTER"] == 0 {
		t.Errorf("expected GO_UNSAFE_POINTER, got %+v", codes)
	}
	if codes["GO_REFLECT_HEADER"] == 0 {
		t.Errorf("expected GO_REFLECT_HEADER, got %+v", codes)
	}
}

func TestScanPath_PythonCtypes(t *testing.T) {
	findings, err := ScanPath("testdata/sample_ctypes.py", Options{})
	if err != nil {
		t.Fatalf("ScanPath python failed: %v", err)
	}
	codes := map[string]int{}
	for _, f := range findings {
		codes[f.Code]++
	}
	if codes["PY_CTYPES_IMPORT"] == 0 {
		t.Errorf("expected PY_CTYPES_IMPORT, got %+v", codes)
	}
	if codes["PY_CTYPES_CDLL"] == 0 {
		t.Errorf("expected PY_CTYPES_CDLL, got %+v", codes)
	}
	if codes["PY_CFFI_IMPORT"] == 0 {
		t.Errorf("expected PY_CFFI_IMPORT, got %+v", codes)
	}
}

func TestScanPath_NodeFFI(t *testing.T) {
	findings, err := ScanPath("testdata/sample_ffi.js", Options{})
	if err != nil {
		t.Fatalf("ScanPath node failed: %v", err)
	}
	codes := map[string]int{}
	for _, f := range findings {
		codes[f.Code]++
	}
	if codes["NODE_FFI_NAPI"] == 0 {
		t.Errorf("expected NODE_FFI_NAPI, got %+v", codes)
	}
}

func TestScanPath_JavaNative(t *testing.T) {
	findings, err := ScanPath("testdata/Sample.java", Options{})
	if err != nil {
		t.Fatalf("ScanPath java failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Code == "JAVA_JNI_NATIVE" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected JAVA_JNI_NATIVE in findings, got %+v", findings)
	}
}

func TestScanPath_CDlsym(t *testing.T) {
	findings, err := ScanPath("testdata/sample_dlsym.c", Options{})
	if err != nil {
		t.Fatalf("ScanPath c failed: %v", err)
	}
	codes := map[string]int{}
	for _, f := range findings {
		codes[f.Code]++
	}
	if codes["C_DLSYM"] == 0 {
		t.Errorf("expected C_DLSYM, got %+v", codes)
	}
	if codes["C_DLOPEN"] == 0 {
		t.Errorf("expected C_DLOPEN, got %+v", codes)
	}
}

func TestScanPath_LanguageFilter(t *testing.T) {
	opts := Options{Languages: map[string]bool{"rust": true}}
	findings, err := ScanPath("testdata", opts)
	if err != nil {
		t.Fatalf("ScanPath filtered failed: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one rust finding with filter")
	}
	for _, f := range findings {
		if f.Language != "rust" {
			t.Errorf("filter leak: got %s finding", f.Language)
		}
	}
}

func TestScanPath_CommentsIgnored(t *testing.T) {
	// Commented-out unsafe should not be reported.
	// Use scanRust directly to avoid fixture churn.
	commented := "// unsafe { let p = 0; }"
	findings := scanRust("fake.rs", commented, 1)
	if len(findings) != 0 {
		t.Errorf("expected no findings from commented line, got %+v", findings)
	}
}

// --- New: hermetic fixture tests using t.TempDir() ---

// writeFixture writes a fixture file inside a test-owned temp dir. It
// only ever writes to a directory returned by t.TempDir(), which Go
// auto-cleans after the test exits. Never writes outside the temp tree.
func writeFixture(t *testing.T, dir, name, body string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", full, err)
	}
	return full
}

func TestScanPath_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	findings, err := ScanPath(dir, Options{})
	if err != nil {
		t.Fatalf("ScanPath empty dir: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings in empty dir, got %d: %+v", len(findings), findings)
	}
}

func TestScanPath_NonexistentPath(t *testing.T) {
	_, err := ScanPath(filepath.Join(t.TempDir(), "does-not-exist"), Options{})
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestScanPath_SkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	// Skipped directories per ScanPath: .git, node_modules, vendor,
	// .venv, venv, dist, build, target.
	for _, skip := range []string{".git", "node_modules", "vendor", ".venv", "venv", "dist", "build", "target"} {
		writeFixture(t, root, filepath.Join(skip, "hit.rs"), "fn x() { unsafe { let _ = 1; } }\n")
	}
	// This one should be picked up.
	writeFixture(t, root, filepath.Join("src", "lib.rs"), "fn x() { unsafe { let _ = 1; } }\n")

	findings, err := ScanPath(root, Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings from src/lib.rs")
	}
	for _, f := range findings {
		for _, skip := range []string{".git", "node_modules", "vendor", ".venv", "venv", "dist", "build", "target"} {
			if hasPathSegment(f.File, skip) {
				t.Errorf("scanner entered skipped dir %s: file=%s", skip, f.File)
			}
		}
	}
}

func hasPathSegment(p, seg string) bool {
	parts := splitPath(p)
	for _, part := range parts {
		if part == seg {
			return true
		}
	}
	return false
}

func splitPath(p string) []string {
	// Handle both OS separators for Windows safety.
	sep := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		p = filepath.ToSlash(p)
		sep = "/"
	}
	var out []string
	start := 0
	for i := 0; i < len(p); i++ {
		if string(p[i]) == sep {
			if i > start {
				out = append(out, p[start:i])
			}
			start = i + 1
		}
	}
	if start < len(p) {
		out = append(out, p[start:])
	}
	return out
}

func TestScanPath_NonScannableExtensions(t *testing.T) {
	root := t.TempDir()
	// These extensions should be ignored entirely.
	for _, name := range []string{"README.md", "config.yaml", "data.json", "notes.txt", "image.png"} {
		writeFixture(t, root, name, "unsafe { } ctypes.CDLL( dlsym(")
	}
	findings, err := ScanPath(root, Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for non-scannable files, got %+v", findings)
	}
}

func TestLanguageForPath_AllExtensions(t *testing.T) {
	tests := map[string]string{
		"foo.rs":    "rust",
		"main.go":   "go",
		"script.py": "python",
		"a.js":      "node",
		"b.mjs":     "node",
		"c.cjs":     "node",
		"d.ts":      "node",
		"Main.java": "java",
		"f.c":       "c",
		"f.cc":      "c",
		"f.cpp":     "c",
		"f.cxx":     "c",
		"f.h":       "c",
		"f.hpp":     "c",
		"README.md": "",
		"no-ext":    "",
	}
	for path, want := range tests {
		got := languageForPath(path)
		if got != want {
			t.Errorf("languageForPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestScanRust_AllDetectors(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"unsafe { *ptr = 0; }", "RUST_UNSAFE_BLOCK"},
		{"pub unsafe fn frobnicate() {}", "RUST_UNSAFE_FN"},
		{"unsafe impl Send for Foo {}", "RUST_UNSAFE_IMPL"},
		{"unsafe trait MyTrait {}", "RUST_UNSAFE_TRAIT"},
	}
	for _, tc := range tests {
		got := scanRust("f.rs", tc.line, 1)
		if len(got) == 0 || got[0].Code != tc.want {
			t.Errorf("scanRust(%q) = %+v, want %s", tc.line, got, tc.want)
		}
		if len(got) > 0 && got[0].Severity != "high" {
			t.Errorf("scanRust(%q) severity = %q, want high", tc.line, got[0].Severity)
		}
	}
}

func TestScanGo_ReflectStringHeader(t *testing.T) {
	got := scanGo("f.go", "_ = (*reflect.StringHeader)(unsafe.Pointer(&s))", 1)
	codes := map[string]bool{}
	for _, f := range got {
		codes[f.Code] = true
	}
	if !codes["GO_UNSAFE_POINTER"] {
		t.Error("expected GO_UNSAFE_POINTER")
	}
	if !codes["GO_REFLECT_HEADER"] {
		t.Error("expected GO_REFLECT_HEADER for StringHeader")
	}
}

func TestScanC_MmapSbrkAndCustomAlloc(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"void *p = mmap(NULL, 4096, PROT_READ, MAP_PRIVATE, -1, 0);", "C_MANUAL_MAPPING"},
		{"void *p = sbrk(0);", "C_MANUAL_MAPPING"},
		{"void *p = arena_alloc(64);", "C_CUSTOM_ALLOCATOR"},
		{"void *p = custom_malloc(16);", "C_CUSTOM_ALLOCATOR"},
		{"void *p = my_malloc(16);", "C_CUSTOM_ALLOCATOR"},
		{"void *p = bump_alloc(16);", "C_CUSTOM_ALLOCATOR"},
	}
	for _, tc := range tests {
		got := scanC("f.c", tc.line, 1)
		found := false
		for _, f := range got {
			if f.Code == tc.want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("scanC(%q) did not emit %s, got %+v", tc.line, tc.want, got)
		}
	}
}

func TestScanNode_BindingsRequire(t *testing.T) {
	got := scanNode("a.js", "const addon = require('bindings')('native');", 1)
	if len(got) == 0 || got[0].Code != "NODE_NAPI_BINDINGS" {
		t.Errorf("expected NODE_NAPI_BINDINGS, got %+v", got)
	}
}

func TestScanNode_ImportFromFfiNapi(t *testing.T) {
	got := scanNode("a.ts", `import ffi from "ffi-napi";`, 1)
	if len(got) == 0 || got[0].Code != "NODE_FFI_NAPI" {
		t.Errorf("expected NODE_FFI_NAPI for ES import, got %+v", got)
	}
}

func TestScanPython_FromCtypesImport(t *testing.T) {
	got := scanPython("a.py", "from ctypes import CDLL, c_int", 1)
	if len(got) == 0 {
		t.Fatal("expected PY_CTYPES_IMPORT for 'from ctypes import …'")
	}
	if got[0].Code != "PY_CTYPES_IMPORT" {
		t.Errorf("expected PY_CTYPES_IMPORT, got %+v", got)
	}
}

func TestScanJava_NativeMethodSignatures(t *testing.T) {
	tests := []string{
		"public native int foo();",
		"private static native void bar(long handle);",
		"protected native long[] baz(String s);",
	}
	for _, line := range tests {
		got := scanJava("A.java", line, 1)
		if len(got) == 0 || got[0].Code != "JAVA_JNI_NATIVE" {
			t.Errorf("expected JAVA_JNI_NATIVE on %q, got %+v", line, got)
		}
	}
}

func TestScanPath_FindingFieldsPopulated(t *testing.T) {
	findings, err := ScanPath("testdata/sample.rs", Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	for _, f := range findings {
		if f.File == "" {
			t.Errorf("empty File: %+v", f)
		}
		if f.Line <= 0 {
			t.Errorf("non-positive Line: %+v", f)
		}
		if f.Language == "" {
			t.Errorf("empty Language: %+v", f)
		}
		if f.Severity == "" {
			t.Errorf("empty Severity: %+v", f)
		}
		if f.Code == "" {
			t.Errorf("empty Code: %+v", f)
		}
		if f.Description == "" {
			t.Errorf("empty Description: %+v", f)
		}
	}
}

func TestScanPath_CommentSkipPerLanguage(t *testing.T) {
	tests := []struct {
		lang string
		line string
		scan func(string, string, int) []Finding
	}{
		{"rust", "// unsafe { }", scanRust},
		{"rust", "/* unsafe fn foo() */", scanRust},
		{"go", "// import \"C\"", scanGo},
		{"python", "# import ctypes", scanPython},
		{"node", "// require('ffi-napi')", scanNode},
		{"java", "// native int foo();", scanJava},
		{"c", "// dlsym(handle, \"sym\")", scanC},
	}
	for _, tc := range tests {
		got := tc.scan("f", tc.line, 1)
		if len(got) != 0 {
			t.Errorf("%s scanner returned findings for commented line %q: %+v", tc.lang, tc.line, got)
		}
	}
}

func TestScanPath_DoesNotModifyFilesystem(t *testing.T) {
	// Safety test: scanner must be read-only. We take a snapshot of
	// testdata, run the scanner, and compare.
	before := snapshotDir(t, "testdata")
	if _, err := ScanPath("testdata", Options{}); err != nil {
		t.Fatalf("scan: %v", err)
	}
	after := snapshotDir(t, "testdata")
	if len(before) != len(after) {
		t.Fatalf("fixture count changed: before=%d after=%d", len(before), len(after))
	}
	for path, beforeSize := range before {
		afterSize, ok := after[path]
		if !ok {
			t.Errorf("file %q disappeared after scan", path)
			continue
		}
		if beforeSize != afterSize {
			t.Errorf("file %q changed size: before=%d after=%d", path, beforeSize, afterSize)
		}
	}
}

func snapshotDir(t *testing.T, root string) map[string]int64 {
	t.Helper()
	snap := map[string]int64{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		snap[rel] = info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return snap
}

func TestScanPath_SingleFileTarget(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "only.rs", "fn main() { unsafe { let x = 1; } }\n")
	findings, err := ScanPath(path, Options{})
	if err != nil {
		t.Fatalf("ScanPath single file: %v", err)
	}
	if len(findings) == 0 {
		t.Error("expected at least one finding for single-file target")
	}
	for _, f := range findings {
		if f.File != path {
			t.Errorf("finding file = %q, want %q", f.File, path)
		}
	}
}

func TestScanPath_SingleFileNonScannable(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, "notes.md", "unsafe { dlsym( ctypes.CDLL(")
	findings, err := ScanPath(path, Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for .md target, got %+v", findings)
	}
}

func TestScanPath_LanguageFilterSkipsOtherLangs(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.rs", "fn main() { unsafe { let _ = 1; } }\n")
	writeFixture(t, root, "a.py", "import ctypes\n")
	writeFixture(t, root, "a.go", "import \"C\"\n")

	opts := Options{Languages: map[string]bool{"python": true}}
	findings, err := ScanPath(root, opts)
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected python findings")
	}
	for _, f := range findings {
		if f.Language != "python" {
			t.Errorf("filter leak: got language %q", f.Language)
		}
	}
}

func TestScanPath_LineNumbersAccurate(t *testing.T) {
	root := t.TempDir()
	body := "fn one() {}\nfn two() {}\nfn bad() { unsafe { } }\nfn four() {}\n"
	path := writeFixture(t, root, "x.rs", body)
	findings, err := ScanPath(path, Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected a finding")
	}
	if findings[0].Line != 3 {
		t.Errorf("line = %d, want 3", findings[0].Line)
	}
}

func TestScanPath_MultipleLanguagesStableOrderWithinFile(t *testing.T) {
	// Line numbers should be monotonically non-decreasing per file.
	findings, err := ScanPath("testdata", Options{})
	if err != nil {
		t.Fatalf("ScanPath: %v", err)
	}
	byFile := map[string][]int{}
	for _, f := range findings {
		byFile[f.File] = append(byFile[f.File], f.Line)
	}
	for file, lines := range byFile {
		sorted := make([]int, len(lines))
		copy(sorted, lines)
		sort.Ints(sorted)
		for i := range lines {
			if lines[i] != sorted[i] {
				t.Errorf("findings in %s not in line order: %v", file, lines)
				break
			}
		}
	}
}

func TestSupportedLangs_CoversAllDetectors(t *testing.T) {
	// Sanity: every entry in supportedLangs must be reachable from
	// languageForPath for at least one extension.
	reachable := map[string]bool{}
	for _, path := range []string{"a.rs", "a.go", "a.py", "a.js", "A.java", "a.c"} {
		lang := languageForPath(path)
		if lang != "" {
			reachable[lang] = true
		}
	}
	for _, lang := range supportedLangs {
		if !reachable[lang] {
			t.Errorf("language %q declared but not reachable from languageForPath", lang)
		}
	}
}
