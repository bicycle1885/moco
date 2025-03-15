// internal/experiment/run.go
package experiment

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/git"
)

// RunOptions contains options for running an experiment
type RunOptions struct {
	Command       []string
	Force         bool
	BaseDir       string
	NoPushd       bool
	CleanupOnFail bool
}

// Run executes a command with experiment tracking
func Run(opts RunOptions) error {
	// Get config
	cfg := config.GetConfig()

	// Check git repository status
	repo, err := git.GetRepoStatus()
	if err != nil {
		return fmt.Errorf("git repository error: %w", err)
	}

	// Validate git status
	if repo.IsDirty && !opts.Force && cfg.Git.RequireClean {
		return fmt.Errorf("git repository has uncommitted changes, use --force to override")
	}

	// Create experiment directory with millisecond timestamp
	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = cfg.Paths.BaseDir
	}

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create unique experiment directory
	timestamp := time.Now().Format("2006-01-02T15:04:05.000")
	dirName := fmt.Sprintf("%s_%s_%s", timestamp, sanitizeName(repo.Branch), repo.ShortHash)
	expDir := filepath.Join(baseDir, dirName)

	if err := os.Mkdir(expDir, 0755); err != nil {
		return fmt.Errorf("failed to create experiment directory: %w", err)
	}

	// Set up signal handling for clean termination
	interrupted := false
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Create summary file
	startTime := time.Now()
	summaryPath := filepath.Join(expDir, cfg.Paths.SummaryFile)
	if err := writeSummary(summaryPath, startTime, repo, opts.Command, expDir); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	// Set up output files
	stdoutPath := filepath.Join(expDir, cfg.Paths.StdoutFile)
	stderrPath := filepath.Join(expDir, cfg.Paths.StderrFile)

	// Execute command
	cmd := exec.Command(opts.Command[0], opts.Command[1:]...)

	// Set working directory if required
	if !opts.NoPushd {
		cmd.Dir = expDir
	}

	// Set up files for capturing output
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return fmt.Errorf("failed to create stdout file: %w", err)
	}
	defer stdoutFile.Close()

	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		return fmt.Errorf("failed to create stderr file: %w", err)
	}
	defer stderrFile.Close()

	// Capture outputs while also displaying them
	cmd.Stdout = io.MultiWriter(os.Stdout, stdoutFile)
	cmd.Stderr = io.MultiWriter(os.Stderr, stderrFile)

	// Start the command
	fmt.Printf("Running command in %s: %s\n", expDir, strings.Join(opts.Command, " "))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for either command completion or signal
	exitCode := 0
	doneChan := make(chan error, 1)

	go func() {
		doneChan <- cmd.Wait()
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}
	case sig := <-signalChan:
		interrupted = true
		fmt.Printf("Received signal: %v\n", sig)

		if cmd.Process != nil {
			if err := cmd.Process.Signal(sig); err != nil {
				fmt.Printf("Failed to send signal to process: %v\n", err)
			}
		}

		<-doneChan
		exitCode = 130 // Convention for interrupted commands
	}

	// Record execution results
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Update summary with results
	appendSummaryResults(summaryPath, endTime, duration, exitCode, interrupted)

	// Handle cleanup on failure
	if exitCode != 0 && opts.CleanupOnFail {
		fmt.Printf("Command failed. Cleaning up directory: %s\n", expDir)
		os.RemoveAll(expDir)
	}

	if exitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", exitCode)
	}

	return nil
}

// Helper function implementations (sanitizeName, writeSummary, appendSummaryResults)
func sanitizeName(name string) string {
	// Replace problematic characters with underscores
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", " ", "_")
	return r.Replace(name)
}

// writeSummary creates the initial summary.md file with experiment metadata
func writeSummary(path string, startTime time.Time, repo git.RepoStatus,
	command []string, expDir string) error {
	// Create the summary file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}
	defer file.Close()

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Format the command as a string
	commandStr := strings.Join(command, " ")

	// Get git commit details
	commitDetails, err := git.GetCommitDetails()
	if err != nil {
		commitDetails = "Error retrieving commit details"
	}

	// Get git status
	gitStatus, err := git.GetRepoStatus()
	if err != nil {
		gitStatus = git.RepoStatus{IsValid: false}
	}

	// Get git diff
	gitDiff, err := git.GetUncommittedChanges()
	if err != nil {
		gitDiff = "Error retrieving uncommitted changes"
	}

	// Get system info
	var sysInfo strings.Builder
	cmd := exec.Command("uname", "-a")
	cmd.Stdout = &sysInfo
	if err := cmd.Run(); err != nil {
		sysInfo.WriteString(fmt.Sprintf("Error retrieving system info: %v", err))
	}

	// Use string builder to avoid backtick issues
	var b strings.Builder

	// Header
	b.WriteString("# Experiment Summary\n\n")

	// Metadata
	b.WriteString("## Metadata\n")
	fmt.Fprintf(&b, "- **Execution datetime:** %s\n", startTime.Format("2006-01-02T15:04:05"))
	fmt.Fprintf(&b, "- **Branch:** `%s`\n", repo.Branch)
	fmt.Fprintf(&b, "- **Commit hash:** `%s`\n", repo.FullHash)
	fmt.Fprintf(&b, "- **Command:** `%s`\n", commandStr)
	fmt.Fprintf(&b, "- **Hostname:** `%s`\n", hostname)
	fmt.Fprintf(&b, "- **Working directory:** `%s`\n", expDir)

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
	b.WriteString(sysInfo.String())
	b.WriteString("\n```\n")

	// Write to file
	_, err = file.WriteString(b.String())
	if err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

// formatGitStatus converts git status to a string for display
func formatGitStatus(repo git.RepoStatus) string {
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

// appendSummaryResults appends execution results to the summary.md
func appendSummaryResults(path string, endTime time.Time, duration time.Duration,
	exitCode int, interrupted bool) error {
	// Open file in append mode
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open summary file: %w", err)
	}
	defer file.Close()

	// Format duration in a human-readable way
	durationStr := formatDuration(duration)

	// Create the results section
	results := fmt.Sprintf(`
## Execution Results
- **Execution finished:** %s
- **Execution time:** %s
- **Exit status:** %d
`,
		endTime.Format("2006-01-02T15:04:05"),
		durationStr,
		exitCode,
	)

	// Add interrupted note if applicable
	if interrupted {
		results += "- **Terminated by user**\n"
	}

	// Append results to file
	_, err = file.WriteString(results)
	if err != nil {
		return fmt.Errorf("failed to append results: %w", err)
	}

	return nil
}

// formatDuration formats a duration in a human-readable way (XhYmZs)
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
