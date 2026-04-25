package spec_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shin0720/auto-adk/pkg/spec"
)

func TestValidationError_Error(t *testing.T) {
	t.Parallel()
	e := spec.ValidationError{Field: "spec.md", Message: "missing required section", Level: "error"}
	assert.Equal(t, "missing required section", e.Error())
}

func TestValidationError_ErrorEmpty(t *testing.T) {
	t.Parallel()
	e := spec.ValidationError{}
	assert.Equal(t, "", e.Error())
}
