package utils_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bicycle1885/moco/internal/git"
	"github.com/bicycle1885/moco/internal/utils"
)

func TestWriteSummaryFileInit(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("Valid summary file", func(t *testing.T) {
		summaryPath := filepath.Join(tempDir, "summary.md")
		startTime, _ := time.Parse("2006-01-02T15:04:05", "2023-01-02T15:04:05")
		endTime, _ := time.Parse("2006-01-02T15:04:05", "2023-01-02T15:05:05")
		repo := git.RepoStatus{
			Branch: "main",
		}
		commmand := []string{"sleep", "5"}
		exitCode := 0
		interrupted := false
		{
			err := utils.WriteSummaryFileInit(summaryPath, startTime, repo, commmand)
			assert.NoError(t, err)
		}
		{
			err := utils.WriteSummaryFileEnd(summaryPath, startTime, endTime, exitCode, interrupted)
			assert.NoError(t, err)
		}
	})
}

func TestParseRunInfo(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("Valid summary file", func(t *testing.T) {
		summaryContent := `# Experiment Summary

## Metadata
- **Execution datetime**: 2023-01-02T15:04:05
- **Branch**: ` + "`main`" + `
- **Commit hash**: ` + "`abcd1234`" + `
- **Command**: ` + "`go test ./...`" + `
- **Hostname**: ` + "`test-host`" + `
- **Working directory**: ` + "`runs/2023-01-02T15:04:05.000_main_abcd1234`" + `

## Latest Commit Details
` + "```diff" + `
commit abcd1234
Author: Test User <test@example.com>
Date:   Mon, 2 Jan 2023 15:00:00 +0100

    Test commit
` + "```" + `

## Environment Info
` + "```" + `
Test environment
` + "```" + `

## Execution Results
- **Execution finished**: 2023-01-02T15:05:05
- **Execution time**: 1m0s
- **Exit status**: 0
`

		// Create test summary file
		summaryPath := filepath.Join(tempDir, "summary.md")
		err := os.WriteFile(summaryPath, []byte(summaryContent), 0644)
		assert.NoError(t, err)

		// Parse the summary file
		info, err := utils.ParseRunInfo(summaryPath)
		assert.NoError(t, err)

		// Verify parsed information
		startTime, _ := time.Parse("2006-01-02T15:04:05", "2023-01-02T15:04:05")
		endTime, _ := time.Parse("2006-01-02T15:04:05", "2023-01-02T15:05:05")

		assert.Equal(t, tempDir+"/", info.Directory)
		assert.Equal(t, "summary.md", info.File)
		assert.Equal(t, "go test ./...", info.Command)
		assert.Equal(t, startTime, info.StartTime)
		assert.Equal(t, endTime, info.EndTime)
		assert.Equal(t, "1m0s", info.Duration)
		assert.Equal(t, 0, info.ExitStatus)
		assert.False(t, info.IsRunning)
		assert.Equal(t, "main", info.Branch)
		assert.Equal(t, "abcd1234", info.CommitHash)
		assert.False(t, info.Interrupted)
	})

	t.Run("Interrupted run", func(t *testing.T) {
		summaryContent := `# Experiment Summary

## Metadata
- **Execution datetime**: 2023-01-02T15:04:05
- **Branch**: ` + "`main`" + `
- **Commit hash**: ` + "`abcd1234`" + `
- **Command**: ` + "`go test ./...`" + `
- **Hostname**: ` + "`test-host`" + `
- **Working directory**: ` + "`runs/2023-01-02T15:04:05.000_main_abcd1234`" + `

## Execution Results
- **Execution finished**: 2023-01-02T15:05:05
- **Execution time**: 1m0s
- **Exit status**: 130
- **Terminated by user**
`

		summaryPath := filepath.Join(tempDir, "summary_interrupted.md")
		err := os.WriteFile(summaryPath, []byte(summaryContent), 0644)
		assert.NoError(t, err)

		info, err := utils.ParseRunInfo(summaryPath)
		assert.NoError(t, err)
		assert.Equal(t, 130, info.ExitStatus)
		assert.True(t, info.Interrupted)
	})

	t.Run("Non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "non_existent.md")
		_, err := utils.ParseRunInfo(nonExistentPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open summary file")
	})
}
