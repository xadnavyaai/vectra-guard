package analyzer

import (
	"strings"
	"unicode"
)

// ValidateProtectedDirectory checks if a command targets a protected directory.
// Returns true if the directory is protected and should be blocked.
func ValidateProtectedDirectory(command string, protectedDirs []string) (bool, string) {
	if len(protectedDirs) == 0 {
		return false, ""
	}

	commandLower := strings.ToLower(command)

	// Extract potential paths from the command
	// Look for common destructive operations: rm, mv, cp, chmod, chown, etc.
	destructiveOps := []string{"rm ", "mv ", "cp ", "chmod ", "chown ", "chgrp ", "find ", "tar ", "dd "}
	hasDestructiveOp := false
	for _, op := range destructiveOps {
		// Check for operation followed by space or flag (to avoid false positives)
		if strings.Contains(commandLower, op) {
			// For tar and dd, they need specific patterns to be destructive
			if op == "tar " {
				// tar is only destructive if it's extracting/creating to protected dir
				// or if it has -C flag pointing to protected dir
				if strings.Contains(commandLower, "tar ") {
					hasDestructiveOp = true
				}
			} else if op == "dd " {
				// dd is only destructive if it has of= pointing to protected dir
				// For now, we'll check it, but it's less common
				if strings.Contains(commandLower, "dd ") {
					hasDestructiveOp = true
				}
			} else {
				hasDestructiveOp = true
			}
			if hasDestructiveOp {
				break
			}
		}
	}

	// If no destructive operation, it's safe
	if !hasDestructiveOp {
		return false, ""
	}

	// Tokenize the command so we can reason about individual arguments/paths.
	// This avoids brittle substring matching and ensures we can distinguish
	// between /usr and /usr/bin, etc.
	fields := strings.Fields(commandLower)

	// First, try to find the *most specific* protected directory that matches
	// any of the command arguments. This ensures that for nested directories
	// like /usr and /usr/bin we return /usr/bin when appropriate.
	bestMatch := ""

	for _, field := range fields {
		// Strip common quoting characters
		fieldTrimmed := strings.Trim(field, "\"'")
		if fieldTrimmed == "" {
			continue
		}

		isUnixAbs := strings.HasPrefix(fieldTrimmed, "/")
		isWinAbs := isWindowsAbsPath(fieldTrimmed)

		if !isUnixAbs && !isWinAbs {
			continue
		}

		var argPathLower string
		if isWinAbs {
			argPathLower = normalizeWindowsPath(fieldTrimmed)
		} else {
			// For Unix-style absolute paths, normalize manually to preserve the leading /
			argPath := strings.TrimRight(fieldTrimmed, "/")
			if argPath == "" {
				argPath = "/"
			}
			argPath = "/" + strings.TrimLeft(strings.ReplaceAll(argPath, "//", "/"), "/")
			argPathLower = strings.ToLower(argPath)
		}

		for _, protectedDir := range protectedDirs {
			if protectedDir == "" {
				continue
			}

			protectedDirLower := strings.ToLower(strings.TrimSpace(protectedDir))
			var protectedDirNormalized string

			if isWindowsAbsPath(protectedDirLower) {
				protectedDirNormalized = normalizeWindowsPath(protectedDirLower)
			} else if strings.HasPrefix(protectedDirLower, "/") {
				protectedDirNormalized = strings.TrimRight(protectedDirLower, "/")
				if protectedDirNormalized == "" {
					protectedDirNormalized = "/"
				} else {
					protectedDirNormalized = "/" + strings.TrimLeft(strings.ReplaceAll(protectedDirNormalized, "//", "/"), "/")
				}
			} else {
				protectedDirNormalized = "/" + strings.TrimLeft(protectedDirLower, "/")
			}

			// Special handling for root directory.
			if protectedDirNormalized == "/" {
				if argPathLower == "/" || argPathLower == "/*" {
					return true, "/"
				}
				continue
			}

			// Exact match: e.g. "rm -rf /etc" or "rm -rf c:\windows"
			if argPathLower == protectedDirNormalized {
				if len(protectedDirNormalized) > len(bestMatch) {
					bestMatch = protectedDirNormalized
				}
				continue
			}

			// Prefix match: e.g. "rm -rf /etc/passwd" or "rm -rf c:\windows\system32"
			sep := "/"
			if isWinAbs && isWindowsAbsPath(protectedDirLower) {
				sep = `\`
			}
			if strings.HasPrefix(argPathLower, protectedDirNormalized+sep) {
				if len(protectedDirNormalized) > len(bestMatch) {
					bestMatch = protectedDirNormalized
				}
				continue
			}

			// Wildcard-style patterns such as "/etc/*" or "c:\windows\*"
			if strings.HasPrefix(argPathLower, protectedDirNormalized+"/*") ||
				strings.HasPrefix(argPathLower, protectedDirNormalized+`\*`) {
				if len(protectedDirNormalized) > len(bestMatch) {
					bestMatch = protectedDirNormalized
				}
				continue
			}
		}
	}

	if bestMatch != "" {
		return true, bestMatch
	}

	// Fallback: direct substring match for Windows protected dirs whose paths
	// contain spaces (e.g. "C:\Program Files"). strings.Fields splits these
	// across tokens so the tokenised loop above won't reassemble them.
	for _, protectedDir := range protectedDirs {
		if protectedDir == "" {
			continue
		}
		protectedDirLower := strings.ToLower(strings.TrimSpace(protectedDir))
		if !isWindowsAbsPath(protectedDirLower) || !strings.Contains(protectedDirLower, " ") {
			continue
		}
		norm := normalizeWindowsPath(protectedDirLower)
		// Also build a forward-slash variant so we match both styles.
		normFwd := strings.ReplaceAll(norm, `\`, "/")

		if strings.Contains(commandLower, norm) || strings.Contains(commandLower, normFwd) {
			if len(norm) > len(bestMatch) {
				bestMatch = norm
			}
		}
	}

	if bestMatch != "" {
		return true, bestMatch
	}

	return false, ""
}

// isWindowsAbsPath returns true for paths like C:\, D:\foo, c:/bar.
func isWindowsAbsPath(p string) bool {
	if len(p) < 2 {
		return false
	}
	return unicode.IsLetter(rune(p[0])) && p[1] == ':' && (len(p) == 2 || p[2] == '\\' || p[2] == '/')
}

// normalizeWindowsPath lowercases and normalizes separators to backslash,
// stripping trailing separators.
func normalizeWindowsPath(p string) string {
	p = strings.ToLower(p)
	p = strings.ReplaceAll(p, "/", `\`)
	p = strings.TrimRight(p, `\`)
	// Ensure drive root keeps trailing backslash: "c:" → "c:\"
	if len(p) == 2 && p[1] == ':' {
		p += `\`
	}
	return p
}

// IsProtectedDirectory checks if a given path matches any protected directory.
// This is a helper function for more precise path matching.
func IsProtectedDirectory(path string, protectedDirs []string) bool {
	if len(protectedDirs) == 0 || path == "" {
		return false
	}

	pathTrimmed := strings.TrimSpace(path)

	// Skip relative paths and home paths - they're not protected
	if strings.HasPrefix(pathTrimmed, "./") || strings.HasPrefix(pathTrimmed, "../") ||
		strings.HasPrefix(pathTrimmed, "~/") || pathTrimmed == "~" {
		return false
	}

	pathLower := strings.ToLower(pathTrimmed)
	isWin := isWindowsAbsPath(pathLower)

	var pathNormalized string
	if isWin {
		pathNormalized = normalizeWindowsPath(pathLower)
	} else if strings.HasPrefix(pathLower, "/") {
		pathNormalized = strings.TrimRight(pathLower, "/")
		if pathNormalized == "" {
			pathNormalized = "/"
		} else {
			pathNormalized = "/" + strings.TrimLeft(strings.ReplaceAll(pathNormalized, "//", "/"), "/")
		}
	} else {
		return false
	}

	for _, protectedDir := range protectedDirs {
		if protectedDir == "" {
			continue
		}

		protectedDirLower := strings.ToLower(strings.TrimSpace(protectedDir))
		var protectedDirNormalized string
		protectedIsWin := isWindowsAbsPath(protectedDirLower)

		if protectedIsWin {
			protectedDirNormalized = normalizeWindowsPath(protectedDirLower)
		} else if strings.HasPrefix(protectedDirLower, "/") {
			protectedDirNormalized = strings.TrimRight(protectedDirLower, "/")
			if protectedDirNormalized == "" {
				protectedDirNormalized = "/"
			} else {
				protectedDirNormalized = "/" + strings.TrimLeft(strings.ReplaceAll(protectedDirNormalized, "//", "/"), "/")
			}
		} else {
			protectedDirNormalized = "/" + strings.TrimLeft(protectedDirLower, "/")
		}

		// Exact match
		if pathNormalized == protectedDirNormalized {
			return true
		}

		// Prefix match
		sep := "/"
		if isWin && protectedIsWin {
			sep = `\`
		}
		if strings.HasPrefix(pathNormalized, protectedDirNormalized+sep) {
			return true
		}

		// Special case: root directory
		if protectedDirNormalized == "/" {
			if pathNormalized != "/" && strings.HasPrefix(pathNormalized, "/") {
				return true
			}
		}
	}

	return false
}
