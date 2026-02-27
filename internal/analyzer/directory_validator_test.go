package analyzer

import (
	"testing"
)

func TestValidateProtectedDirectory(t *testing.T) {
	protectedDirs := []string{
		"/",
		"/etc",
		"/usr",
		"/bin",
		"/sbin",
		"/var",
		"/opt",
		"/home",
		"/root",
	}

	tests := []struct {
		name           string
		command        string
		protectedDirs  []string
		expectedResult bool
		expectedDir    string
	}{
		// Root directory tests
		{
			name:           "rm -rf /",
			command:        "rm -rf /",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/",
		},
		{
			name:           "rm -r /",
			command:        "rm -r /",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/",
		},
		{
			name:           "rm -rf /*",
			command:        "rm -rf /*",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/",
		},
		{
			name:           "rm -r /*",
			command:        "rm -r /*",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/",
		},
		{
			name:           "find / -delete",
			command:        "find / -delete",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/",
		},

		// /etc directory tests
		{
			name:           "rm -rf /etc",
			command:        "rm -rf /etc",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "rm -rf /etc/",
			command:        "rm -rf /etc/",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "rm -rf /etc/passwd",
			command:        "rm -rf /etc/passwd",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "chmod -R 777 /etc",
			command:        "chmod -R 777 /etc",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "mv /etc/passwd /tmp",
			command:        "mv /etc/passwd /tmp",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},

		// /usr directory tests
		{
			name:           "rm -rf /usr",
			command:        "rm -rf /usr",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/usr",
		},
		{
			name:           "rm -rf /usr/bin",
			command:        "rm -rf /usr/bin",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/usr",
		},
		{
			name:           "chown -R root /usr",
			command:        "chown -R root /usr",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/usr",
		},

		// /bin directory tests
		{
			name:           "rm -rf /bin",
			command:        "rm -rf /bin",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/bin",
		},
		{
			name:           "rm /bin/bash",
			command:        "rm /bin/bash",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/bin",
		},

		// /var directory tests
		{
			name:           "rm -rf /var",
			command:        "rm -rf /var",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/var",
		},
		{
			name:           "rm -rf /var/log",
			command:        "rm -rf /var/log",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/var",
		},

		// /home directory tests
		{
			name:           "rm -rf /home",
			command:        "rm -rf /home",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/home",
		},
		{
			name:           "rm -rf /home/user",
			command:        "rm -rf /home/user",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/home",
		},

		// /root directory tests
		{
			name:           "rm -rf /root",
			command:        "rm -rf /root",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/root",
		},
		{
			name:           "chmod -R 777 /root",
			command:        "chmod -R 777 /root",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/root",
		},

		// Safe commands (should not be blocked)
		{
			name:           "rm -rf /tmp/test",
			command:        "rm -rf /tmp/test",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "rm -rf ./local",
			command:        "rm -rf ./local",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "rm -rf ~/test",
			command:        "rm -rf ~/test",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "echo test",
			command:        "echo test",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "ls /etc",
			command:        "ls /etc",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "cat /etc/passwd",
			command:        "cat /etc/passwd",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "grep pattern /etc/file",
			command:        "grep pattern /etc/file",
			protectedDirs:  protectedDirs,
			expectedResult: false,
			expectedDir:    "",
		},

		// Windows path tests (backslash)
		{
			name:           "rm -rf C:\\Windows (backslash)",
			command:        `rm -rf C:\Windows`,
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: true,
			expectedDir:    `c:\windows`,
		},
		{
			name:           "rm -rf C:\\Windows\\System32 (backslash subdir)",
			command:        `rm -rf C:\Windows\System32`,
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: true,
			expectedDir:    `c:\windows`,
		},
		{
			name:           "rm -rf C:\\Program Files (backslash)",
			command:        `rm -rf C:\Program Files`,
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: true,
			expectedDir:    `c:\program files`,
		},

		// Windows path tests (forward slash)
		{
			name:           "rm -rf C:/Windows (forward slash)",
			command:        "rm -rf C:/Windows",
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: true,
			expectedDir:    `c:\windows`,
		},
		{
			name:           "rm -rf c:/windows (lowercase)",
			command:        "rm -rf c:/windows",
			protectedDirs:  []string{`C:\Windows`},
			expectedResult: true,
			expectedDir:    `c:\windows`,
		},

		// Windows safe path (not protected)
		{
			name:           "rm -rf C:\\Users\\vikas\\tmp (safe Windows path)",
			command:        `rm -rf C:\Users\vikas\tmp`,
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: false,
			expectedDir:    "",
		},

		// Edge cases
		{
			name:           "rm -rf '/etc'",
			command:        "rm -rf '/etc'",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "rm -rf \"/etc\"",
			command:        "rm -rf \"/etc\"",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "rm -rf /etc/*",
			command:        "rm -rf /etc/*",
			protectedDirs:  protectedDirs,
			expectedResult: true,
			expectedDir:    "/etc",
		},
		{
			name:           "empty protected dirs",
			command:        "rm -rf /etc",
			protectedDirs:  []string{},
			expectedResult: false,
			expectedDir:    "",
		},
		{
			name:           "nil protected dirs",
			command:        "rm -rf /etc",
			protectedDirs:  nil,
			expectedResult: false,
			expectedDir:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, dir := ValidateProtectedDirectory(tt.command, tt.protectedDirs)
			if result != tt.expectedResult {
				t.Errorf("ValidateProtectedDirectory() result = %v, want %v", result, tt.expectedResult)
			}
			if result && dir != tt.expectedDir {
				t.Errorf("ValidateProtectedDirectory() dir = %v, want %v", dir, tt.expectedDir)
			}
		})
	}
}

func TestIsProtectedDirectory(t *testing.T) {
	protectedDirs := []string{
		"/",
		"/etc",
		"/usr",
		"/bin",
		"/home",
	}

	tests := []struct {
		name           string
		path           string
		protectedDirs  []string
		expectedResult bool
	}{
		{
			name:           "exact match /etc",
			path:           "/etc",
			protectedDirs:  protectedDirs,
			expectedResult: true,
		},
		{
			name:           "subdirectory /etc/passwd",
			path:           "/etc/passwd",
			protectedDirs:  protectedDirs,
			expectedResult: true,
		},
		{
			name:           "subdirectory /etc/ssh/config",
			path:           "/etc/ssh/config",
			protectedDirs:  protectedDirs,
			expectedResult: true,
		},
		{
			name:           "exact match /usr",
			path:           "/usr",
			protectedDirs:  protectedDirs,
			expectedResult: true,
		},
		{
			name:           "subdirectory /usr/bin",
			path:           "/usr/bin",
			protectedDirs:  protectedDirs,
			expectedResult: true,
		},
		{
			name:           "root directory",
			path:           "/",
			protectedDirs:  protectedDirs,
			expectedResult: true,
		},
		{
			name:           "any path under root",
			path:           "/tmp/test",
			protectedDirs:  protectedDirs,
			expectedResult: true, // Because / is protected
		},
		{
			name:           "safe path /tmp",
			path:           "/tmp",
			protectedDirs:  []string{"/etc", "/usr"}, // / not in list
			expectedResult: false,
		},
		{
			name:           "safe path /tmp/test",
			path:           "/tmp/test",
			protectedDirs:  []string{"/etc", "/usr"}, // / not in list
			expectedResult: false,
		},
		{
			name:           "relative path",
			path:           "./test",
			protectedDirs:  protectedDirs,
			expectedResult: false,
		},
		{
			name:           "home path",
			path:           "~/test",
			protectedDirs:  protectedDirs,
			expectedResult: false,
		},
		{
			name:           "empty path",
			path:           "",
			protectedDirs:  protectedDirs,
			expectedResult: false,
		},
		{
			name:           "empty protected dirs",
			path:           "/etc",
			protectedDirs:  []string{},
			expectedResult: false,
		},

		// Windows path tests
		{
			name:           "Windows exact match C:\\Windows",
			path:           `C:\Windows`,
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: true,
		},
		{
			name:           "Windows subdirectory C:\\Windows\\System32",
			path:           `C:\Windows\System32`,
			protectedDirs:  []string{`C:\Windows`},
			expectedResult: true,
		},
		{
			name:           "Windows forward slash C:/Windows",
			path:           "C:/Windows",
			protectedDirs:  []string{`C:\Windows`},
			expectedResult: true,
		},
		{
			name:           "Windows case insensitive",
			path:           `c:\windows`,
			protectedDirs:  []string{`C:\Windows`},
			expectedResult: true,
		},
		{
			name:           "Windows safe path",
			path:           `C:\Users\vikas\Documents`,
			protectedDirs:  []string{`C:\Windows`, `C:\Program Files`},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProtectedDirectory(tt.path, tt.protectedDirs)
			if result != tt.expectedResult {
				t.Errorf("IsProtectedDirectory() result = %v, want %v for path %s", result, tt.expectedResult, tt.path)
			}
		})
	}
}

func TestValidateProtectedDirectory_Extensive(t *testing.T) {
	// Test with comprehensive list of protected directories
	protectedDirs := []string{
		"/",
		"/bin",
		"/sbin",
		"/usr",
		"/usr/bin",
		"/usr/sbin",
		"/usr/local",
		"/etc",
		"/var",
		"/lib",
		"/lib64",
		"/opt",
		"/boot",
		"/root",
		"/sys",
		"/proc",
		"/dev",
		"/home",
		"/srv",
	}

	// Test various destructive operations on each protected directory
	destructiveOps := []string{
		"rm -rf",
		"rm -r",
		"chmod -R",
		"chown -R",
		"chgrp -R",
		"mv",
		"cp",
		"find",
		"tar",
		"dd",
	}

	for _, dir := range protectedDirs {
		if dir == "/" {
			continue // Skip root, tested separately
		}
		for _, op := range destructiveOps {
			command := op + " " + dir
			t.Run(command, func(t *testing.T) {
				result, foundDir := ValidateProtectedDirectory(command, protectedDirs)
				if !result {
					t.Errorf("Expected %s to be protected, but ValidateProtectedDirectory returned false", dir)
				}
				if foundDir != dir {
					t.Errorf("Expected found directory %s, got %s", dir, foundDir)
				}
			})
		}
	}
}
