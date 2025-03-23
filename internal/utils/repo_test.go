package utils_test

import (
	"testing"

	"github.com/bicycle1885/moco/internal/utils"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeBranchName(t *testing.T) {
	t.Run("Valid branch name", func(t *testing.T) {
		branchName := "main"
		sanitized := utils.SanitizeBranchName(branchName)
		assert.Equal(t, "main", sanitized)
	})

	t.Run("Invalid branch name", func(t *testing.T) {
		branchName := "foo/bar"
		sanitized := utils.SanitizeBranchName(branchName)
		assert.Equal(t, "foo-bar", sanitized)
	})
}
