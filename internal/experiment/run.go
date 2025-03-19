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
	"github.com/charmbracelet/log"
)

// RunOptions contains options for running an experiment
type RunOptions struct {
	BaseDir       string
	Force         bool
	NoPushd       bool
	CleanupOnFail bool
}

// Run executes a command with experiment tracking
func Run(commands []string, opts RunOptions) error {
	// Get config
	cfg := config.GetConfig()

	// Check git repository status
	repo, err := git.GetRepoStatus()
	if err != nil {
		return fmt.Errorf("git repository error: %w", err)
	}

	// Validate git status
	if repo.IsDirty && !opts.Force && cfg.Git.RequireClean {
		return fmt.Errorf("git repository has uncommitted changes, use --force to run anyway")
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
	dirName := fmt.Sprintf("%s_%s_%s", startTime.Format("2006-01-02T15:04:05.000"), git.SanitizeBranchName(repo.Branch), repo.ShortHash)
	expDir := filepath.Join(baseDir, dirName)

	log.Infof("Creating experiment directory: %s", expDir)
	if err := os.Mkdir(expDir, 0755); err != nil {
		return fmt.Errorf("failed to create experiment directory: %w", err)
	}

	// Set up signal handling for clean termination
	interrupted := false
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Write metadata to summary file
	summaryPath := filepath.Join(expDir, cfg.Paths.SummaryFile)
	if err := utils.WriteSummaryFileInit(summaryPath, startTime, repo, commands); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	// Set up output files
	stdoutPath := filepath.Join(expDir, cfg.Paths.StdoutFile)
	stderrPath := filepath.Join(expDir, cfg.Paths.StderrFile)

	// Execute command
	cmd := exec.Command(commands[0], commands[1:]...)

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
	log.Infof("Starting command: %s", strings.Join(commands, " "))
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
		log.Warnf("Received signal: %v", sig)

		if cmd.Process != nil {
			// Check if the process is still running before sending the signal
			// by sending signal 0, which doesn't actually send a signal but checks if process exists
			err := cmd.Process.Signal(syscall.Signal(0))
			if err == nil {
				// Process is still running, send the termination signal
				if err := cmd.Process.Signal(sig); err != nil {
					log.Errorf("Failed to send signal to process: %v", err)
				}
			} else {
				log.Debugf("Process already terminated, no signal sent")
			}
		}

		<-doneChan
		exitCode = 130 // Convention for interrupted commands
	}

	if exitCode == 0 {
		log.Info("Command finished successfully")
	} else {

		log.Infof("Command finished with exit code %d", exitCode)
	}

	// Record execution results
	endTime := time.Now()
	if err := utils.WriteSummaryFileEnd(summaryPath, startTime, endTime, exitCode, interrupted); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	// Handle cleanup on failure
	if exitCode != 0 && opts.CleanupOnFail {
		log.Infof("Cleaning up directory: %s", expDir)
		os.RemoveAll(expDir)
	}

	if exitCode != 0 {
		return fmt.Errorf("command failed with exit code %d", exitCode)
	}

	return nil
}
