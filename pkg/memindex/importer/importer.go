package importer

import "github.com/insajin/autopus-adk/pkg/memindex"

type Record = memindex.Record
type Skip = memindex.Skip

func Scan(projectDir string) ([]Record, []Skip, error) {
	return memindex.Scan(projectDir)
}
