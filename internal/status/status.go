package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/git"
	"github.com/bicycle1885/moco/internal/utils"
)

// Options defines status display options
type Options struct {
	DetailLevel string // Level of detail to show (minimal, normal, full)
	Format      string // Output format (text, json, markdown)
}

// ProjectStats contains project statistics
type ProjectStats struct {
	TotalRuns      int             `json:"total_runs"`
	SuccessCount   int             `json:"success_count"`
	FailureCount   int             `json:"failure_count"`
	RunningCount   int             `json:"running_count"`
	DiskUsage      string          `json:"disk_usage"`
	DiskUsageBytes int64           `json:"disk_usage_bytes"`
	RecentRuns     []utils.RunInfo `json:"recent_runs,omitempty"`
}

// Show displays project status
func Show(opts Options) error {
	// Get config and repository status
	cfg := config.GetConfig()
	repo, err := git.GetRepoStatus()
	if err != nil {
		return fmt.Errorf("failed to get git status: %w", err)
	}

	// Get project statistics
	stats, err := getProjectStats(cfg.Paths.BaseDir, opts.DetailLevel != "minimal")
	if err != nil {
		return fmt.Errorf("failed to get project statistics: %w", err)
	}

	// Display status based on format and detail level
	switch opts.Format {
	case "json":
		return outputStatusJSON(repo, stats, opts.DetailLevel)
	case "markdown":
		return outputStatusMarkdown(repo, stats, opts.DetailLevel)
	default: // text
		return outputStatusText(repo, stats, opts.DetailLevel)
	}
}

// getProjectStats computes statistics about runs
func getProjectStats(baseDir string, includeRecentRuns bool) (ProjectStats, error) {
	stats := ProjectStats{
		RecentRuns: []utils.RunInfo{},
	}

	// Ensure base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return stats, nil // Return empty stats if directory doesn't exist
	}

	// Get config
	cfg := config.GetConfig()

	// Pattern for runs directories
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3})_(.+)_([a-f0-9]{7})$`)

	// Walk the base directory to gather stats
	var totalSize int64
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the base directory
		if !info.IsDir() || path == baseDir {
			if !info.IsDir() {
				totalSize += info.Size() // Add file size to total
			}
			return nil
		}

		// Add directory size to total
		size, err := dirSize(path)
		if err != nil {
			return fmt.Errorf("failed to get directory size: %w", err)
		}
		totalSize += size

		// Check if it's a run directory
		dirName := filepath.Base(path)
		matches := pattern.FindStringSubmatch(dirName)
		if len(matches) != 4 {
			return nil // Not a run directory
		}

		// It's a run directory
		stats.TotalRuns++

		// Parse summary file for status
		summaryPath := filepath.Join(path, cfg.Paths.SummaryFile)
		runInfo, err := utils.ParseRunInfo(summaryPath)
		if err != nil {
			// TODO: fix this
			// If we can't parse the summary, assume it's still running
			stats.RunningCount++
			return nil
		}

		if runInfo.IsRunning {
			stats.RunningCount++
		} else if runInfo.ExitStatus == 0 {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}

		// Add to recent runs if requested
		if includeRecentRuns && len(stats.RecentRuns) < 5 {
			stats.RecentRuns = append(stats.RecentRuns, runInfo)
		}

		// Don't recurse into run directories
		return filepath.SkipDir
	})

	if err != nil {
		return stats, fmt.Errorf("error walking directory: %w", err)
	}

	// Format disk usage
	stats.DiskUsageBytes = totalSize
	stats.DiskUsage = formatSize(totalSize)

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
func outputStatusText(repo git.RepoStatus, stats ProjectStats, detailLevel string) error {
	// Output git information
	fmt.Println("Git Repository Status:")
	fmt.Printf("  Branch: %s\n", repo.Branch)
	fmt.Printf("  Commit: %s\n", repo.ShortHash)
	if repo.IsDirty {
		fmt.Println("  Status: Dirty (has uncommitted changes)")
	} else {
		fmt.Println("  Status: Clean")
	}

	// Output basic project stats
	fmt.Println("\nProject Statistics:")
	fmt.Printf("  Total runs: %d\n", stats.TotalRuns)
	fmt.Printf("  Success rate: %.1f%% (%d/%d)\n",
		percentOrZero(stats.SuccessCount, stats.SuccessCount+stats.FailureCount),
		stats.SuccessCount, stats.SuccessCount+stats.FailureCount)
	fmt.Printf("  Disk usage: %s\n", stats.DiskUsage)

	// Show running runs if any
	if stats.RunningCount > 0 {
		fmt.Printf("\nRunning runs: %d\n", stats.RunningCount)

		if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
			for _, run := range stats.RecentRuns {
				if run.IsRunning {
					elapsed := time.Since(run.StartTime).Round(time.Second)
					fmt.Printf("  • %s (Running for %s)\n    Command: %s\n",
						run.Directory, elapsed, run.Command)
				}
			}
		}
	}

	// Show recent completed runs if requested
	if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
		fmt.Println("\nRecent Runs:")
		for _, run := range stats.RecentRuns {
			if !run.IsRunning {
				status := "Success"
				if run.ExitStatus != 0 {
					status = fmt.Sprintf("Failed (exit: %d)", run.ExitStatus)
				}
				fmt.Printf("  • %s (%s)\n    Command: %s\n    Duration: %s\n",
					run.Directory, status, run.Command, run.Duration)
			}
		}
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

	return nil
}

// outputStatusJSON outputs status in JSON format
func outputStatusJSON(repo git.RepoStatus, stats ProjectStats, detailLevel string) error {
	// Create a structure for the full status
	status := struct {
		Git struct {
			Branch        string    `json:"branch"`
			CommitHash    string    `json:"commit_hash"`
			FullHash      string    `json:"full_hash,omitempty"`
			IsDirty       bool      `json:"is_dirty"`
			CommitMessage string    `json:"commit_message,omitempty"`
			CommitAuthor  string    `json:"commit_author,omitempty"`
			CommitDate    time.Time `json:"commit_date,omitempty"`
		} `json:"git"`
		Stats ProjectStats `json:"stats"`
	}{
		Stats: stats,
	}

	// Fill git info
	status.Git.Branch = repo.Branch
	status.Git.CommitHash = repo.ShortHash
	status.Git.IsDirty = repo.IsDirty

	// Add detailed info if requested
	if detailLevel != "minimal" {
		status.Git.FullHash = repo.FullHash
	}

	if detailLevel == "full" {
		status.Git.CommitMessage = repo.CommitMessage
		status.Git.CommitAuthor = repo.CommitAuthor
		status.Git.CommitDate = repo.CommitDate
	}

	// Remove recent runs if minimal detail level
	if detailLevel == "minimal" {
		status.Stats.RecentRuns = nil
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// outputStatusMarkdown outputs status in Markdown format
func outputStatusMarkdown(repo git.RepoStatus, stats ProjectStats, detailLevel string) error {
	// Output git information
	fmt.Println("# Moco Project Status")

	fmt.Println("\n## Git Repository Status")
	fmt.Printf("- **Branch**: `%s`\n", repo.Branch)
	fmt.Printf("- **Commit**: `%s`\n", repo.ShortHash)
	if repo.IsDirty {
		fmt.Println("- **Status**: Dirty (has uncommitted changes)")
	} else {
		fmt.Println("- **Status**: Clean")
	}

	// Output basic project stats
	fmt.Println("\n## Project Statistics")
	fmt.Printf("- **Total runs**: %d\n", stats.TotalRuns)
	fmt.Printf("- **Success rate**: %.1f%% (%d/%d)\n",
		percentOrZero(stats.SuccessCount, stats.SuccessCount+stats.FailureCount),
		stats.SuccessCount, stats.SuccessCount+stats.FailureCount)
	fmt.Printf("- **Disk usage**: %s\n", stats.DiskUsage)

	// Show running runs if any
	if stats.RunningCount > 0 {
		fmt.Printf("\n## Running runs: %d\n", stats.RunningCount)

		if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
			for _, run := range stats.RecentRuns {
				if run.IsRunning {
					elapsed := time.Since(run.StartTime).Round(time.Second)
					fmt.Printf("\n### %s\n", run.Directory)
					fmt.Printf("- **Running for**: %s\n", elapsed)
					fmt.Printf("- **Command**: `%s`\n", run.Command)
					fmt.Printf("- **Branch**: `%s`\n", run.Branch)
				}
			}
		}
	}

	// Show recent runs if requested
	if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
		fmt.Println("\n## Recent Runs")
		for _, run := range stats.RecentRuns {
			if !run.IsRunning {
				status := "Success"
				if run.ExitStatus != 0 {
					status = fmt.Sprintf("Failed (exit: %d)", run.ExitStatus)
				}
				fmt.Printf("\n### %s\n", run.Directory)
				fmt.Printf("- **Status**: %s\n", status)
				fmt.Printf("- **Command**: `%s`\n", run.Command)
				fmt.Printf("- **Duration**: %s\n", run.Duration)
				fmt.Printf("- **Branch**: `%s`\n", run.Branch)
			}
		}
	}

	// Show detailed info if requested
	if detailLevel == "full" {
		fmt.Println("\n## Detailed Git Information")
		if repo.CommitMessage != "" {
			fmt.Printf("- **Last commit**: %s\n", strings.Split(repo.CommitMessage, "\n")[0])
			fmt.Printf("- **Author**: %s\n", repo.CommitAuthor)
			fmt.Printf("- **Date**: %s\n", repo.CommitDate.Format(time.RFC1123))
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
