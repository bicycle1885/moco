package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RunInfo contains information about a specific run
type RunInfo struct {
	Directory   string    `json:"directory"`
	File        string    `json:"file_name"`
	Command     string    `json:"command"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	ExitStatus  int       `json:"exit_status"`
	IsRunning   bool      `json:"is_running"`
	Branch      string    `json:"branch"`
	CommitHash  string    `json:"commit_hash"`
	Interrupted bool      `json:"interrupted"`
}

// ParseRunInfo extracts info from a summary.md file
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
			startTime, err := parseDateTime(after)
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
			endTime, err := parseDateTime(after)
			if err != nil {
				return runInfo, fmt.Errorf("failed to parse end time: %w", err)
			}
			runInfo.EndTime = endTime
		} else if after, found := strings.CutPrefix(line, "- **Execution time**: "); found {
			// Extract duration
			runInfo.Duration = after
		} else if strings.Contains(line, "**Terminated by user**") {
			// Check if interrupted
			runInfo.Interrupted = true
		}
	}

	return runInfo, nil
}

func parseDateTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05", s)
}

// trimBackticks removes backticks from the both ends of a string
func trimBackticks(s string) (string, error) {
	if len(s) < 2 || s[0] != '`' || s[len(s)-1] != '`' {
		return "", fmt.Errorf("string is not enclosed in backticks")
	}
	return s[1 : len(s)-1], nil
}
