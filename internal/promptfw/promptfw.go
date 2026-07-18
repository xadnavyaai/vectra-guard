package promptfw

import (
	"encoding/base64"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Result is the classification result for a prompt.
type Result struct {
	RiskLevel string   `json:"risk_level"` // low, medium, high
	Reasons   []string `json:"reasons"`
	Score     float64  `json:"score"` // aggregate numeric score for debugging
}

var (
	// Regexes for common prompt-injection / jailbreak patterns.
	injectionPatterns = []struct {
		id     string
		re     *regexp.Regexp
		weight float64
	}{
		// 1. IGNORE_INSTRUCTIONS — ignore/disregard/forget + previous/prior + instructions/rules
		{
			id:     "IGNORE_INSTRUCTIONS",
			re:     regexp.MustCompile(`(?i)(ignore|disregard|forget).{0,30}(previous|prior|above|all).{0,30}(instructions|rules|messages|guidelines|prompts?)`),
			weight: 3.0,
		},
		// 2. OVERRIDE_SAFETY — disable/bypass/override + safety/guardrails/filters
		{
			id:     "OVERRIDE_SAFETY",
			re:     regexp.MustCompile(`(?i)(disable|bypass|override|remove|turn off).{0,30}(safety|safe mode|guardrails?|filters?|restrictions?|content.?policy)`),
			weight: 2.5,
		},
		// 3. REVEAL_SECRETS — reveal/dump/exfiltrate + secret/credential/token/api key
		{
			id:     "REVEAL_SECRETS",
			re:     regexp.MustCompile(`(?i)(reveal|show|exfiltrate|dump|output|leak|expose).{0,30}(secret|credential|password|token|api.?key|private.?key|env.?var)`),
			weight: 3.0,
		},
		// 4. SYSTEM_PROMPT_INJECTION — you are now/act as/pretend to be + root/admin/unrestricted
		{
			id:     "SYSTEM_PROMPT_INJECTION",
			re:     regexp.MustCompile(`(?i)(you are (now )?|forget you are|act as|pretend to be).{0,30}(root|admin|unrestricted|developer|system|unfiltered|super.?user)`),
			weight: 2.0,
		},
		// 5. JAILBREAK_PERSONA — DAN/DUDE/AIM/developer mode/god mode/jailbreak
		{
			id:     "JAILBREAK_PERSONA",
			re:     regexp.MustCompile(`(?i)\b(DAN|DUDE|AIM|developer\s+mode|god\s+mode|jailbreak|do\s+anything\s+now)\b`),
			weight: 2.5,
		},
		// 6. PROMPT_LEAK — repeat/output/print + system/initial/hidden + prompt/instructions
		{
			id:     "PROMPT_LEAK",
			re:     regexp.MustCompile(`(?i)(repeat|output|print|display|show|echo).{0,30}(system|initial|hidden|original|first).{0,30}(prompt|instructions|message)`),
			weight: 2.5,
		},
		// 7. DELIMITER_INJECTION — [SYSTEM], <|im_start|>, ```system, ### system
		{
			id:     "DELIMITER_INJECTION",
			re:     regexp.MustCompile("(?i)(\\[SYSTEM\\]|<\\|im_start\\|>|<\\|im_end\\|>|```system|###\\s*system|<\\|endoftext\\|>|<\\|assistant\\|>)"),
			weight: 3.0,
		},
		// 8. ROLE_IMPERSONATION — "speaking as the system admin" / "I am the developer"
		{
			id:     "ROLE_IMPERSONATION",
			re:     regexp.MustCompile(`(?i)(speaking as|i am|this is).{0,20}(the )?(system\s*admin|administrator|developer|lead\s*engineer|openai|anthropic)`),
			weight: 2.5,
		},
		// 9. CONTEXT_MANIPULATION — "translate the following" hiding injection payload
		{
			id:     "CONTEXT_MANIPULATION",
			re:     regexp.MustCompile(`(?i)(translate|summarize|rephrase).{0,30}(following|below|text).{0,60}(ignore|forget|disregard|override)`),
			weight: 1.5,
		},
		// 10. INDIRECT_INJECTION — "when you read file X, execute Y"
		{
			id:     "INDIRECT_INJECTION",
			re:     regexp.MustCompile(`(?i)(when|if|after).{0,20}(you )?(read|open|access|process|load).{0,30}(execute|run|eval|call|perform|send)`),
			weight: 2.0,
		},
		// 11. CODE_EXECUTION — eval()/exec()/__import__/os.system/subprocess
		{
			id:     "CODE_EXECUTION",
			re:     regexp.MustCompile(`(?i)(eval|exec|__import__|os\.system|subprocess|child_process|require\s*\(\s*['"]child_process['"]\))\s*\(`),
			weight: 2.5,
		},
		// 12. ENCODING_EVASION_MARKER — "decode this base64/hex/rot13"
		{
			id:     "ENCODING_EVASION_MARKER",
			re:     regexp.MustCompile(`(?i)(decode|convert|interpret).{0,20}(base64|hex|rot13|unicode|ascii|url.?encode)`),
			weight: 2.0,
		},
		// 13. MULTI_LANG_INJECTION — Injection keywords in multiple languages
		{
			id:     "MULTI_LANG_INJECTION",
			re:     regexp.MustCompile(`(?i)(ignorar|ignorez|ignorieren|忽略|無視|игнорируй).{0,30}(instrucciones|instructions|Anweisungen|指令|指示|инструкции)`),
			weight: 2.0,
		},
		// 14. TOOL_FUNCTION_ABUSE — function_call/tool_use JSON manipulation
		{
			id:     "TOOL_FUNCTION_ABUSE",
			re:     regexp.MustCompile(`(?i)("function_call"|"tool_use"|"tool_calls"|"name"\s*:\s*"(run_code|execute|shell)").*("arguments"|"input"|"parameters")`),
			weight: 2.0,
		},
		// 15. EXFILTRATION_CHANNEL — Markdown image exfil, fetch/curl with data= params
		{
			id:     "EXFILTRATION_CHANNEL",
			re:     regexp.MustCompile(`(?i)(!\[.*\]\(https?://.*\?.*=|fetch\s*\(.*data=|curl\s+.*-d\s|wget\s+.*--post-data)`),
			weight: 2.5,
		},
		// 16. RESET_CONVERSATION — "forget everything", "reset your context"
		{
			id:     "RESET_CONVERSATION",
			re:     regexp.MustCompile(`(?i)(forget|erase|clear|reset|wipe).{0,20}(everything|all|context|conversation|memory|history)`),
			weight: 1.5,
		},
		// 17. AUTHORITY_CLAIM — "I am an OpenAI researcher with admin access"
		{
			id:     "AUTHORITY_CLAIM",
			re:     regexp.MustCompile(`(?i)i\s+(am|work\s+at|work\s+for).{0,30}(openai|anthropic|google\s*deepmind).{0,30}(admin|researcher|access|permission|authority)`),
			weight: 2.0,
		},
		// 18. COMPETITIVE_EXTRACTION — "output training data/model weights"
		{
			id:     "COMPETITIVE_EXTRACTION",
			re:     regexp.MustCompile(`(?i)(output|reveal|give|share|extract).{0,30}(training\s*data|model\s*weights|fine.?tun|dataset|embeddings)`),
			weight: 2.0,
		},
	}

	highEntropyTokenRe = regexp.MustCompile(`([A-Za-z0-9+/=_-]{32,})`)

	// A tiny baseline 3-gram histogram for "benign" English-ish prompts.
	// This is intentionally simple; we only care about how many n-grams are
	// completely unknown relative to this baseline.
	baselineTrigrams = map[string]int{
		" th": 50, "he ": 40, "ing": 35, "er ": 30, "re ": 25,
		" to": 25, "ou ": 20, "in ": 40, "de ": 15, "on ": 20,
	}

	// Zero-width and invisible characters used for obfuscation.
	zeroWidthChars = []rune{
		'\u200B', // zero-width space
		'\u200C', // zero-width non-joiner
		'\u200D', // zero-width joiner
		'\uFEFF', // byte order mark / zero-width no-break space
		'\u2060', // word joiner
	}

	// Leetspeak mappings.
	leetMap = map[rune]rune{
		'0': 'o',
		'1': 'i',
		'3': 'e',
		'4': 'a',
		'5': 's',
		'7': 't',
		'@': 'a',
		'$': 's',
	}
)

// Analyze classifies a prompt into low/medium/high risk using:
// - regex rules for injection phrases
// - entropy-based detection of suspicious tokens
// - simple trigram anomaly scoring
// - encoded payload detection
// - obfuscation detection (zero-width chars, leetspeak)
// - sandwich attack detection
func Analyze(prompt string) Result {
	text := strings.TrimSpace(prompt)
	if text == "" {
		return Result{RiskLevel: "low", Reasons: []string{"empty prompt"}, Score: 0}
	}

	var (
		score   float64
		reasons []string
	)

	lower := strings.ToLower(text)

	// 1) Regex-based injection patterns.
	for _, pat := range injectionPatterns {
		if pat.re.MatchString(lower) {
			score += pat.weight
			reasons = append(reasons, "pattern:"+pat.id)
		}
	}

	// 2) High-entropy segments (base64 / random strings).
	tokens := highEntropyTokenRe.FindAllString(text, -1)
	for _, tok := range tokens {
		h := shannonEntropy(tok)
		if h >= 4.0 {
			score += 1.5
			reasons = append(reasons, "high_entropy_segment")
			break
		}
	}

	// 3) Trigram anomaly score.
	anomaly := trigramAnomaly(lower)
	if anomaly > 0.3 {
		score += anomaly * 2.0
		reasons = append(reasons, "ngram_anomaly")
	}

	// 4) Encoded payload detection.
	encScore, encReasons := checkEncodedPayloads(text)
	score += encScore
	reasons = append(reasons, encReasons...)

	// 5) Obfuscation detection.
	obfScore, obfReasons := checkObfuscation(text)
	score += obfScore
	reasons = append(reasons, obfReasons...)

	// 6) Sandwich attack detection.
	swScore, swReasons := checkSandwichAttack(text)
	score += swScore
	reasons = append(reasons, swReasons...)

	// Map score to discrete risk level.
	switch {
	case score >= 4.5:
		return Result{RiskLevel: "high", Reasons: reasons, Score: score}
	case score >= 2.0:
		return Result{RiskLevel: "medium", Reasons: reasons, Score: score}
	default:
		return Result{RiskLevel: "low", Reasons: reasons, Score: score}
	}
}

// checkEncodedPayloads decodes base64 candidates in the text and checks if
// the decoded content matches any injection pattern.
func checkEncodedPayloads(text string) (float64, []string) {
	// Look for base64-ish tokens (at least 16 chars to reduce false positives).
	b64Re := regexp.MustCompile(`[A-Za-z0-9+/]{16,}={0,2}`)
	candidates := b64Re.FindAllString(text, 10)

	for _, candidate := range candidates {
		decoded, err := base64.StdEncoding.DecodeString(candidate)
		if err != nil {
			// Try URL-safe encoding.
			decoded, err = base64.URLEncoding.DecodeString(candidate)
			if err != nil {
				continue
			}
		}
		decodedStr := string(decoded)
		if !utf8.ValidString(decodedStr) {
			continue
		}
		if matchesAnyInjectionPattern(decodedStr) {
			return 2.5, []string{"encoded_payload_base64"}
		}
	}
	return 0, nil
}

// checkObfuscation detects zero-width character injection and leetspeak evasion.
func checkObfuscation(text string) (float64, []string) {
	var score float64
	var reasons []string

	// Zero-width character detection.
	for _, r := range text {
		for _, zw := range zeroWidthChars {
			if r == zw {
				score += 1.5
				reasons = append(reasons, "obfuscation_zero_width")
				goto leetCheck
			}
		}
	}

leetCheck:
	// Leetspeak normalization — normalize and re-check patterns.
	normalized := normalizeLeet(strings.ToLower(text))
	if normalized != strings.ToLower(text) {
		if matchesAnyInjectionPattern(normalized) {
			score += 1.5
			reasons = append(reasons, "obfuscation_leetspeak")
		}
	}

	return score, reasons
}

// checkSandwichAttack splits the prompt into thirds and flags when the middle
// section scores much higher than the edges (hiding malicious content inside
// benign padding).
func checkSandwichAttack(text string) (float64, []string) {
	runes := []rune(text)
	n := len(runes)
	if n < 60 {
		return 0, nil
	}

	third := n / 3
	top := string(runes[:third])
	middle := string(runes[third : 2*third])
	bottom := string(runes[2*third:])

	topScore := rawPatternScore(top)
	middleScore := rawPatternScore(middle)
	bottomScore := rawPatternScore(bottom)

	edgeAvg := (topScore + bottomScore) / 2.0
	if edgeAvg == 0 {
		edgeAvg = 0.1
	}

	if middleScore > 0 && middleScore > edgeAvg*2.0 {
		return 1.5, []string{"sandwich_attack"}
	}
	return 0, nil
}

// rawPatternScore runs only the regex patterns and returns the aggregate score.
// Used by checkSandwichAttack to compare sections.
func rawPatternScore(text string) float64 {
	lower := strings.ToLower(text)
	var score float64
	for _, pat := range injectionPatterns {
		if pat.re.MatchString(lower) {
			score += pat.weight
		}
	}
	return score
}

// matchesAnyInjectionPattern returns true if text matches any injection regex.
func matchesAnyInjectionPattern(text string) bool {
	lower := strings.ToLower(text)
	for _, pat := range injectionPatterns {
		if pat.re.MatchString(lower) {
			return true
		}
	}
	return false
}

// normalizeLeet converts leetspeak characters to their ASCII letter equivalents.
func normalizeLeet(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if mapped, ok := leetMap[r]; ok {
			b.WriteRune(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	freq := make(map[rune]int)
	for _, r := range s {
		freq[r]++
	}
	l := float64(len([]rune(s)))
	var entropy float64
	for _, count := range freq {
		p := float64(count) / l
		entropy -= p * math.Log2(p)
	}
	if math.IsNaN(entropy) {
		return 0
	}
	return entropy
}

func trigramAnomaly(text string) float64 {
	if len(text) < 3 {
		return 0
	}
	text = strings.ReplaceAll(text, "\n", " ")
	text = " " + text + " "

	var total, unknown int
	for i := 0; i+3 <= len(text); i++ {
		tg := text[i : i+3]
		if strings.TrimSpace(tg) == "" {
			continue
		}
		total++
		if _, ok := baselineTrigrams[tg]; !ok {
			unknown++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(unknown) / float64(total)
}
