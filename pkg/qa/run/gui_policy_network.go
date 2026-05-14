package run

import (
	"net/url"
	"strings"
)

func outsideAllowedNetworkRequests(doc map[string]any, allowedOrigins []string) []string {
	requests, ok := doc["requests"].([]any)
	if !ok {
		return nil
	}
	allowed := allowedOriginSet(allowedOrigins)
	outside := []string{}
	for _, request := range requests {
		label, origin, ok := requestLabelAndOrigin(request)
		if !ok || !allowed[origin] {
			outside = append(outside, "network_request_outside_allowed:"+label)
		}
	}
	return outside
}

func requestLabelAndOrigin(request any) (string, string, bool) {
	switch typed := request.(type) {
	case string:
		origin, ok := originFromURL(typed)
		return redactedURLLabel(typed), origin, ok
	case map[string]any:
		if value := firstText(typed, "url", "origin"); value != "" {
			origin, ok := originFromURL(value)
			label := redactedURLLabel(value)
			return label, origin, ok
		}
		return "missing_origin", "", false
	default:
		return "missing_origin", "", false
	}
}

func originFromURL(value string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port != "" && !defaultPort(scheme, port) {
		host += ":" + port
	}
	return scheme + "://" + host, true
}

func defaultPort(scheme, port string) bool {
	return (scheme == "http" && port == "80") || (scheme == "https" && port == "443")
}

func redactedURLLabel(value string) string {
	origin, ok := originFromURL(value)
	if !ok {
		return "invalid_url"
	}
	parsed, _ := url.Parse(strings.TrimSpace(value))
	if parsed.Path != "" && parsed.Path != "/" {
		return origin + parsed.EscapedPath()
	}
	return origin
}

func allowedOriginSet(origins []string) map[string]bool {
	out := map[string]bool{}
	for _, origin := range origins {
		if normalized, ok := originFromURL(origin); ok {
			out[normalized] = true
		}
	}
	return out
}
