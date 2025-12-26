package analyzer

import (
	"regexp"
	"strings"
)

// extractPythonCommands extracts shell commands from Python code
// Handles os.system(), subprocess.*, os.popen(), etc.
func extractPythonCommands(pythonCode string) []string {
	var commands []string
	
	// Normalize the code (handle escaped quotes, etc.)
	normalized := normalizePythonCode(pythonCode)
	
	// Extract from os.system("command")
	commands = append(commands, extractOsSystemCommands(normalized)...)
	
	// Extract from subprocess.call/run/Popen(["cmd", "args"])
	commands = append(commands, extractSubprocessCommands(normalized)...)
	
	// Extract from os.popen("command")
	commands = append(commands, extractOsPopenCommands(normalized)...)
	
	// Extract from eval() or exec() with string literals
	commands = append(commands, extractEvalExecCommands(normalized)...)
	
	return deduplicateCommands(commands)
}

// normalizePythonCode handles escaped quotes and common Python string formats
func normalizePythonCode(code string) string {
	// Replace escaped quotes with placeholders temporarily
	code = strings.ReplaceAll(code, "\\'", "'ESCAPED_SINGLE_QUOTE'")
	code = strings.ReplaceAll(code, "\\\"", "\"ESCAPED_DOUBLE_QUOTE\"")
	return code
}

// extractOsSystemCommands extracts commands from os.system() calls
func extractOsSystemCommands(code string) []string {
	var commands []string
	
	// Pattern: os.system("command") or os.system('command')
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`os\.system\s*\(\s*["']([^"']+)["']`),
		regexp.MustCompile(`os\.system\s*\(\s*"""([^"]+)"""`),
		regexp.MustCompile(`os\.system\s*\(\s*'''([^']+)'''`),
	}
	
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				cmd := restoreEscapedQuotes(match[1])
				commands = append(commands, cmd)
			}
		}
	}
	
	return commands
}

// extractSubprocessCommands extracts commands from subprocess.* calls
func extractSubprocessCommands(code string) []string {
	var commands []string
	
	// Pattern: subprocess.call/run/Popen(["cmd", "arg1", "arg2"])
	// Also handles: subprocess.call(["cmd", "arg"], shell=True)
	// Use non-greedy matching for nested structures
	patterns := []*regexp.Regexp{
		// Array format: ["cmd", "arg1", "arg2"] - use non-greedy match
		regexp.MustCompile(`subprocess\.(call|run|Popen|check_call|check_output)\s*\(\s*\[(.*?)\]`),
		// Tuple format: ("cmd", "arg1", "arg2")
		regexp.MustCompile(`subprocess\.(call|run|Popen|check_call|check_output)\s*\(\s*\((.*?)\)`),
		// String format with shell=True: "cmd arg1 arg2"
		regexp.MustCompile(`subprocess\.(call|run|Popen|check_call|check_output)\s*\(\s*["']([^"']+)["'].*shell\s*=\s*True`),
	}
	
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 2 {
				argsStr := match[2]
				// Parse array/tuple elements
				args := parsePythonArray(argsStr)
				if len(args) > 0 {
					// Join arguments into a command string
					cmd := strings.Join(args, " ")
					commands = append(commands, cmd)
				}
			}
		}
	}
	
	return commands
}

// extractOsPopenCommands extracts commands from os.popen() calls
func extractOsPopenCommands(code string) []string {
	var commands []string
	
	// Pattern: os.popen("command")
	pattern := regexp.MustCompile(`os\.popen\s*\(\s*["']([^"']+)["']`)
	matches := pattern.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			cmd := restoreEscapedQuotes(match[1])
			commands = append(commands, cmd)
		}
	}
	
	return commands
}

// extractEvalExecCommands extracts commands from eval() or exec() calls with string literals
func extractEvalExecCommands(code string) []string {
	var commands []string
	
	// Pattern: eval("command") or exec("command")
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:eval|exec)\s*\(\s*["']([^"']+)["']`),
		regexp.MustCompile(`(?:eval|exec)\s*\(\s*"""([^"]+)"""`),
		regexp.MustCompile(`(?:eval|exec)\s*\(\s*'''([^']+)'''`),
	}
	
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				cmd := restoreEscapedQuotes(match[1])
				// Recursively check if the eval/exec contains more Python code
				if strings.Contains(cmd, "os.system") || strings.Contains(cmd, "subprocess") {
					commands = append(commands, extractPythonCommands(cmd)...)
				} else {
					commands = append(commands, cmd)
				}
			}
		}
	}
	
	return commands
}

// parsePythonArray parses a Python array/tuple string into individual elements
// Handles: ["cmd", "arg1", "arg2"] or ("cmd", "arg1", "arg2")
func parsePythonArray(arrayStr string) []string {
	var args []string
	
	// Remove brackets/parentheses if present
	arrayStr = strings.TrimSpace(arrayStr)
	arrayStr = strings.TrimPrefix(arrayStr, "[")
	arrayStr = strings.TrimSuffix(arrayStr, "]")
	arrayStr = strings.TrimPrefix(arrayStr, "(")
	arrayStr = strings.TrimSuffix(arrayStr, ")")
	arrayStr = strings.TrimSpace(arrayStr)
	
	if arrayStr == "" {
		return args
	}
	
	// Parse quoted strings properly - handle both single and double quotes
	// Use regex to extract quoted strings (more reliable than splitting)
	quotedPattern := regexp.MustCompile(`["']([^"']*)["']`)
	matches := quotedPattern.FindAllStringSubmatch(arrayStr, -1)
	
	if len(matches) > 0 {
		// Extract all quoted strings
		for _, match := range matches {
			if len(match) > 1 {
				arg := match[1]
				if arg != "" {
					args = append(args, arg)
				}
			}
		}
	} else {
		// Fallback: split by comma if no quotes found
		parts := strings.Split(arrayStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			part = strings.Trim(part, `"'`)
			if part != "" {
				args = append(args, part)
			}
		}
	}
	
	return args
}

// restoreEscapedQuotes restores escaped quotes in command strings
func restoreEscapedQuotes(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "'ESCAPED_SINGLE_QUOTE'", "'")
	cmd = strings.ReplaceAll(cmd, "\"ESCAPED_DOUBLE_QUOTE\"", "\"")
	return cmd
}

// deduplicateCommands removes duplicate commands
func deduplicateCommands(commands []string) []string {
	seen := make(map[string]bool)
	var unique []string
	
	for _, cmd := range commands {
		normalized := strings.TrimSpace(cmd)
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			unique = append(unique, normalized)
		}
	}
	
	return unique
}

// isPythonCommand checks if a line contains a Python command invocation
func isPythonCommand(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "python") && 
		(strings.Contains(lower, "-c") || strings.Contains(lower, "python -c"))
}

// extractPythonCodeFromCommand extracts the Python code from a command like:
// python -c 'import os; os.system("rm -rf /")'
func extractPythonCodeFromCommand(cmd string) string {
	// Find python -c '...' or python -c "..."
	// Handle nested quotes by finding the opening quote and matching to the closing quote
	// First, try to find python -c followed by a quote
	pythonPattern := regexp.MustCompile(`python\s+-c\s+(['"])`)
	match := pythonPattern.FindStringSubmatch(cmd)
	if len(match) < 2 {
		return ""
	}
	
	quote := match[1] // The quote character (' or ")
	startIdx := strings.Index(cmd, match[0]) + len(match[0])
	
	// Find the matching closing quote, handling escaped quotes
	var code strings.Builder
	escaped := false
	for i := startIdx; i < len(cmd); i++ {
		char := cmd[i]
		if escaped {
			code.WriteByte(char)
			escaped = false
			continue
		}
		if char == '\\' {
			escaped = true
			code.WriteByte(char)
			continue
		}
		if char == quote[0] {
			// Found closing quote
			break
		}
		code.WriteByte(char)
	}
	
	extracted := code.String()
	if extracted != "" {
		// Unescape quotes
		extracted = strings.ReplaceAll(extracted, "\\'", "'")
		extracted = strings.ReplaceAll(extracted, "\\\"", "\"")
		return extracted
	}
	
	// Fallback to regex for triple quotes
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`python\s+-c\s+'''([^']+)'''`),
		regexp.MustCompile(`python\s+-c\s+"""([^"]+)"""`),
	}
	
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(cmd)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

