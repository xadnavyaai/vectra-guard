package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
)

func runTrustList(ctx context.Context) error {
	cfg := config.FromContext(ctx)
	
	trustStore, err := sandbox.NewTrustStore(cfg.Sandbox.TrustStorePath)
	if err != nil {
		return fmt.Errorf("open trust store: %w", err)
	}
	
	entries := trustStore.List()
	
	if len(entries) == 0 {
		fmt.Println("No trusted commands found.")
		return nil
	}
	
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMAND\tAPPROVED\tUSE COUNT\tLAST USED\tEXPIRES")
	
	for _, entry := range entries {
		expires := "Never"
		if !entry.ExpiresAt.IsZero() {
			expires = entry.ExpiresAt.Format("2006-01-02")
		}
		
		lastUsed := "Never"
		if !entry.LastUsed.IsZero() {
			lastUsed = entry.LastUsed.Format("2006-01-02 15:04")
		}
		
		cmd := entry.Command
		if len(cmd) > 50 {
			cmd = cmd[:47] + "..."
		}
		
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			cmd,
			entry.ApprovedAt.Format("2006-01-02"),
			entry.UseCount,
			lastUsed,
			expires,
		)
	}
	
	w.Flush()
	return nil
}

func runTrustAdd(ctx context.Context, command, note, durationStr string) error {
	cfg := config.FromContext(ctx)
	
	trustStore, err := sandbox.NewTrustStore(cfg.Sandbox.TrustStorePath)
	if err != nil {
		return fmt.Errorf("open trust store: %w", err)
	}
	
	var duration time.Duration
	if durationStr != "" {
		duration, err = time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
	}
	
	if err := trustStore.Add(command, duration, note); err != nil {
		return fmt.Errorf("add command: %w", err)
	}
	
	fmt.Printf("✅ Added command to trust store: %s\n", command)
	return nil
}

func runTrustRemove(ctx context.Context, command string) error {
	cfg := config.FromContext(ctx)
	
	trustStore, err := sandbox.NewTrustStore(cfg.Sandbox.TrustStorePath)
	if err != nil {
		return fmt.Errorf("open trust store: %w", err)
	}
	
	if err := trustStore.Remove(command); err != nil {
		return fmt.Errorf("remove command: %w", err)
	}
	
	fmt.Printf("✅ Removed command from trust store: %s\n", command)
	return nil
}

func runTrustClean(ctx context.Context) error {
	cfg := config.FromContext(ctx)
	
	trustStore, err := sandbox.NewTrustStore(cfg.Sandbox.TrustStorePath)
	if err != nil {
		return fmt.Errorf("open trust store: %w", err)
	}
	
	if err := trustStore.CleanExpired(); err != nil {
		return fmt.Errorf("clean expired: %w", err)
	}
	
	fmt.Println("✅ Cleaned expired entries from trust store")
	return nil
}

