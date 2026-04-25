package opencode

import (
	"fmt"
	"path/filepath"

	contentfs "github.com/shin0720/auto-adk/content"
	"github.com/shin0720/auto-adk/pkg/adapter"
	pkgcontent "github.com/shin0720/auto-adk/pkg/content"
)

func (a *Adapter) prepareAgentMappings() ([]adapter.FileMapping, error) {
	sources, err := pkgcontent.LoadAgentSourcesFromFS(contentfs.FS, "agents")
	if err != nil {
		return nil, fmt.Errorf("agent source 로드 실패: %w", err)
	}
	files := make([]adapter.FileMapping, 0, len(sources))
	for _, src := range sources {
		content := pkgcontent.TransformAgentForOpenCode(src)
		files = append(files, adapter.FileMapping{
			TargetPath:      filepath.Join(".opencode", "agents", src.Meta.Name+".md"),
			OverwritePolicy: adapter.OverwriteAlways,
			Checksum:        adapter.Checksum(content),
			Content:         []byte(content),
		})
	}
	return files, nil
}
