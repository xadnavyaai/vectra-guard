// Package behavioral builds a structured action profile for a session.
//
// The v1 scope is deliberately narrow: a closed taxonomy of action
// categories, a deterministic graph builder, and an adapter that reads
// existing session audit records. There is NO anomaly scoring, no
// spectral math, no alerts, and no new storage layer. This package
// produces a profile that can be diffed against other profiles for
// post-hoc review.
package behavioral

import (
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/session"
)

// ActionCategory is a closed enum of behavioral action categories.
// Every observed session event MUST map to exactly one of these.
type ActionCategory string

const (
	CatDataRead        ActionCategory = "data_read"
	CatDataWrite       ActionCategory = "data_write"
	CatFileAccess      ActionCategory = "file_access"
	CatCodeExec        ActionCategory = "code_exec"
	CatExternalAPI     ActionCategory = "external_api"
	CatNetworkCall     ActionCategory = "network_call"
	CatAuthAction      ActionCategory = "auth_action"
	CatInternalCompute ActionCategory = "internal_compute"
)

// AllCategories returns the closed set of categories, in a stable order.
// Consumers that need to iterate categories deterministically (e.g. for
// graph output) should use this function.
func AllCategories() []ActionCategory {
	return []ActionCategory{
		CatDataRead,
		CatDataWrite,
		CatFileAccess,
		CatCodeExec,
		CatExternalAPI,
		CatNetworkCall,
		CatAuthAction,
		CatInternalCompute,
	}
}

// TimedAction is the normalized input to the graph builder. It is a
// timestamp-ordered, already-categorized event stream.
type TimedAction struct {
	Timestamp time.Time      `json:"timestamp"`
	Category  ActionCategory `json:"category"`
	Source    string         `json:"source"` // "command" or "file_op"
	Label     string         `json:"label"`  // short human-readable tag
}

// CategorizeFileOp maps a session.FileOperation to an ActionCategory.
// `read` → file_access; `create`/`modify`/`delete` → data_write.
// Anything unknown falls through to file_access (the safest default
// for the "I touched the filesystem" signal).
func CategorizeFileOp(op session.FileOperation) ActionCategory {
	switch strings.ToLower(strings.TrimSpace(op.Operation)) {
	case "read":
		return CatFileAccess
	case "create", "modify", "delete", "write":
		return CatDataWrite
	}
	return CatFileAccess
}

// CategorizeCommand maps a session.Command to an ActionCategory by
// looking at the binary and (lightly) its arguments. The mapping is
// intentionally rule-based and fixed in this file so that every future
// reader knows exactly where the taxonomy lives.
func CategorizeCommand(cmd session.Command) ActionCategory {
	name := strings.ToLower(strings.TrimSpace(cmd.Command))
	if name == "" {
		return CatInternalCompute
	}
	// Strip any path prefix: "/usr/bin/curl" → "curl".
	if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
		name = name[idx+1:]
	}

	// Auth / credential actions (checked first to beat the generic
	// "exec" bucket).
	switch name {
	case "ssh", "ssh-add", "ssh-keygen", "gpg", "gpg-agent", "keyring",
		"security", "keychain", "vault", "doppler", "op", "aws", "gcloud",
		"az", "kubectl", "kubelogin":
		return CatAuthAction
	}

	// Network / remote calls.
	switch name {
	case "curl", "wget", "http", "httpie", "nc", "netcat", "ncat", "dig",
		"host", "nslookup", "ping", "traceroute", "ssh-copy-id", "scp",
		"rsync", "ftp", "sftp":
		return CatNetworkCall
	}

	// External API clients / SDK CLIs (call into cloud or SaaS).
	switch name {
	case "gh", "glab", "hub", "stripe", "heroku", "fly", "netlify",
		"vercel", "supabase", "firebase", "openai", "anthropic":
		return CatExternalAPI
	}

	// Code execution (language runtimes, shells, package managers that
	// spawn arbitrary code during install).
	switch name {
	case "python", "python3", "py", "node", "deno", "bun", "ruby", "perl",
		"php", "lua", "go", "cargo", "rustc", "java", "javac", "gcc", "g++",
		"clang", "make", "cmake", "ninja", "bash", "sh", "zsh", "fish",
		"pwsh", "powershell", "npm", "pnpm", "yarn", "pip", "pip3", "pipx",
		"poetry", "pipenv", "gem", "bundler", "bundle":
		return CatCodeExec
	}

	// Filesystem read / write utilities.
	switch name {
	case "cat", "less", "more", "head", "tail", "grep", "rg", "ripgrep",
		"find", "fd", "awk", "sed", "cut", "wc", "sort", "uniq", "diff",
		"ls", "tree", "file", "stat", "du":
		return CatDataRead
	case "rm", "mv", "cp", "install", "mkdir", "rmdir", "touch", "chmod",
		"chown", "ln", "tar", "zip", "unzip", "gzip", "gunzip", "dd":
		return CatDataWrite
	}

	// Git — writes to the repo filesystem via a higher-level action.
	if name == "git" {
		if len(cmd.Args) > 0 {
			sub := strings.ToLower(cmd.Args[0])
			switch sub {
			case "clone", "pull", "fetch", "push", "remote":
				return CatNetworkCall
			case "log", "status", "diff", "show", "blame", "reflog",
				"branch", "rev-parse", "config":
				return CatDataRead
			}
		}
		return CatDataWrite
	}

	return CatInternalCompute
}
