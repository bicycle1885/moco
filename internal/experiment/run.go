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
	"github.com/bicycle1885/moco/internal/utils"
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
	startTime := time.Now()
	dirName := fmt.Sprintf("%s_%s_%s", startTime.Format("2006-01-02T15:04:05.000"), sanitizeBranchName(repo.Branch), repo.ShortHash)
	expDir := filepath.Join(baseDir, dirName)

	if err := os.Mkdir(expDir, 0755); err != nil {
		return fmt.Errorf("failed to create experiment directory: %w", err)
	}

	// Set up signal handling for clean termination
	interrupted := false
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Write metadata to summary file
	summaryPath := filepath.Join(expDir, cfg.Paths.SummaryFile)
	if err := utils.WriteSummaryFileInit(summaryPath, startTime, repo, opts.Command, expDir); err != nil {
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
	if err := utils.WriteSummaryFileEnd(summaryPath, startTime, endTime, exitCode, interrupted); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

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

// sanitizeBranchName replaces problematic characters in branch names
func sanitizeBranchName(name string) string {
	// Replace problematic characters with underscores
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", " ", "_")
	return r.Replace(name)
}
