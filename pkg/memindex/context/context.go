package context

import "github.com/insajin/autopus-adk/pkg/memindex"

type Options = memindex.ContextOptions
type Result = memindex.ContextResult

func Render(opts Options) (Result, error) {
	return memindex.Context(opts)
}
