package memindex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	qaevidence "github.com/insajin/autopus-adk/pkg/qa/evidence"
	qarun "github.com/insajin/autopus-adk/pkg/qa/run"
)

func scanQAMESH(projectDir string) ([]Record, []Skip, error) {
	root := filepath.Join(projectDir, ".autopus", "qa", "runs")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	records := make([]Record, 0)
	skips := make([]Skip, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "run-index.json" {
			return nil
		}
		next, nextSkips, err := qameshIndexRecords(projectDir, path)
		if err != nil {
			return err
		}
		records = append(records, next...)
		skips = append(skips, nextSkips...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return records, skips, nil
}

func qameshIndexRecords(projectDir, path string) ([]Record, []Skip, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var index qarun.Index
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, []Skip{{Path: slashRel(projectDir, path), Reason: "invalid_qamesh_run_index"}}, nil
	}
	// @AX:WARN [AUTO]: QAMESH run indexes enter memory only after the redaction status gate passes.
	// @AX:REASON: Memory recall output must not expose raw provider, browser, or artifact text.
	if index.RedactionStatus.Status != "" && index.RedactionStatus.Status != "passed" {
		return nil, []Skip{{Path: slashRel(projectDir, path), Reason: "unredacted_qamesh_run"}}, nil
	}
	records := make([]Record, 0)
	skips := make([]Skip, 0)
	record, ok, skip, err := qameshRunRecord(projectDir, path, index)
	if err != nil {
		return nil, nil, err
	}
	if ok {
		records = append(records, record)
		records = append(records, qameshRunDetailRecords(projectDir, path, index)...)
	} else {
		skips = append(skips, skip)
	}
	for _, manifestRef := range index.ManifestPaths {
		manifestPath, skipReason := resolveQAMESHRef(projectDir, filepath.Dir(path), manifestRef)
		if skipReason != "" {
			skips = append(skips, Skip{Path: manifestRef, Reason: skipReason})
			continue
		}
		record, ok, skip, err := qameshManifestRecord(projectDir, manifestPath)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			records = append(records, record)
			records = append(records, qameshManifestDetailRecords(projectDir, manifestPath, record.SourceHash, record.Timestamp, manifestRef)...)
		} else {
			skips = append(skips, skip)
		}
	}
	return records, skips, nil
}

func qameshRunRecord(projectDir, path string, index qarun.Index) (Record, bool, Skip, error) {
	ref := slashRel(projectDir, path)
	content := qameshRunText(index)
	if findings := qaevidence.FindUnsafeText(content, ref); len(findings) > 0 {
		return Record{}, false, Skip{Path: ref, Reason: "unsafe_source_text"}, nil
	}
	hash, err := hashFile(path)
	if err != nil {
		return Record{}, false, Skip{}, err
	}
	return Record{
		SourceType:      "qamesh_run",
		SourceRef:       ref,
		SourceHash:      hash,
		Title:           safeText("QAMESH run " + index.RunID),
		Summary:         safeText(content),
		Tags:            []string{"qamesh", index.Profile, index.Lane, index.Status},
		AcceptanceIDs:   qameshRunAcceptanceRefs(index),
		Severity:        qameshSeverity(index.Status),
		Timestamp:       index.EndedAt,
		RedactionStatus: Redacted,
		Content:         safeText(content),
	}, true, Skip{}, nil
}

func qameshManifestRecord(projectDir, path string) (Record, bool, Skip, error) {
	rel := slashRel(projectDir, path)
	manifest, err := qaevidence.LoadManifest(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, false, Skip{Path: rel, Reason: "missing_qamesh_manifest"}, nil
		}
		return Record{}, false, Skip{Path: rel, Reason: "invalid_qamesh_manifest"}, nil
	}
	// @AX:WARN [AUTO]: QAMESH manifests enter memory only after redaction validation passes.
	// @AX:REASON: Evidence manifests can reference captured artifacts; indexing unredacted manifests would leak sensitive test data.
	if manifest.RedactionStatus.Status != "passed" {
		return Record{}, false, Skip{Path: rel, Reason: "unredacted_qamesh_manifest"}, nil
	}
	content := qameshManifestText(manifest)
	if findings := qaevidence.FindUnsafeText(content, rel); len(findings) > 0 {
		return Record{}, false, Skip{Path: rel, Reason: "unsafe_source_text"}, nil
	}
	hash, err := hashFile(path)
	if err != nil {
		return Record{}, false, Skip{}, err
	}
	return Record{
		SourceType:      "qamesh_evidence",
		SourceRef:       rel,
		SourceHash:      hash,
		Title:           safeText(fmt.Sprintf("QAMESH evidence %s", manifest.QAResultID)),
		Summary:         safeText(content),
		Tags:            []string{"qamesh", manifest.Surface, manifest.Lane, manifest.Status},
		SpecID:          manifest.SourceRefs.SourceSpec,
		AcceptanceIDs:   manifest.SourceRefs.AcceptanceRefs,
		FileRefs:        manifest.SourceRefs.OwnedPaths,
		Severity:        qameshSeverity(manifest.Status),
		Timestamp:       manifest.EndedAt,
		RedactionStatus: Redacted,
		Content:         safeText(content),
	}, true, Skip{}, nil
}

func qameshRunText(index qarun.Index) string {
	parts := []string{
		"status " + index.Status,
		"profile " + index.Profile,
		"lane " + index.Lane,
	}
	for _, check := range index.Checks {
		if check.Status == "failed" || check.Status == "blocked" {
			parts = append(parts, fmt.Sprintf("failed check %s %s %s %s", check.ID, check.Expected, check.Actual, check.FailureSummary))
		}
	}
	for _, gap := range index.SetupGaps {
		parts = append(parts, fmt.Sprintf("setup gap %s %s %s", gap.Adapter, gap.JourneyID, gap.Reason))
	}
	if len(index.FeedbackBundlePaths) > 0 {
		parts = append(parts, "repair prompt refs "+strings.Join(index.FeedbackBundlePaths, " "))
	}
	return strings.Join(parts, "\n")
}

func qameshManifestText(manifest qaevidence.Manifest) string {
	parts := []string{
		"status " + manifest.Status,
		"surface " + manifest.Surface,
		"lane " + manifest.Lane,
		"scenario " + manifest.ScenarioRef,
		"repair prompt ref " + manifest.RepairPromptRef,
		"acceptance refs " + strings.Join(manifest.SourceRefs.AcceptanceRefs, " "),
	}
	for _, check := range manifest.OracleResults.Checks {
		if check.Status == "failed" || check.Status == "blocked" {
			parts = append(parts, fmt.Sprintf("failed check %s %s %s %s", check.ID, check.Expected, check.Actual, check.FailureSummary))
		}
	}
	return strings.Join(parts, "\n")
}

func qameshRunAcceptanceRefs(index qarun.Index) []string {
	refs := make([]string, 0)
	for _, result := range index.AdapterResults {
		if result.FailureSummary != "" {
			refs = append(refs, acceptanceIDs(result.FailureSummary)...)
		}
	}
	sort.Strings(refs)
	return uniqueStrings(refs)
}

func qameshSeverity(status string) string {
	switch status {
	case "failed":
		return "high"
	case "blocked":
		return "medium"
	default:
		return "low"
	}
}

func resolveQAMESHRef(projectDir, baseDir, ref string) (string, string) {
	if filepath.IsAbs(ref) {
		return validateQAMESHPath(projectDir, ref)
	}
	if strings.HasPrefix(filepath.ToSlash(ref), ".autopus/") {
		return validateQAMESHPath(projectDir, filepath.Join(projectDir, filepath.FromSlash(ref)))
	}
	return validateQAMESHPath(projectDir, filepath.Join(baseDir, ref))
}

func validateQAMESHPath(projectDir, path string) (string, string) {
	root := filepath.Join(projectDir, ".autopus", "qa", "runs")
	ok, err := pathWithinForCreate(root, path)
	if err != nil || !ok {
		return "", "outside_configured_roots"
	}
	return filepath.Clean(path), ""
}
