package memindex

import (
	"fmt"

	qaevidence "github.com/insajin/autopus-adk/pkg/qa/evidence"
	qarun "github.com/insajin/autopus-adk/pkg/qa/run"
)

func qameshRunDetailRecords(projectDir, path string, index qarun.Index) []Record {
	rel := slashRel(projectDir, path)
	hash, err := hashFile(path)
	if err != nil {
		return nil
	}
	records := make([]Record, 0)
	for _, check := range index.Checks {
		if check.Status != "failed" && check.Status != "blocked" {
			continue
		}
		content := safeText(fmt.Sprintf("deterministic QA failure failed check %s %s %s %s %s", check.ID, check.JourneyID, check.Expected, check.Actual, check.FailureSummary))
		records = append(records, Record{
			SourceType:      "qamesh_failed_check",
			SourceRef:       rel + "#check:" + check.ID,
			SourceHash:      hash,
			Title:           safeText(check.ID),
			Summary:         content,
			Tags:            []string{"qamesh", "failed_check", check.Adapter, check.JourneyID, check.Status},
			Severity:        qameshSeverity(index.Status),
			Timestamp:       index.EndedAt,
			RedactionStatus: Redacted,
			Content:         content,
			SourceMetadata: map[string]any{
				"check_id":        check.ID,
				"journey_id":      check.JourneyID,
				"adapter":         check.Adapter,
				"status":          check.Status,
				"failure_summary": check.FailureSummary,
			},
		})
	}
	records = append(records, qameshSetupGapRecords(rel, hash, index)...)
	records = append(records, qameshRunRepairRecords(rel, hash, index)...)
	return records
}

func qameshSetupGapRecords(rel, hash string, index qarun.Index) []Record {
	records := make([]Record, 0)
	for _, gap := range index.SetupGaps {
		content := safeText(fmt.Sprintf("deterministic QA failure setup gap %s %s %s", gap.Adapter, gap.JourneyID, gap.Reason))
		records = append(records, Record{
			SourceType:      "qamesh_setup_gap",
			SourceRef:       rel + "#setup-gap:" + gap.Adapter + ":" + gap.JourneyID,
			SourceHash:      hash,
			Title:           safeText("QAMESH setup gap " + gap.Adapter),
			Summary:         content,
			Tags:            []string{"qamesh", "setup_gap", gap.Adapter, gap.JourneyID},
			Severity:        "medium",
			Timestamp:       index.EndedAt,
			RedactionStatus: Redacted,
			Content:         content,
			SourceMetadata: map[string]any{
				"adapter":    gap.Adapter,
				"journey_id": gap.JourneyID,
				"reason":     gap.Reason,
			},
		})
	}
	return records
}

func qameshRunRepairRecords(rel, hash string, index qarun.Index) []Record {
	records := make([]Record, 0)
	for _, ref := range index.FeedbackBundlePaths {
		content := safeText("deterministic QA failure repair prompt ref " + ref + "/repair-prompt.md")
		records = append(records, Record{
			SourceType:      "qamesh_repair_prompt",
			SourceRef:       rel + "#repair:" + ref,
			SourceHash:      hash,
			Title:           safeText("QAMESH repair prompt " + ref),
			Summary:         content,
			Tags:            []string{"qamesh", "repair_prompt"},
			Severity:        qameshSeverity(index.Status),
			Timestamp:       index.EndedAt,
			RedactionStatus: Redacted,
			Content:         content,
			SourceMetadata:  map[string]any{"repair_prompt_ref": ref},
		})
	}
	return records
}

func qameshManifestDetailRecords(projectDir, path, hash, timestamp, manifestRef string) []Record {
	manifest, err := qaevidence.LoadManifest(path)
	if err != nil || manifest.RedactionStatus.Status != "passed" {
		return nil
	}
	rel := slashRel(projectDir, path)
	if manifest.RepairPromptRef == "" {
		return nil
	}
	content := safeText("deterministic QA failure repair prompt ref " + manifest.ScenarioRef + " " + manifest.RepairPromptRef)
	return []Record{{
		SourceType:      "qamesh_repair_prompt",
		SourceRef:       rel + "#repair:" + manifest.RepairPromptRef,
		SourceHash:      hash,
		Title:           safeText("QAMESH repair prompt " + manifest.QAResultID),
		Summary:         content,
		Tags:            []string{"qamesh", "repair_prompt", manifest.Surface, manifest.Lane},
		SpecID:          manifest.SourceRefs.SourceSpec,
		AcceptanceIDs:   manifest.SourceRefs.AcceptanceRefs,
		FileRefs:        manifest.SourceRefs.OwnedPaths,
		Severity:        qameshSeverity(manifest.Status),
		Timestamp:       timestamp,
		RedactionStatus: Redacted,
		Content:         content,
		SourceMetadata: map[string]any{
			"qa_result_id":      manifest.QAResultID,
			"manifest_ref":      manifestRef,
			"repair_prompt_ref": manifest.RepairPromptRef,
		},
	}}
}
