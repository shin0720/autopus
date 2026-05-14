package run

import (
	"strings"

	qacompile "github.com/insajin/autopus-adk/pkg/qa/compile"
	"github.com/insajin/autopus-adk/pkg/qa/journey"
)

func artifactPreviewsForPack(pack journey.Pack) []ArtifactPreview {
	out := make([]ArtifactPreview, 0, len(pack.Artifacts))
	for _, artifact := range pack.Artifacts {
		kind := artifact.Kind
		if kind == "" {
			kind = "artifact"
		}
		publishable := !strings.Contains(strings.ToLower(kind), "quarantine")
		redaction := "text_redacted_and_scanned"
		if !publishable {
			redaction = "local_only_quarantine_ref"
		}
		out = append(out, ArtifactPreview{
			JourneyID:   pack.ID,
			Adapter:     pack.Adapter.ID,
			Kind:        kind,
			Path:        artifact.Path,
			Publishable: publishable,
			Redaction:   redaction,
		})
	}
	return out
}

func artifactPreviewsForCandidate(candidate qacompile.Candidate) []ArtifactPreview {
	out := make([]ArtifactPreview, 0, len(candidate.Artifacts))
	for _, path := range candidate.Artifacts {
		out = append(out, ArtifactPreview{
			JourneyID:   candidate.JourneyID,
			Adapter:     candidate.Adapter,
			Kind:        "artifact",
			Path:        path,
			Publishable: true,
			Redaction:   "text_redacted_and_scanned",
		})
	}
	return out
}
