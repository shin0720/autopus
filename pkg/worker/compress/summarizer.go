package compress

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/insajin/autopus-adk/pkg/worker/security"
)

// @AX:NOTE: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: hardcoded markdown header patterns define the structured compaction schema
// Section headers recognized in phase output.
var sectionPatterns = map[string]*regexp.Regexp{
	"goal":        regexp.MustCompile(`(?i)^#{1,6}\s*(goal|objective|task)\s*$`),
	"constraints": regexp.MustCompile(`(?i)^#{1,6}\s*(constraints?|guardrails?|requirements?)\s*$`),
	"progress":    regexp.MustCompile(`(?i)^#{1,6}\s*(progress|actions?|done|completed|changes?)\s*$`),
	"decision":    regexp.MustCompile(`(?i)^#{1,6}\s*(decisions?|decision|design|architecture)\s*$`),
	"files":       regexp.MustCompile(`(?i)^#{1,6}\s*((relevant\s+)?files?(\s*(modified|changed))?|modified\s*files?)\s*$`),
	"next":        regexp.MustCompile(`(?i)^#{1,6}\s*(next\s*steps?|todo|remaining)\s*$`),
	"critical":    regexp.MustCompile(`(?i)^#{1,6}\s*(critical\s+context|warnings?|risks?|blockers?|cautions?)\s*$`),
}

// filePattern matches file paths in text (e.g., "pkg/worker/compress/budget.go").
var filePattern = regexp.MustCompile(`(?:^|\s)([\w./\-]+\.\w{1,5})`)

var (
	previousSummaryIDPattern = regexp.MustCompile(`(?m)^summary_id:\s*([^\s]+)\s*$`)
	// @AX:NOTE: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: redaction regexes define what derived summaries may carry into indexes and compaction events
	secretPattern     = regexp.MustCompile(`(?i)\bsk-[a-z0-9][a-z0-9_-]{8,}\b`)
	credentialPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|password)\s*[:=]\s*["']?[^\s"',}]{8,}`)
	localPathPattern  = regexp.MustCompile(`(?i)([A-Z]:\\[^\s"'<>)]+|/(Users|home|private/var/folders|var/folders|tmp)/[^\s"'<>)]+)`)
)

// @AX:ANCHOR: [AUTO] @AX:SPEC: SPEC-CONTEXT-COMPRESS-001: structured compaction schema boundary
// @AX:REASON: Compressor, pipeline, orchestra, and acceptance tests depend on the metadata fields and seven summary sections staying compatible.
// Summarize generates a structured summary from phase output text.
// The summary follows the structured compaction schema plus a compatibility
// Files Modified section for existing callers.
func Summarize(phaseName, text string, maxTokens int) string {
	redactedText, redactionReasons := redactUnsafeContext(text)
	summaryInput, omittedToolBodies := omitToolPayloadBodies(redactedText)
	if omittedToolBodies {
		redactionReasons = appendUnique(redactionReasons, "provider_payload")
	}
	sections := extractSections(summaryInput)
	files := extractFiles(summaryInput)
	previousSummaryID := extractPreviousSummaryID(summaryInput)

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Phase Summary: %s\n\n", phaseName)
	writeMetadata(&sb, phaseName, summaryInput, previousSummaryID, redactionReasons)

	writeSection(&sb, "Goal", sections["goal"], "Phase output analysis")
	writeSection(&sb, "Constraints", sections["constraints"], "No explicit constraints recorded")
	writeSection(&sb, "Progress", sections["progress"], "Completed phase tasks")
	writeSection(&sb, "Decisions", sections["decision"], "No explicit decisions recorded")
	writeFileSection(&sb, "Relevant Files", files, sections["files"], "No relevant file paths detected")
	writeSection(&sb, "Next Steps", sections["next"], "Continue to next phase")
	writeSection(&sb, "Critical Context", sections["critical"], "No critical context recorded")
	writeFileSection(&sb, "Files Modified", files, sections["files"], "No file paths detected")

	result := sb.String()
	if shouldFailClosedForBudget(result, sections, maxTokens) {
		return budgetBlockerSummary(phaseName, summaryInput, sections, files, previousSummaryID, redactionReasons, maxTokens)
	}
	return truncateToTokens(result, maxTokens)
}

// extractSections parses text looking for known markdown section headers
// and captures content under each.
func extractSections(text string) map[string]string {
	lines := strings.Split(text, "\n")
	sections := make(map[string]string)
	currentSection := ""
	var content []string

	for _, line := range lines {
		matched := false
		for name, pat := range sectionPatterns {
			if pat.MatchString(strings.TrimSpace(line)) {
				flushSection(sections, currentSection, content)
				currentSection = name
				content = nil
				matched = true
				break
			}
		}
		if !matched && currentSection != "" {
			content = append(content, line)
		}
	}

	flushSection(sections, currentSection, content)
	return sections
}

func flushSection(sections map[string]string, name string, content []string) {
	if name == "" || len(content) == 0 {
		return
	}
	value := strings.TrimSpace(strings.Join(content, "\n"))
	if value == "" {
		return
	}
	if existing := strings.TrimSpace(sections[name]); existing != "" {
		sections[name] = existing + "\n\n" + value
		return
	}
	sections[name] = value
}

// extractFiles finds file paths mentioned in the text.
func extractFiles(text string) []string {
	matches := filePattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var files []string
	for _, m := range matches {
		path := m[1]
		if !strings.Contains(path, "/") {
			continue // skip bare filenames without directory
		}
		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files
}

func writeSection(sb *strings.Builder, title, content, fallback string) {
	fmt.Fprintf(sb, "### %s\n", title)
	if content != "" {
		fmt.Fprintf(sb, "%s\n\n", content)
	} else {
		fmt.Fprintf(sb, "%s\n\n", fallback)
	}
}

func writeFileSection(sb *strings.Builder, title string, files []string, content, fallback string) {
	fmt.Fprintf(sb, "### %s\n", title)
	if len(files) == 0 {
		if strings.TrimSpace(content) != "" {
			fmt.Fprintf(sb, "%s\n\n", strings.TrimSpace(content))
			return
		}
		fmt.Fprintf(sb, "%s\n\n", fallback)
		return
	}
	for _, f := range files {
		fmt.Fprintf(sb, "- %s\n", f)
	}
	sb.WriteString("\n")
}

func writeMetadata(sb *strings.Builder, phaseName, text, previousSummaryID string, redactionReasons []string) {
	fmt.Fprintf(sb, "summary_id: %s\n", summaryID(phaseName, text))
	if previousSummaryID != "" {
		fmt.Fprintf(sb, "previous_summary_id: %s\n", previousSummaryID)
	}
	for _, reason := range redactionReasons {
		fmt.Fprintf(sb, "redaction_reason=%s\n", reason)
	}
	if strings.Contains(text, "index_export=true") && len(redactionReasons) > 0 {
		sb.WriteString("index_eligibility=redacted_derived_context\n")
	}
	sb.WriteString("\n")
}

func summaryID(phaseName, text string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(phaseName))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(text))
	return fmt.Sprintf("summary-%08x", h.Sum32())
}

func extractPreviousSummaryID(text string) string {
	match := previousSummaryIDPattern.FindStringSubmatch(text)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func redactUnsafeContext(text string) (string, []string) {
	var reasons []string
	scanner := security.NewSecretScanner()
	if scanner.ContainsSecret(text) {
		reasons = appendUnique(reasons, "secret")
		text = strings.ReplaceAll(scanner.Scan(text), "***REDACTED***", "[redacted:secret]")
	}
	if secretPattern.MatchString(text) {
		reasons = appendUnique(reasons, "secret")
		text = secretPattern.ReplaceAllString(text, "[redacted:secret]")
	}
	if credentialPattern.MatchString(text) {
		reasons = appendUnique(reasons, "secret")
		text = credentialPattern.ReplaceAllString(text, "[redacted:secret]")
	}
	if localPathPattern.MatchString(text) {
		reasons = appendUnique(reasons, "local_path")
		text = localPathPattern.ReplaceAllString(text, "[redacted:local_path]")
	}
	return text, reasons
}

func shouldFailClosedForBudget(result string, sections map[string]string, maxTokens int) bool {
	if maxTokens <= 0 || EstimateTokens(result) <= maxTokens {
		return false
	}
	return strings.TrimSpace(sections["constraints"]) != "" ||
		strings.TrimSpace(sections["decision"]) != "" ||
		strings.TrimSpace(sections["critical"]) != ""
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func budgetBlockerSummary(phaseName, text string, sections map[string]string, files []string, previousSummaryID string, redactionReasons []string, maxTokens int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## Phase Summary: %s\n\n", phaseName)
	writeMetadata(&sb, phaseName, text, previousSummaryID, redactionReasons)
	fmt.Fprintf(&sb, "blocker: context-budget\n")
	fmt.Fprintf(&sb, "context-budget: max_tokens=%d required_schema_sections=Goal,Constraints,Progress,Decisions,Relevant Files,Next Steps,Critical Context\n\n", maxTokens)
	writeSection(&sb, "Goal", sections["goal"], "Phase output analysis")
	writeSection(&sb, "Constraints", sections["constraints"], "No explicit constraints recorded")
	writeSection(&sb, "Progress", sections["progress"], "Completed phase tasks")
	writeSection(&sb, "Decisions", sections["decision"], "No explicit decisions recorded")
	writeFileSection(&sb, "Relevant Files", files, sections["files"], "No relevant file paths detected")
	writeSection(&sb, "Next Steps", sections["next"], "Continue to next phase")
	writeSection(&sb, "Critical Context", sections["critical"], "No critical context recorded")
	return sb.String()
}

// truncateToTokens truncates text to fit within token budget.
func truncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 || EstimateTokens(text) <= maxTokens {
		return text
	}
	maxChars := maxTokens * 4
	if maxChars >= len(text) {
		return text
	}
	return text[:maxChars] + "\n\n[...truncated to fit summary budget]\n"
}
