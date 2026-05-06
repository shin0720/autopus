package memindex

import "regexp"

func regexpAcceptanceID() *regexp.Regexp {
	return regexp.MustCompile(`AC-[A-Z0-9-]+-\d+`)
}

func fileTimestamp(path string) string {
	info, err := osStat(path)
	if err != nil {
		return ""
	}
	return info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00")
}
