package status

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/utils"
	"github.com/charmbracelet/log"
)

// ProjectStats contains project statistics
type ProjectStats struct {
	DiskUsage    int64           `json:"disk_usage"`
	RunningCount int             `json:"running_count"`
	FailureCount int             `json:"failure_count"`
	SuccessCount int             `json:"success_count"`
	TotalRuns    int             `json:"total_runs"`
	RecentRuns   []utils.RunInfo `json:"recent_runs,omitempty"`
}

const maxRecentRuns = 5

// Show displays project status
func Show() error {
	// Get config and repository status
	cfg := config.Get()
	repo, err := utils.GetRepoStatus()
	if err != nil {
		return fmt.Errorf("failed to get git status: %w", err)
	}

	// Get project statistics
	level := cfg.Status.Level
	stats, err := getProjectStats(cfg.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to get project statistics: %w", err)
	}

	// Display status based on detail level
	return outputStatusText(repo, stats, level)
}

// getProjectStats computes statistics about runs
func getProjectStats(baseDir string) (ProjectStats, error) {
	stats := ProjectStats{
		RecentRuns: []utils.RunInfo{},
	}

	// Ensure base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return stats, nil // Return empty stats if directory doesn't exist
	}

	// Get config
	cfg := config.Get()

	// Pattern for runs directories
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3})_(.+)_([a-f0-9]{7})$`)

	// Walk the base directory to gather stats
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the base directory
		if !info.IsDir() || path == baseDir {
			if !info.IsDir() {
				stats.DiskUsage += info.Size() // Add file size to total
			}
			return nil
		}

		// Add directory size to total
		size, err := dirSize(path)
		if err != nil {
			return fmt.Errorf("failed to get directory size: %w", err)
		}
		stats.DiskUsage += size

		// Check if it's a run directory
		dirName := filepath.Base(path)
		matches := pattern.FindStringSubmatch(dirName)
		if len(matches) != 4 {
			return nil // Not a run directory
		}

		// Parse summary file for status
		summaryPath := filepath.Join(path, cfg.SummaryFile)
		runInfo, err := utils.ParseRunInfo(summaryPath)
		if err != nil {
			log.Warnf("Failed to parse summary file: %v", err)
			return nil
		}

		stats.RecentRuns = append(stats.RecentRuns, runInfo)

		// Don't recurse into run directories
		return filepath.SkipDir
	})

	if err != nil {
		return stats, fmt.Errorf("error walking directory: %w", err)
	}

	// Count running, success, and failure runs
	for _, run := range stats.RecentRuns {
		stats.TotalRuns++
		if run.IsRunning {
			stats.RunningCount++
		} else if run.ExitStatus == 0 {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
	}

	// Reverse the list to show most recent runs first
	slices.Reverse(stats.RecentRuns)

	return stats, nil
}

// dirSize computes the size of a directory
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// formatSize formats a file size in bytes to human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// outputStatusText outputs status in text format
func outputStatusText(repo utils.RepoStatus, stats ProjectStats, detailLevel string) error {
	// Output git information
	fmt.Println("Git Repository Status:")
	fmt.Printf("  Branch: %s\n", repo.Branch)
	fmt.Printf("  Commit: %s\n", repo.ShortHash)
	if repo.IsDirty {
		fmt.Println("  Status: Dirty (has uncommitted changes)")
	} else {
		fmt.Println("  Status: Clean")
	}

	// Show detailed info if requested
	if detailLevel == "full" {
		fmt.Println("\nDetailed Git Information:")
		if repo.CommitMessage != "" {
			fmt.Printf("  Last commit: %s\n", strings.Split(repo.CommitMessage, "\n")[0])
			fmt.Printf("  Author: %s\n", repo.CommitAuthor)
			fmt.Printf("  Date: %s\n", repo.CommitDate.Format(time.RFC1123))
		}
	}

	// Output basic project stats
	fmt.Println("\nProject Statistics:")
	fmt.Printf("  Total runs: %d\n", stats.TotalRuns)
	fmt.Printf("  Success rate: %.1f%% (%d/%d)\n",
		percentOrZero(stats.SuccessCount, stats.SuccessCount+stats.FailureCount),
		stats.SuccessCount, stats.SuccessCount+stats.FailureCount)
	fmt.Printf("  Disk usage: %s\n", formatSize(stats.DiskUsage))

	// Show recent runs if requested
	if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
		fmt.Println("\nRecent Runs:")
		for _, run := range stats.RecentRuns[:min(maxRecentRuns, len(stats.RecentRuns))] {
			status := statusString(run)
			fmt.Printf("  • %s\n    Status: %s\n    Command: %s\n    Duration: %s\n",
				run.Directory, status, run.Command, run.Duration())
		}
	}

	return nil
}

// percentOrZero calculates percentage and returns 0 if denominator is 0
func percentOrZero(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return 100.0 * float64(numerator) / float64(denominator)
}

func statusString(run utils.RunInfo) string {
	if run.IsRunning {
		return "Running"
	} else if run.ExitStatus == 0 {
		return "Success"
	} else {
		return fmt.Sprintf("Failed (exit: %d)", run.ExitStatus)
	}
}
