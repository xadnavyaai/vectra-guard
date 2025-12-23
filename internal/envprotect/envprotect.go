package envprotect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// SensitivePatterns are common patterns for sensitive environment variables
var SensitivePatterns = []string{
	"PASSWORD", "SECRET", "KEY", "TOKEN", "API_KEY",
	"AWS_SECRET", "AWS_ACCESS_KEY", "GITHUB_TOKEN", "SSH_KEY",
	"DB_PASSWORD", "DATABASE_URL", "PRIVATE_KEY", "AUTH_TOKEN",
	"CREDENTIALS", "CERT", "APIKEY", "ACCESS_KEY", "SESSION",
}

// MaskingMode determines how to mask sensitive values
type MaskingMode string

const (
	MaskFull    MaskingMode = "full"     // ********
	MaskPartial MaskingMode = "partial"  // abc***xyz
	MaskHash    MaskingMode = "hash"     // sha256:abc123...
	MaskFake    MaskingMode = "fake"     // Generated fake value
)

// EnvProtector handles environment variable protection
type EnvProtector struct {
	mode             MaskingMode
	protectedVars    map[string]bool
	fakeValues       map[string]string
	allowReadPatterns []string
}

// NewEnvProtector creates a new environment protector
func NewEnvProtector(mode MaskingMode) *EnvProtector {
	return &EnvProtector{
		mode:          mode,
		protectedVars: make(map[string]bool),
		fakeValues:    make(map[string]string),
		allowReadPatterns: []string{
			"HOME", "USER", "PATH", "SHELL", "TERM", "LANG",
			"PWD", "TMPDIR", "EDITOR", "PAGER",
		},
	}
}

// IsSensitive checks if an environment variable name is sensitive
func (ep *EnvProtector) IsSensitive(name string) bool {
	upper := strings.ToUpper(name)
	
	// Check if explicitly protected
	if ep.protectedVars[name] {
		return true
	}
	
	// Check against allow list first
	for _, allowed := range ep.allowReadPatterns {
		if upper == allowed {
			return false
		}
	}
	
	// Check against sensitive patterns
	for _, pattern := range SensitivePatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	
	return false
}

// MaskValue masks a sensitive value based on the masking mode
func (ep *EnvProtector) MaskValue(name, value string) string {
	if value == "" {
		return ""
	}
	
	switch ep.mode {
	case MaskFull:
		return "********"
	
	case MaskPartial:
		return maskPartial(value)
	
	case MaskHash:
		hash := sha256.Sum256([]byte(value))
		return "sha256:" + hex.EncodeToString(hash[:])[:16] + "..."
	
	case MaskFake:
		if fake, exists := ep.fakeValues[name]; exists {
			return fake
		}
		fake := generateFakeValue(name, value)
		ep.fakeValues[name] = fake
		return fake
	
	default:
		return "********"
	}
}

// maskPartial shows first and last few characters
func maskPartial(value string) string {
	length := len(value)
	if length <= 6 {
		return "***"
	}
	if length <= 12 {
		return value[:2] + "***" + value[length-2:]
	}
	return value[:4] + "***" + value[length-4:]
}

// generateFakeValue creates a realistic-looking fake value
func generateFakeValue(name, original string) string {
	upper := strings.ToUpper(name)
	length := len(original)
	
	// Generate appropriate fake based on type
	if strings.Contains(upper, "URL") || strings.Contains(upper, "ENDPOINT") {
		return "https://example.com/api/v1"
	}
	
	if strings.Contains(upper, "EMAIL") {
		return "user@example.com"
	}
	
	if strings.Contains(upper, "TOKEN") || strings.Contains(upper, "JWT") {
		return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.FAKE"
	}
	
	if strings.Contains(upper, "KEY") || strings.Contains(upper, "SECRET") {
		if length > 30 {
			return "sk_test_FAKE1234567890abcdefghijklmnopqrstuvwxyz"
		}
		return "fake_key_" + strings.Repeat("x", max(8, length-9))
	}
	
	if strings.Contains(upper, "PASSWORD") {
		return "FakeP@ssw0rd123"
	}
	
	if strings.Contains(upper, "PORT") {
		return "8080"
	}
	
	// Default fake value
	if length > 20 {
		return "fake_" + strings.Repeat("x", length-5)
	}
	return "fake_value"
}

// SanitizeEnvOutput masks sensitive environment variables in command output
func (ep *EnvProtector) SanitizeEnvOutput(output string) string {
	lines := strings.Split(output, "\n")
	sanitized := make([]string, 0, len(lines))
	
	for _, line := range lines {
		// Check if line looks like ENV_VAR=value
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := parts[0]
				value := parts[1]
				
				if ep.IsSensitive(name) {
					masked := ep.MaskValue(name, value)
					sanitized = append(sanitized, fmt.Sprintf("%s=%s [MASKED]", name, masked))
					continue
				}
			}
		}
		sanitized = append(sanitized, line)
	}
	
	return strings.Join(sanitized, "\n")
}

// GetSanitizedEnv returns a sanitized copy of environment variables
func (ep *EnvProtector) GetSanitizedEnv() map[string]string {
	result := make(map[string]string)
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		name := parts[0]
		value := parts[1]
		
		if ep.IsSensitive(name) {
			result[name] = ep.MaskValue(name, value)
		} else {
			result[name] = value
		}
	}
	
	return result
}

// AddProtectedVar explicitly marks a variable as protected
func (ep *EnvProtector) AddProtectedVar(name string) {
	ep.protectedVars[name] = true
}

// AddFakeValue sets a custom fake value for a variable
func (ep *EnvProtector) AddFakeValue(name, fakeValue string) {
	ep.fakeValues[name] = fakeValue
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

