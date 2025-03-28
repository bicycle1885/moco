package utils_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bicycle1885/moco/internal/utils"
)

func TestWriteSummaryFileInit(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("Valid summary file", func(t *testing.T) {
		summaryPath := filepath.Join(tempDir, "summary.md")
		startTime, _ := time.Parse("2006-01-02T15:04:05", "2023-01-02T15:04:05")
		endTime, _ := time.Parse("2006-01-02T15:04:05", "2023-01-02T15:05:05")
		repo := utils.RepoStatus{
			Branch: "main",
		}
		commmand := []string{"sleep", "5"}
		message := "Test message"
		exitCode := 0
		interrupted := false
		{
			err := utils.WriteSummaryFileInit(summaryPath, startTime, repo, commmand, message)
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
		// Parse the summary file
		summaryPath := filepath.Join("testdata", "summary.md")
		info, err := utils.ParseRunInfo(summaryPath)
		assert.NoError(t, err)

		// Verify parsed information
		startTime, _ := time.Parse(time.RFC3339, "2025-03-24T00:34:51+01:00")
		endTime, _ := time.Parse(time.RFC3339, "2025-03-24T00:34:56+01:00")

		assert.Equal(t, "testdata/", info.Directory)
		assert.Equal(t, "summary.md", info.File)
		assert.Equal(t, "sleep 5", info.Command)
		assert.Equal(t, startTime, info.StartTime)
		assert.Equal(t, endTime, info.EndTime)
		assert.Equal(t, "5s", info.Duration())
		assert.Equal(t, 0, info.ExitStatus)
		assert.False(t, info.IsRunning)
		assert.Equal(t, "main", info.Branch)
		assert.Equal(t, "7a9162c4ad32037a036d71e03f5a9262551a7e46", info.CommitHash)
		assert.False(t, info.Interrupted)
	})

	t.Run("Interrupted run", func(t *testing.T) {
		summaryPath := filepath.Join("testdata", "summary_interrupted.md")
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
