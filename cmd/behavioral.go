package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/vectra-guard/vectra-guard/internal/behavioral"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runBehavioralProfile(ctx context.Context, sessionID, outputFormat, diffSessionID string) error {
	logger := logging.FromContext(ctx)

	if sessionID == "" {
		return fmt.Errorf("--session <id> is required")
	}

	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	baseActions, err := behavioral.LoadTimedActions(mgr, sessionID)
	if err != nil {
		return err
	}
	baseGraph := behavioral.BuildGraph(sessionID, baseActions)

	// --diff path: build both graphs, render the diff, return.
	if diffSessionID != "" {
		targetActions, err := behavioral.LoadTimedActions(mgr, diffSessionID)
		if err != nil {
			return err
		}
		targetGraph := behavioral.BuildGraph(diffSessionID, targetActions)
		diff := behavioral.DiffGraphs(baseGraph, targetGraph)

		switch outputFormat {
		case "json":
			data, err := diff.ToJSON()
			if err != nil {
				return fmt.Errorf("marshal diff: %w", err)
			}
			fmt.Println(string(data))
			return nil
		case "dot":
			return fmt.Errorf("--output dot is not supported with --diff (use json or text)")
		case "", "text":
			fmt.Print(diff.ToText())
			return nil
		default:
			return fmt.Errorf("unknown --output format %q (want text or json)", outputFormat)
		}
	}

	// Single-session path: print the graph.
	switch outputFormat {
	case "json":
		data, err := baseGraph.ToJSON()
		if err != nil {
			return fmt.Errorf("marshal graph: %w", err)
		}
		fmt.Println(string(data))
		return nil
	case "dot":
		fmt.Print(baseGraph.ToDOT())
		return nil
	case "", "text":
		return printBehavioralText(baseGraph)
	default:
		return fmt.Errorf("unknown --output format %q (want text, json, or dot)", outputFormat)
	}
}

func printBehavioralText(g *behavioral.SessionGraph) error {
	fmt.Printf("Behavioral profile for session %s\n", g.SessionID)
	fmt.Printf("Actions: %d\n", g.Actions)
	if g.StartTime != nil && g.EndTime != nil {
		fmt.Printf("Window:  %s → %s\n", g.StartTime.Format("15:04:05"), g.EndTime.Format("15:04:05"))
	}
	fmt.Println()

	if len(g.Nodes) == 0 {
		fmt.Println("No categorized actions found. Session may be empty.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tCOUNT")
	fmt.Fprintln(w, "--------\t-----")
	for _, n := range g.Nodes {
		fmt.Fprintf(w, "%s\t%d\n", n.Category, n.Count)
	}
	w.Flush()

	if len(g.Edges) > 0 {
		fmt.Println()
		w2 := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w2, "FROM\tTO\tCOUNT\tTOTAL GAP")
		fmt.Fprintln(w2, "----\t--\t-----\t---------")
		for _, e := range g.Edges {
			fmt.Fprintf(w2, "%s\t%s\t%d\t%s\n", e.From, e.To, e.Count, e.TotalGap)
		}
		w2.Flush()
	}

	return nil
}
