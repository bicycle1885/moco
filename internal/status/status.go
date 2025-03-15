// internal/status/status.go
package status

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/git"
)

// StatusOptions defines status display options
type StatusOptions struct {
	DetailLevel string // Level of detail to show (minimal, normal, full)
	Format      string // Output format (text, json, markdown)
}

// ProjectStats contains project statistics
type ProjectStats struct {
	TotalExperiments int       `json:"total_experiments"`
	SuccessCount     int       `json:"success_count"`
	FailureCount     int       `json:"failure_count"`
	RunningCount     int       `json:"running_count"`
	DiskUsage        string    `json:"disk_usage"`
	DiskUsageBytes   int64     `json:"disk_usage_bytes"`
	RecentRuns       []RunInfo `json:"recent_runs,omitempty"`
}

// RunInfo contains information about a specific run
type RunInfo struct {
	Directory  string    `json:"directory"`
	Command    string    `json:"command"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Duration   string    `json:"duration,omitempty"`
	ExitStatus int       `json:"exit_status"`
	IsRunning  bool      `json:"is_running"`
	Branch     string    `json:"branch"`
	CommitHash string    `json:"commit_hash"`
}

// Show displays project status
func Show(opts StatusOptions) error {
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

// getProjectStats computes statistics about experiments
func getProjectStats(baseDir string, includeRecentRuns bool) (ProjectStats, error) {
	stats := ProjectStats{
		RecentRuns: []RunInfo{},
	}

	// Ensure base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return stats, nil // Return empty stats if directory doesn't exist
	}

	// Pattern for experiment directories
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

		// Check if it's an experiment directory
		dirName := filepath.Base(path)
		matches := pattern.FindStringSubmatch(dirName)
		if len(matches) != 4 {
			return nil // Not an experiment directory
		}

		// It's an experiment directory
		stats.TotalExperiments++

		// Parse summary file for status
		summaryPath := filepath.Join(path, "Summary.md")
		runInfo, err := parseRunInfo(summaryPath, dirName, matches)
		if err != nil {
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

		// Don't recurse into experiment directories
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

// parseRunInfo extracts info from a Summary.md file
func parseRunInfo(summaryPath, dirName string, matches []string) (RunInfo, error) {
	runInfo := RunInfo{
		Directory:  dirName,
		Branch:     matches[2],
		CommitHash: matches[3],
		IsRunning:  true, // Assume running until we find evidence it's finished
	}

	// Parse timestamp from directory name
	timestamp := matches[1]
	startTime, err := time.Parse("2006-01-02T15:04:05.000", timestamp)
	if err != nil {
		return runInfo, fmt.Errorf("unable to parse timestamp: %w", err)
	}
	runInfo.StartTime = startTime

	// Open summary file
	file, err := os.Open(summaryPath)
	if err != nil {
		return runInfo, fmt.Errorf("failed to open summary file: %w", err)
	}
	defer file.Close()

	// Scan for relevant information
	scanner := bufio.NewScanner(file)
	var commandFound bool

	for scanner.Scan() {
		line := scanner.Text()

		// Extract command
		if strings.Contains(line, "**Command:**") {
			parts := strings.SplitN(line, "`", 3)
			if len(parts) >= 2 {
				runInfo.Command = parts[1]
				commandFound = true
			}
		}

		// Check if execution has finished
		if strings.Contains(line, "**Exit status:**") {
			runInfo.IsRunning = false
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				status, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					runInfo.ExitStatus = status
				}
			}
		}

		// Extract end time
		if strings.Contains(line, "**Execution finished:**") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				endTime, err := time.Parse("2006-01-02T15:04:05", strings.TrimSpace(parts[1]))
				if err == nil {
					runInfo.EndTime = endTime
				}
			}
		}

		// Extract duration
		if strings.Contains(line, "**Execution time:**") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				runInfo.Duration = strings.TrimSpace(parts[1])
			}
		}
	}

	if !commandFound {
		return runInfo, fmt.Errorf("command not found in summary")
	}

	return runInfo, nil
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
	fmt.Printf("  Total experiments: %d\n", stats.TotalExperiments)
	fmt.Printf("  Success rate: %.1f%% (%d/%d)\n",
		percentOrZero(stats.SuccessCount, stats.SuccessCount+stats.FailureCount),
		stats.SuccessCount, stats.SuccessCount+stats.FailureCount)
	fmt.Printf("  Disk usage: %s\n", stats.DiskUsage)

	// Show running experiments if any
	if stats.RunningCount > 0 {
		fmt.Printf("\nRunning Experiments: %d\n", stats.RunningCount)

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

	// Show recent completed experiments if requested
	if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
		fmt.Println("\nRecent Completed Experiments:")
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
	fmt.Printf("- **Branch:** `%s`\n", repo.Branch)
	fmt.Printf("- **Commit:** `%s`\n", repo.ShortHash)
	if repo.IsDirty {
		fmt.Println("- **Status:** Dirty (has uncommitted changes)")
	} else {
		fmt.Println("- **Status:** Clean")
	}

	// Output basic project stats
	fmt.Println("\n## Project Statistics")
	fmt.Printf("- **Total experiments:** %d\n", stats.TotalExperiments)
	fmt.Printf("- **Success rate:** %.1f%% (%d/%d)\n",
		percentOrZero(stats.SuccessCount, stats.SuccessCount+stats.FailureCount),
		stats.SuccessCount, stats.SuccessCount+stats.FailureCount)
	fmt.Printf("- **Disk usage:** %s\n", stats.DiskUsage)

	// Show running experiments if any
	if stats.RunningCount > 0 {
		fmt.Printf("\n## Running Experiments: %d\n", stats.RunningCount)

		if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
			for _, run := range stats.RecentRuns {
				if run.IsRunning {
					elapsed := time.Since(run.StartTime).Round(time.Second)
					fmt.Printf("\n### %s\n", run.Directory)
					fmt.Printf("- **Running for:** %s\n", elapsed)
					fmt.Printf("- **Command:** `%s`\n", run.Command)
					fmt.Printf("- **Branch:** `%s`\n", run.Branch)
				}
			}
		}
	}

	// Show recent completed experiments if requested
	if detailLevel != "minimal" && len(stats.RecentRuns) > 0 {
		fmt.Println("\n## Recent Completed Experiments")
		for _, run := range stats.RecentRuns {
			if !run.IsRunning {
				status := "Success"
				if run.ExitStatus != 0 {
					status = fmt.Sprintf("Failed (exit: %d)", run.ExitStatus)
				}
				fmt.Printf("\n### %s\n", run.Directory)
				fmt.Printf("- **Status:** %s\n", status)
				fmt.Printf("- **Command:** `%s`\n", run.Command)
				fmt.Printf("- **Duration:** %s\n", run.Duration)
				fmt.Printf("- **Branch:** `%s`\n", run.Branch)
			}
		}
	}

	// Show detailed info if requested
	if detailLevel == "full" {
		fmt.Println("\n## Detailed Git Information")
		if repo.CommitMessage != "" {
			fmt.Printf("- **Last commit:** %s\n", strings.Split(repo.CommitMessage, "\n")[0])
			fmt.Printf("- **Author:** %s\n", repo.CommitAuthor)
			fmt.Printf("- **Date:** %s\n", repo.CommitDate.Format(time.RFC1123))
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
