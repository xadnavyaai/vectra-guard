package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/cve"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
	"github.com/vectra-guard/vectra-guard/internal/serve"
)

func init() {
	// Expose version to the serve package
	serve.SetVersionFunc(func() string { return Version })
}

func runServe(ctx context.Context, port int) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	workspace, err := serve.DefaultWorkspace()
	if err != nil {
		return err
	}

	// Load trust store
	trustPath := cfg.Sandbox.TrustStorePath
	if trustPath == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			trustPath = filepath.Join(homeDir, ".vectra-guard", "trust.json")
		}
	}
	trust, err := sandbox.NewTrustStore(trustPath)
	if err != nil {
		logger.Warn("failed to load trust store, continuing without", map[string]any{"error": err.Error()})
		trust = nil
	}

	// Load CVE store
	cveDir := cfg.CVE.CacheDir
	if cveDir == "" {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			cveDir = filepath.Join(homeDir, ".vectra-guard")
		}
	}
	var cveStore *cve.Store
	if cveDir != "" {
		cvePath := filepath.Join(cveDir, "cve-cache.json")
		cveStore, err = cve.LoadStore(cvePath)
		if err != nil {
			logger.Warn("failed to load CVE store, continuing without", map[string]any{"error": err.Error()})
			cveStore = nil
		}
	}

	srv, err := serve.New(workspace, logger, cfg, trust, cveStore)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Serving Vectra Guard dashboard at http://127.0.0.1:%d\n", port)
	return srv.ListenAndServe(port)
}

// parsePortEnv is a small helper for tests / future flags.
func parsePortEnv(envVal string, defaultPort int) int {
	if envVal == "" {
		return defaultPort
	}
	if p, err := strconv.Atoi(envVal); err == nil && p > 0 && p < 65536 {
		return p
	}
	return defaultPort
}
