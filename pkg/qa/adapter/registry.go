package adapter

type Metadata struct {
	ID                   string   `json:"id"`
	Surfaces             []string `json:"surfaces"`
	RequiredBinaries     []string `json:"required_binaries"`
	DefaultLanes         []string `json:"default_lanes"`
	ArtifactCapabilities []string `json:"artifact_capabilities"`
	SetupGapReason       string   `json:"setup_gap_reason,omitempty"`
}

func Registry() []Metadata {
	return []Metadata{
		metadata("go-test", []string{"cli"}, []string{"go"}),
		metadata("node-script", []string{"package"}, []string{"node", "npm"}),
		metadata("vitest", []string{"frontend", "package"}, []string{"node", "npm"}),
		metadata("jest", []string{"frontend", "package"}, []string{"node", "npm"}),
		metadata("playwright", []string{"frontend"}, []string{"node", "npm"}),
		metadata("gui-explore", []string{"frontend", "desktop"}, []string{"node", "npm"}),
		metadata("pytest", []string{"cli"}, []string{"pytest"}),
		metadata("cargo-test", []string{"cli"}, []string{"cargo"}),
		metadata("auto-test-run", []string{"multi"}, []string{"auto"}),
		metadata("auto-verify", []string{"frontend"}, []string{"auto"}),
		metadata("canary-template", []string{"multi"}, nil),
		metadata("custom-command", []string{"custom"}, nil),
	}
}

func ByID(id string) (Metadata, bool) {
	for _, item := range Registry() {
		if item.ID == id {
			return item, true
		}
	}
	return Metadata{}, false
}

func metadata(id string, surfaces, binaries []string) Metadata {
	item := Metadata{
		ID:                   id,
		Surfaces:             surfaces,
		RequiredBinaries:     binaries,
		DefaultLanes:         []string{"fast"},
		ArtifactCapabilities: []string{"stdout", "stderr"},
	}
	if id == "gui-explore" {
		item.DefaultLanes = []string{"gui-explore"}
		item.ArtifactCapabilities = append(item.ArtifactCapabilities,
			"journey_graph",
			"aria_snapshot",
			"a11y_violations",
			"console_summary",
			"network_summary",
			"screenshot_quarantine_ref",
			"video_trace_ref",
			"dom_snapshot_digest",
		)
	}
	return item
}
