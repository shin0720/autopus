package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	qaevidence "github.com/insajin/autopus-adk/pkg/qa/evidence"
	"github.com/insajin/autopus-adk/pkg/qa/journey"
)

func applyGUIArtifactPublicationOracle(projectDir string, pack journey.Pack, result *commandResult) (IndexCheck, bool) {
	if pack.Adapter.ID != "gui-explore" {
		return IndexCheck{}, false
	}
	findings := unsafeGUIArtifactFindings(projectDir, pack)
	check := IndexCheck{
		ID:        guiArtifactPublicationCheckID,
		JourneyID: pack.ID,
		Adapter:   pack.Adapter.ID,
		Expected:  "declared_gui_artifacts=sanitized_relative_text; raw_network_headers_bodies=false",
		Actual:    "unsafe_findings=" + joinOrNone(findings),
	}
	if len(findings) == 0 {
		check.Status = "passed"
		return check, true
	}
	check.Status = "blocked"
	check.FailureSummary = "gui artifact publication boundary rejected unsafe declared artifact"
	result.Status = "blocked"
	result.FailureSummary = check.FailureSummary
	return check, true
}

func unsafeGUIArtifactFindings(projectDir string, pack journey.Pack) []string {
	findings := []string{}
	for _, artifact := range pack.Artifacts {
		kind := artifactKind(artifact)
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			findings = append(findings, "artifact_path_missing:"+kind)
			continue
		}
		if rawGUIArtifact(kind, path) {
			findings = append(findings, "raw_media_artifact:"+kind)
		}
		for _, finding := range qaevidence.FindUnsafeText(path, kind) {
			findings = append(findings, fmt.Sprintf("%s:%s", finding.Type, kind))
		}
		absPath := path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(projectDir, absPath)
		}
		body, err := os.ReadFile(absPath)
		if err != nil {
			findings = append(findings, "artifact_unreadable:"+kind)
			continue
		}
		for _, finding := range qaevidence.FindUnsafeText(string(body), kind) {
			findings = append(findings, fmt.Sprintf("%s:%s", finding.Type, kind))
		}
		findings = append(findings, rawNetworkArtifactFindings(kind, body)...)
	}
	return findings
}

func artifactKind(artifact journey.Artifact) string {
	kind := strings.TrimSpace(artifact.Kind)
	if kind == "" {
		return "artifact"
	}
	return kind
}

func rawGUIArtifact(kind, path string) bool {
	normalizedKind := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(kind))
	if strings.Contains(normalizedKind, "quarantine") ||
		strings.Contains(normalizedKind, "summary") ||
		strings.Contains(normalizedKind, "digest") ||
		strings.Contains(normalizedKind, "ref") {
		return false
	}
	switch normalizedKind {
	case "screenshot", "video", "trace", "har", "browser_trace", "playwright_trace", "network_har":
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".mp4", ".webm", ".mov", ".zip", ".trace", ".har":
		return true
	default:
		return false
	}
}

func rawNetworkArtifactFindings(kind string, body []byte) []string {
	if !strings.EqualFold(strings.TrimSpace(kind), "network_summary") {
		return nil
	}
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil
	}
	if hasRawNetworkPayload(doc) {
		return []string{"raw_network_payload:network_summary"}
	}
	return nil
}

func hasRawNetworkPayload(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
			if rawNetworkPayloadKey(normalized) && !emptyArtifactValue(child) {
				return true
			}
			if hasRawNetworkPayload(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if hasRawNetworkPayload(child) {
				return true
			}
		}
	}
	return false
}

func rawNetworkPayloadKey(key string) bool {
	return key == "body" || key == "headers" || strings.HasSuffix(key, "_body") || strings.HasSuffix(key, "_headers")
}

func emptyArtifactValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func blocksGUIArtifactPublication(checks []IndexCheck) bool {
	for _, check := range checks {
		if check.ID == guiArtifactPublicationCheckID && check.Status == "blocked" {
			return true
		}
	}
	return false
}
