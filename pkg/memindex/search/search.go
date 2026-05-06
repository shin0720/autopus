package search

import "github.com/insajin/autopus-adk/pkg/memindex"

type Options = memindex.SearchOptions
type Response = memindex.SearchResponse
type Result = memindex.SearchResult

func Search(opts Options) (Response, error) {
	return memindex.Search(opts)
}
