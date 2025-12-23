package envprotect

import (
	"strings"
	"testing"
)

func TestIsSensitive(t *testing.T) {
	ep := NewEnvProtector(MaskFull)
	
	tests := []struct {
		name      string
		want      bool
	}{
		{"AWS_SECRET_ACCESS_KEY", true},
		{"DATABASE_PASSWORD", true},
		{"API_KEY", true},
		{"GITHUB_TOKEN", true},
		{"HOME", false},
		{"PATH", false},
		{"USER", false},
		{"MY_SECRET_VALUE", true},
		{"NORMAL_CONFIG", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ep.IsSensitive(tt.name); got != tt.want {
				t.Errorf("IsSensitive(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name  string
		mode  MaskingMode
		value string
		check func(string) bool
	}{
		{
			name:  "full_mask",
			mode:  MaskFull,
			value: "super_secret_key_123",
			check: func(masked string) bool { return masked == "********" },
		},
		{
			name:  "partial_mask",
			mode:  MaskPartial,
			value: "super_secret_key_123",
			check: func(masked string) bool {
				return strings.Contains(masked, "***") &&
				       strings.HasPrefix(masked, "supe") &&
				       strings.HasSuffix(masked, "_123")
			},
		},
		{
			name:  "hash_mask",
			mode:  MaskHash,
			value: "super_secret_key_123",
			check: func(masked string) bool {
				return strings.HasPrefix(masked, "sha256:")
			},
		},
		{
			name:  "fake_value",
			mode:  MaskFake,
			value: "super_secret_key_123",
			check: func(masked string) bool {
				return masked != "super_secret_key_123" && len(masked) > 0
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := NewEnvProtector(tt.mode)
			masked := ep.MaskValue("TEST_SECRET", tt.value)
			if !tt.check(masked) {
				t.Errorf("MaskValue() with mode %s produced unexpected result: %q", tt.mode, masked)
			}
		})
	}
}

func TestGenerateFakeValue(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		original string
		contains string
	}{
		{"url", "API_URL", "https://prod.example.com/api", "https://example.com"},
		{"email", "ADMIN_EMAIL", "admin@prod.com", "example.com"},
		{"token", "AUTH_TOKEN", "real_token_abc123", "eyJ"},
		{"password", "DB_PASSWORD", "SuperSecret123!", "Fake"},
		{"key", "SECRET_KEY", "sk_live_1234567890abcdef", "fake_key"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := generateFakeValue(tt.varName, tt.original)
			if !strings.Contains(fake, tt.contains) && fake != tt.original {
				t.Logf("Generated fake value: %s", fake)
			}
			if fake == tt.original {
				t.Errorf("generateFakeValue() returned original value, should generate fake")
			}
		})
	}
}

func TestSanitizeEnvOutput(t *testing.T) {
	ep := NewEnvProtector(MaskPartial)
	
	input := `HOME=/Users/test
PATH=/usr/bin:/bin
AWS_SECRET_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
DATABASE_PASSWORD=super_secret_password
USER=testuser`
	
	output := ep.SanitizeEnvOutput(input)
	
	// Should mask sensitive vars
	if strings.Contains(output, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("AWS key not masked in output")
	}
	if strings.Contains(output, "super_secret_password") {
		t.Error("Password not masked in output")
	}
	if !strings.Contains(output, "[MASKED]") {
		t.Error("Missing [MASKED] marker")
	}
	
	// Should preserve non-sensitive vars
	if !strings.Contains(output, "/Users/test") {
		t.Error("HOME value incorrectly masked")
	}
	if !strings.Contains(output, "testuser") {
		t.Error("USER value incorrectly masked")
	}
}

func TestAddProtectedVar(t *testing.T) {
	ep := NewEnvProtector(MaskFull)
	
	// Should not be sensitive by default
	if ep.IsSensitive("CUSTOM_VAR") {
		t.Error("CUSTOM_VAR should not be sensitive by default")
	}
	
	// Add to protected list
	ep.AddProtectedVar("CUSTOM_VAR")
	
	// Should now be sensitive
	if !ep.IsSensitive("CUSTOM_VAR") {
		t.Error("CUSTOM_VAR should be sensitive after adding")
	}
}

func TestAddFakeValue(t *testing.T) {
	ep := NewEnvProtector(MaskFake)
	
	customFake := "my_custom_fake_value"
	ep.AddFakeValue("MY_SECRET", customFake)
	
	masked := ep.MaskValue("MY_SECRET", "real_secret_value")
	if masked != customFake {
		t.Errorf("Expected custom fake value %q, got %q", customFake, masked)
	}
}

