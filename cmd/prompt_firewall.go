package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/promptfw"
	"github.com/vectra-guard/vectra-guard/internal/promptfw/corpus"
)

func runPromptFirewall(ctx context.Context, fromFile string, benchmark bool) error {
	logger := logging.FromContext(ctx)

	if benchmark {
		return runPromptFirewallBenchmark(ctx)
	}

	text, err := readPromptInput(fromFile)
	if err != nil {
		return err
	}

	result := promptfw.Analyze(text)

	fields := map[string]any{
		"risk_level": result.RiskLevel,
		"score":      fmt.Sprintf("%.2f", result.Score),
	}
	if len(result.Reasons) > 0 {
		fields["reasons"] = result.Reasons
	}

	switch result.RiskLevel {
	case "high":
		logger.Warn("prompt blocked: malicious or risky instructions detected", fields)
		return &exitError{message: "prompt blocked by firewall", code: 2}
	case "medium":
		logger.Warn("prompt flagged: potentially risky instructions", fields)
	default:
		logger.Info("prompt allowed", fields)
	}

	return nil
}

// runPromptFirewallBenchmark scores the bundled corpus and prints
// precision / recall / F1. A "positive" is any prompt classified as
// medium or high risk. TP = malicious flagged, FP = benign flagged,
// FN = malicious missed, TN = benign cleared.
func runPromptFirewallBenchmark(_ context.Context) error {
	malicious := corpus.Malicious()
	benign := corpus.Benign()

	var tp, fn int
	var maliciousMisses []string
	for _, e := range malicious {
		r := promptfw.Analyze(e.Prompt)
		if r.RiskLevel == "high" || r.RiskLevel == "medium" {
			tp++
		} else {
			fn++
			maliciousMisses = append(maliciousMisses, e.Prompt)
		}
	}

	var fp, tn int
	var benignFalsePositives []string
	for _, e := range benign {
		r := promptfw.Analyze(e.Prompt)
		if r.RiskLevel == "high" || r.RiskLevel == "medium" {
			fp++
			benignFalsePositives = append(benignFalsePositives, e.Prompt)
		} else {
			tn++
		}
	}

	precision := 0.0
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	recall := 0.0
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	fmt.Printf("promptfw benchmark on bundled corpus\n")
	fmt.Printf("  malicious prompts: %d\n", len(malicious))
	fmt.Printf("  benign prompts:    %d\n", len(benign))
	fmt.Printf("  TP=%d  FP=%d  FN=%d  TN=%d\n", tp, fp, fn, tn)
	fmt.Printf("  precision: %.3f\n", precision)
	fmt.Printf("  recall:    %.3f\n", recall)
	fmt.Printf("  f1:        %.3f\n", f1)

	if len(maliciousMisses) > 0 {
		fmt.Printf("\nmissed malicious (%d):\n", len(maliciousMisses))
		for _, p := range maliciousMisses {
			fmt.Printf("  - %s\n", truncate(p, 100))
		}
	}
	if len(benignFalsePositives) > 0 {
		fmt.Printf("\nbenign false positives (%d):\n", len(benignFalsePositives))
		for _, p := range benignFalsePositives {
			fmt.Printf("  - %s\n", truncate(p, 100))
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func readPromptInput(fromFile string) (string, error) {
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return "", fmt.Errorf("read prompt file: %w", err)
		}
		return string(data), nil
	}

	// Read from stdin until EOF.
	info, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("stat stdin: %w", err)
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return "", fmt.Errorf("no prompt provided on stdin and no --file specified")
	}

	var b strings.Builder
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		b.WriteString(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("read stdin: %w", err)
		}
	}
	return b.String(), nil
}
