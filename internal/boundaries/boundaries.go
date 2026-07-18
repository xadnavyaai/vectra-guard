// Package boundaries scans a repository for trust boundaries — places
// where a memory-safe language hands trust back to the developer, or
// where code crosses into foreign function interfaces (FFI). This is
// the surface where subtle, high-impact bugs tend to live.
//
// The scanner is intentionally narrow in v1: it only reports locations
// of boundary crossings. It does NOT try to find bugs. The output is a
// map of "places that require directed review" — nothing more.
package boundaries

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is a single boundary-crossing location. The shape intentionally
// mirrors internal/secscan.Finding so downstream tooling can treat both
// the same way.
type Finding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Language    string `json:"language"`
	Severity    string `json:"severity"` // low, medium, high
	Code        string `json:"code"`     // short identifier, e.g. RUST_UNSAFE_BLOCK
	Description string `json:"description"`
}

// Options controls how the scan is performed.
type Options struct {
	// Languages is an optional allowlist. If empty, all supported
	// languages are scanned.
	Languages map[string]bool
}

// supportedLangs is the list of languages boundaries can scan.
var supportedLangs = []string{"rust", "go", "python", "node", "java", "c"}

// ScanPath walks a directory tree and reports every trust boundary
// finding. Single-file targets are also supported.
func ScanPath(root string, opts Options) ([]Finding, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}

	if len(opts.Languages) == 0 {
		opts.Languages = make(map[string]bool, len(supportedLangs))
		for _, l := range supportedLangs {
			opts.Languages[l] = true
		}
	}

	if !info.IsDir() {
		lang := languageForPath(root)
		if lang == "" || !opts.Languages[lang] {
			return nil, nil
		}
		return scanFile(root, lang)
	}

	var findings []Finding
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == ".venv" || name == "venv" || name == "dist" ||
				name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		lang := languageForPath(path)
		if lang == "" || !opts.Languages[lang] {
			return nil
		}
		fs, err := scanFile(path, lang)
		if err != nil {
			return nil
		}
		findings = append(findings, fs...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return findings, nil
}

// languageForPath maps a file path to one of the supported language
// identifiers. Returns "" if the file is not scannable.
func languageForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".rs":
		return "rust"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs", ".ts":
		return "node"
	case ".java":
		return "java"
	case ".c", ".cc", ".cpp", ".cxx", ".h", ".hpp":
		return "c"
	}
	return ""
}

func scanFile(path, lang string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var findings []Finding
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		switch lang {
		case "rust":
			findings = append(findings, scanRust(path, line, lineNum)...)
		case "go":
			findings = append(findings, scanGo(path, line, lineNum)...)
		case "python":
			findings = append(findings, scanPython(path, line, lineNum)...)
		case "node":
			findings = append(findings, scanNode(path, line, lineNum)...)
		case "java":
			findings = append(findings, scanJava(path, line, lineNum)...)
		case "c":
			findings = append(findings, scanC(path, line, lineNum)...)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return findings, nil
}

// --- language detectors ---

var (
	rustUnsafeBlockRe = regexp.MustCompile(`\bunsafe\s*\{`)
	rustUnsafeFnRe    = regexp.MustCompile(`\bunsafe\s+fn\b`)
	rustUnsafeImplRe  = regexp.MustCompile(`\bunsafe\s+impl\b`)
	rustUnsafeTraitRe = regexp.MustCompile(`\bunsafe\s+trait\b`)

	goCgoImportRe    = regexp.MustCompile(`^\s*import\s+"C"`)
	goUnsafePtrRe    = regexp.MustCompile(`\bunsafe\.Pointer\b`)
	goSliceHeaderRe  = regexp.MustCompile(`\breflect\.SliceHeader\b`)
	goStringHeaderRe = regexp.MustCompile(`\breflect\.StringHeader\b`)

	pyCtypesImportRe = regexp.MustCompile(`^\s*(import\s+ctypes|from\s+ctypes\s+import)`)
	pyCtypesCDLLRe   = regexp.MustCompile(`\bctypes\.(CDLL|WinDLL|PyDLL|OleDLL)\(`)
	pyCffiRe         = regexp.MustCompile(`^\s*(import\s+cffi|from\s+cffi\s+import)`)

	nodeFfiNapiRe = regexp.MustCompile(`require\s*\(\s*['"](ffi-napi|ref-napi|node-ffi|node-ffi-napi)['"]\s*\)`)
	nodeImportFfi = regexp.MustCompile(`from\s+['"](ffi-napi|ref-napi|node-ffi|node-ffi-napi)['"]`)
	nodeNAPIRe    = regexp.MustCompile(`require\s*\(\s*['"]bindings['"]\s*\)`)

	javaNativeRe = regexp.MustCompile(`\bnative\s+\w[\w<>\[\],\s]*\s+\w+\s*\(`)

	cDlsymRe     = regexp.MustCompile(`\bdlsym\s*\(`)
	cDlopenRe    = regexp.MustCompile(`\bdlopen\s*\(`)
	cMmapRe      = regexp.MustCompile(`\bmmap\s*\(`)
	cSbrkRe      = regexp.MustCompile(`\bsbrk\s*\(`)
	cCustomAlloc = regexp.MustCompile(`\b(custom_malloc|my_malloc|arena_alloc|bump_alloc)\s*\(`)
)

func isRustComment(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*")
}

func scanRust(path, line string, lineNum int) []Finding {
	if isRustComment(line) {
		return nil
	}
	var out []Finding
	if rustUnsafeBlockRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "rust", Severity: "high",
			Code:        "RUST_UNSAFE_BLOCK",
			Description: "`unsafe { }` block — manual memory-safety contract. Audit for aliasing, lifetimes, and pointer validity.",
		})
	}
	if rustUnsafeFnRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "rust", Severity: "high",
			Code:        "RUST_UNSAFE_FN",
			Description: "`unsafe fn` declaration — callers must uphold safety invariants. Verify preconditions at every call site.",
		})
	}
	if rustUnsafeImplRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "rust", Severity: "high",
			Code:        "RUST_UNSAFE_IMPL",
			Description: "`unsafe impl` — usually Send/Sync for an otherwise non-thread-safe type. Data-race risk.",
		})
	}
	if rustUnsafeTraitRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "rust", Severity: "high",
			Code:        "RUST_UNSAFE_TRAIT",
			Description: "`unsafe trait` — implementors carry memory-safety obligations. Audit each implementation.",
		})
	}
	return out
}

func scanGo(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return nil
	}
	var out []Finding
	if goCgoImportRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "go", Severity: "high",
			Code:        "GO_CGO_IMPORT",
			Description: "`import \"C\"` — file crosses the cgo boundary. Every call to C is an FFI trust handoff.",
		})
	}
	if goUnsafePtrRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "go", Severity: "high",
			Code:        "GO_UNSAFE_POINTER",
			Description: "`unsafe.Pointer` — bypasses Go's type and memory-safety guarantees. Audit aliasing and GC interactions.",
		})
	}
	if goSliceHeaderRe.MatchString(line) || goStringHeaderRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "go", Severity: "high",
			Code:        "GO_REFLECT_HEADER",
			Description: "`reflect.SliceHeader`/`StringHeader` — manual slice/string reconstruction; GC-invisible memory.",
		})
	}
	return out
}

func scanPython(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return nil
	}
	var out []Finding
	if pyCtypesImportRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "python", Severity: "medium",
			Code:        "PY_CTYPES_IMPORT",
			Description: "`ctypes` import — Python crosses into native code. Validate all buffer sizes and pointer lifetimes.",
		})
	}
	if pyCtypesCDLLRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "python", Severity: "high",
			Code:        "PY_CTYPES_CDLL",
			Description: "`ctypes.CDLL(...)` / WinDLL / PyDLL load — loading a shared library is an FFI boundary.",
		})
	}
	if pyCffiRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "python", Severity: "medium",
			Code:        "PY_CFFI_IMPORT",
			Description: "`cffi` import — native FFI. Audit ABI bindings and buffer ownership.",
		})
	}
	return out
}

func scanNode(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return nil
	}
	var out []Finding
	if nodeFfiNapiRe.MatchString(line) || nodeImportFfi.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "node", Severity: "high",
			Code:        "NODE_FFI_NAPI",
			Description: "Node FFI (ffi-napi/ref-napi) — JavaScript into native code. Audit pointer lifetimes and buffer ownership.",
		})
	}
	if nodeNAPIRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "node", Severity: "medium",
			Code:        "NODE_NAPI_BINDINGS",
			Description: "`bindings('...')` — a Node native addon is loaded. The addon runs outside the JS sandbox.",
		})
	}
	return out
}

func scanJava(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return nil
	}
	var out []Finding
	if javaNativeRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "java", Severity: "high",
			Code:        "JAVA_JNI_NATIVE",
			Description: "`native` method declaration — JNI boundary. Implementation runs in C/C++ without JVM safety checks.",
		})
	}
	return out
}

func scanC(path, line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return nil
	}
	var out []Finding
	if cDlsymRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "c", Severity: "high",
			Code:        "C_DLSYM",
			Description: "`dlsym()` — dynamic symbol lookup. Caller assumes full responsibility for signature correctness.",
		})
	}
	if cDlopenRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "c", Severity: "medium",
			Code:        "C_DLOPEN",
			Description: "`dlopen()` — runtime shared library load; review the source and path of the loaded object.",
		})
	}
	if cMmapRe.MatchString(line) || cSbrkRe.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "c", Severity: "medium",
			Code:        "C_MANUAL_MAPPING",
			Description: "Manual memory mapping (mmap/sbrk); review for alignment, permissions, and lifetime.",
		})
	}
	if cCustomAlloc.MatchString(line) {
		out = append(out, Finding{
			File: path, Line: lineNum, Language: "c", Severity: "medium",
			Code:        "C_CUSTOM_ALLOCATOR",
			Description: "Custom allocator signature detected; custom allocators are common source of UAF/OOB bugs.",
		})
	}
	return out
}
