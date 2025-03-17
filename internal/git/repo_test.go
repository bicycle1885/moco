package git_test

import (
	"testing"

	"github.com/bicycle1885/moco/internal/git"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeBranchName(t *testing.T) {
	t.Run("Valid branch name", func(t *testing.T) {
		branchName := "main"
		sanitized := git.SanitizeBranchName(branchName)
		assert.Equal(t, "main", sanitized)
	})

	t.Run("Invalid branch name", func(t *testing.T) {
		branchName := "foo/bar"
		sanitized := git.SanitizeBranchName(branchName)
		assert.Equal(t, "foo-bar", sanitized)
	})
}
