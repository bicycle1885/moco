package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Timezone is indispensable for correct parsing of timestamps
// RFC3339: "2006-01-02T15:04:05Z07:00"
const timestampFormat = time.RFC3339

// RunInfo contains information about a specific run
type RunInfo struct {
	Directory   string    `json:"directory"`
	File        string    `json:"file_name"`
	Command     string    `json:"command"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	ExitStatus  int       `json:"exit_status"`
	IsRunning   bool      `json:"is_running"`
	Branch      string    `json:"branch"`
	CommitHash  string    `json:"commit_hash"`
	Interrupted bool      `json:"interrupted"`
}

// Duration returns a formatted duration of the run
func (r *RunInfo) Duration() string {
	var d time.Duration

	// Check if the run is still running
	if r.IsRunning || r.EndTime.IsZero() {
		// Calculate duration from start to now
		d = time.Since(r.StartTime)
	} else {
		// Calculate duration from start to end
		d = r.EndTime.Sub(r.StartTime)
	}

	// Use the existing formatDuration function
	return formatDuration(d)
}

func WriteSummaryFileInit(summaryPath string, startTime time.Time, repo RepoStatus, command []string) error {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Get working directory
	directry, _ := filepath.Split(summaryPath)

	// Get git commit details
	commitDetails, err := GetCommitDetails()
	if err != nil {
		commitDetails = "Error retrieving commit details"
	}

	// Get git status
	gitStatus, err := GetRepoStatus()
	if err != nil {
		gitStatus = RepoStatus{IsValid: false}
	}

	// Get git diff
	gitDiff, err := GetUncommittedChanges()
	if err != nil {
		gitDiff = "Error retrieving uncommitted changes"
	}

	// Get system info
	sysInfo := getSystemInfo()

	// Construct metadata section
	var b strings.Builder

	// Header
	b.WriteString("# Experiment Summary\n\n")

	// Metadata
	b.WriteString("## Metadata\n")
	fmt.Fprintf(&b, "- **Execution datetime**: %s\n", startTime.Format(timestampFormat))
	fmt.Fprintf(&b, "- **Branch**: `%s`\n", repo.Branch)
	fmt.Fprintf(&b, "- **Commit hash**: `%s`\n", repo.FullHash)
	fmt.Fprintf(&b, "- **Command**: `%s`\n", strings.Join(command, " "))
	fmt.Fprintf(&b, "- **Hostname**: `%s`\n", hostname)
	fmt.Fprintf(&b, "- **Working directory**: `%s`\n", directry)

	// Commit details
	b.WriteString("\n## Latest Commit Details\n")
	b.WriteString("```diff\n")
	b.WriteString(commitDetails)
	b.WriteString("\n```\n")

	// Git status
	b.WriteString("\n## Git Status\n")
	b.WriteString("```\n")
	b.WriteString(formatGitStatus(gitStatus))
	b.WriteString("\n```\n")

	// Git diff
	b.WriteString("\n## Uncommitted Changes (Diff)\n")
	b.WriteString("```diff\n")
	b.WriteString(gitDiff)
	b.WriteString("\n```\n")

	// System info
	b.WriteString("\n## Environment Info\n")
	b.WriteString("```\n")
	b.WriteString(sysInfo)
	b.WriteString("\n```\n")

	// Create summary file
	file, err := os.Create(summaryPath)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}
	defer file.Close()

	// Write metadata to file
	if _, err := file.WriteString(b.String()); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// getSystemInfo retrieves system information
func getSystemInfo() string {
	var sysInfo strings.Builder
	cmd := exec.Command("uname", "-a")
	cmd.Stdout = &sysInfo
	if err := cmd.Run(); err != nil {
		sysInfo.WriteString(fmt.Sprintf("Error retrieving system info: %v", err))
	}
	return sysInfo.String()
}

// formatGitStatus converts git status to a string for display
func formatGitStatus(repo RepoStatus) string {
	if !repo.IsValid {
		return "Not a valid git repository"
	}

	status := fmt.Sprintf("On branch %s\n", repo.Branch)
	if repo.IsDirty {
		status += "Changes not staged for commit:\n"
		status += "  (use \"git add <file>...\" to update what will be committed)\n"
		status += "  (use \"git restore <file>...\" to discard changes in working directory)\n"
		status += "\n"
		status += "        modified:   [uncommitted changes present]\n"
	} else {
		status += "nothing to commit, working tree clean\n"
	}

	return status
}

func WriteSummaryFileEnd(summaryPath string, startTime, endTime time.Time, exitCode int, interrupted bool) error {
	// Open the summary file
	file, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open summary file: %w", err)
	}
	defer file.Close()

	// Create the results section
	results := fmt.Sprintf(`
## Execution Results
- **Execution finished**: %s
- **Execution time**: %s
- **Exit status**: %d
`, endTime.Format(timestampFormat), formatDuration(endTime.Sub(startTime)), exitCode)

	if interrupted {
		results += "- **Terminated by user**\n"
	}

	// Write results to file
	if _, err := file.WriteString(results); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	return nil
}

// formatDuration formats a duration in a human-readable way (Xh Ym Zs)
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// ParseRunInfo extracts info from a summary file
func ParseRunInfo(summaryPath string) (RunInfo, error) {
	dirName, fileName := filepath.Split(summaryPath)
	runInfo := RunInfo{
		Directory: dirName,
		File:      fileName,
		IsRunning: true,
	}

	// Open summary file
	file, err := os.Open(summaryPath)
	if err != nil {
		return runInfo, fmt.Errorf("failed to open summary file: %w", err)
	}
	defer file.Close()

	// Scan for relevant information
	scanner := bufio.NewScanner(file)
	withinCodeBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "```") {
			// Toggle code block state
			withinCodeBlock = !withinCodeBlock
		}

		if withinCodeBlock {
			// Skip lines within code blocks
			continue
		}

		if after, found := strings.CutPrefix(line, "- **Execution datetime**: "); found {
			// Extract start time
			startTime, err := time.Parse(timestampFormat, after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse start time: %w", err)
			}
			runInfo.StartTime = startTime
		} else if after, found := strings.CutPrefix(line, "- **Branch**: "); found {
			branch, err := trimBackticks(after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse branch: %w", err)
			}
			runInfo.Branch = branch
		} else if after, found := strings.CutPrefix(line, "- **Commit hash**: "); found {
			commitHash, err := trimBackticks(after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse commit hash: %w", err)
			}
			runInfo.CommitHash = commitHash
		} else if after, found := strings.CutPrefix(line, "- **Command**: "); found {
			// Extract command
			command, err := trimBackticks(after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse command: %w", err)
			}
			runInfo.Command = command
		} else if after, found := strings.CutPrefix(line, "- **Exit status**: "); found {
			runInfo.IsRunning = false
			// Extract exit status
			status, err := strconv.Atoi(after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse exit status: %w", err)
			}
			runInfo.ExitStatus = status
		} else if after, found := strings.CutPrefix(line, "- **Execution finished**: "); found {
			// Extract end time
			endTime, err := time.Parse(timestampFormat, after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse end time: %w", err)
			}
			runInfo.EndTime = endTime
		} else if strings.Contains(line, "**Terminated by user**") {
			// Check if interrupted
			runInfo.Interrupted = true
		}
	}

	return runInfo, nil
}

// trimBackticks removes backticks from the both ends of a string
func trimBackticks(s string) (string, error) {
	if len(s) < 2 || s[0] != '`' || s[len(s)-1] != '`' {
		return "", fmt.Errorf("string is not enclosed in backticks")
	}
	return s[1 : len(s)-1], nil
}
