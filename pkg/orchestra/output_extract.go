package orchestra

import (
	"encoding/json"
	"strings"
)

// ExtractClaudeJSONOutput extracts displayable text from a Claude subprocess
// response. When the response is a JSON object with a "content", "text", or
// "result" field, the string value of that field is returned. Otherwise the
// raw output is returned after stripping leading/trailing whitespace.
func ExtractClaudeJSONOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "{") {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &m); err == nil {
			for _, key := range []string{"content", "text", "result"} {
				if raw, ok := m[key]; ok {
					var s string
					if err := json.Unmarshal(raw, &s); err == nil {
						return s
					}
				}
			}
		}
	}
	return trimmed
}
