package adapter

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironWithToolPathPrependsWellKnownDirs(t *testing.T) {
	env := EnvironWithToolPath([]string{
		"FOO=bar",
		"PATH=/usr/bin:/bin",
	})

	assert.Contains(t, env, "FOO=bar")
	pathValue := envValue(env, "PATH")
	require.NotEmpty(t, pathValue)

	parts := strings.Split(pathValue, string(os.PathListSeparator))
	assert.Equal(t, "/opt/homebrew/bin", parts[0])
	assert.Contains(t, parts, "/usr/local/bin")
	assert.Contains(t, parts, "/usr/bin")
	assert.Contains(t, parts, "/bin")
}

func TestEnvironWithToolPathUsesLastInputPathAndDedupes(t *testing.T) {
	env := EnvironWithToolPath([]string{
		"PATH=/first",
		"PATH=/usr/local/bin:/custom",
	})

	pathValue := envValue(env, "PATH")
	assert.Contains(t, strings.Split(pathValue, string(os.PathListSeparator)), "/custom")
	assert.NotContains(t, strings.Split(pathValue, string(os.PathListSeparator)), "/first")
	assert.Equal(t, 1, strings.Count(pathValue, "/usr/local/bin"))
	assert.Equal(t, 1, strings.Count(strings.Join(env, "\n"), "PATH="))
}
