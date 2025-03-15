package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bicycle1885/moco/internal/config"
)

// Options defines archiving options
type Options struct {
	OlderThan   string // Archive experiments older than duration (e.g., "30d")
	Status      string // Filter by status (success, failure, all)
	Format      string // Archive format (zip, tar.gz)
	Destination string // Archive destination directory
	Delete      bool   // Delete after archiving
	DryRun      bool   // Show what would be archived
}

// ExperimentInfo holds information about an experiment
type ExperimentInfo struct {
	Path       string    // Full path to experiment directory
	Name       string    // Directory name
	Timestamp  time.Time // Experiment start time
	Branch     string    // Git branch
	CommitHash string    // Git commit hash
	ExitStatus int       // Command exit status
	IsFinished bool      // Whether experiment has finished
}

// Run archives experiments
func Run(opts Options) error {
	// Get config
	cfg := config.GetConfig()

	// Validate format
	if opts.Format == "" {
		opts.Format = cfg.Archive.Format
	}
	if opts.Format != "tar.gz" && opts.Format != "zip" {
		return fmt.Errorf("unsupported archive format: %s", opts.Format)
	}

	// Parse olderThan
	cutoff, err := parseCutoff(opts.OlderThan)
	if err != nil {
		return fmt.Errorf("invalid olderThan format: %w", err)
	}

	// Ensure destination directory
	destDir := opts.Destination
	if destDir == "" {
		destDir = "archives"
	}

	if !opts.DryRun {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	// Find experiments to archive
	exps, err := findExperimentsToArchive(cfg.Paths.BaseDir, cutoff, opts.Status)
	if err != nil {
		return fmt.Errorf("failed to find experiments: %w", err)
	}

	if len(exps) == 0 {
		fmt.Println("No experiments found matching the criteria.")
		return nil
	}

	// Show what would be archived
	fmt.Printf("Found %d experiment(s) to archive:\n", len(exps))
	for _, exp := range exps {
		statusStr := "Running"
		if exp.IsFinished {
			if exp.ExitStatus == 0 {
				statusStr = "Success"
			} else {
				statusStr = fmt.Sprintf("Failed (exit: %d)", exp.ExitStatus)
			}
		}
		fmt.Printf("  • %s - %s\n", exp.Name, statusStr)
	}

	if opts.DryRun {
		fmt.Println("\nDry run - no changes made.")
		return nil
	}

	// Confirm with user if not in dry run mode
	if !confirmArchive() {
		fmt.Println("Archive operation cancelled.")
		return nil
	}

	// Archive each experiment
	for _, exp := range exps {
		archivePath := filepath.Join(destDir, exp.Name+"."+opts.Format)
		fmt.Printf("Archiving %s to %s...\n", exp.Name, archivePath)

		if err := archiveDirectory(exp.Path, archivePath, opts.Format); err != nil {
			return fmt.Errorf("failed to archive %s: %w", exp.Name, err)
		}

		// Delete original if requested
		if opts.Delete {
			fmt.Printf("Deleting original directory %s...\n", exp.Path)
			if err := os.RemoveAll(exp.Path); err != nil {
				return fmt.Errorf("failed to delete %s: %w", exp.Path, err)
			}
		}
	}

	// Create or update archive index
	if err := updateArchiveIndex(destDir, exps, opts.Format, opts.Delete); err != nil {
		fmt.Printf("Warning: failed to update archive index: %v\n", err)
	}

	fmt.Printf("Successfully archived %d experiment(s).\n", len(exps))
	return nil
}

// findExperimentsToArchive finds experiments matching criteria
func findExperimentsToArchive(baseDir string, cutoff time.Time, statusFilter string) ([]ExperimentInfo, error) {
	var results []ExperimentInfo

	// Ensure base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return results, nil // Return empty results if directory doesn't exist
	}

	// Pattern for experiment directories
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3})_(.+)_([a-f0-9]{7})$`)

	// Read all entries in base directory
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	// Check each entry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories
		}

		// Check if the name matches our pattern
		name := entry.Name()
		matches := pattern.FindStringSubmatch(name)
		if len(matches) != 4 {
			continue // Not an experiment directory
		}

		// Parse timestamp from directory name
		timestamp, err := time.Parse("2006-01-02T15:04:05.000", matches[1])
		if err != nil {
			continue // Invalid timestamp format
		}

		// Skip if it's newer than cutoff
		if !timestamp.Before(cutoff) {
			continue
		}

		// Create experiment info
		expInfo := ExperimentInfo{
			Path:       filepath.Join(baseDir, name),
			Name:       name,
			Timestamp:  timestamp,
			Branch:     matches[2],
			CommitHash: matches[3],
		}

		// Parse summary file to check if it's finished and the exit status
		summaryPath := filepath.Join(expInfo.Path, "summary.md")
		if err := parseExperimentStatus(&expInfo, summaryPath); err != nil {
			// If we can't parse, skip based on filter
			if statusFilter != "" && statusFilter != "all" {
				continue
			}
		}

		// Apply status filter
		if statusFilter != "" && statusFilter != "all" {
			if statusFilter == "success" && (expInfo.ExitStatus != 0 || !expInfo.IsFinished) {
				continue
			}
			if statusFilter == "failure" && (expInfo.ExitStatus == 0 || !expInfo.IsFinished) {
				continue
			}
			if statusFilter == "running" && expInfo.IsFinished {
				continue
			}
		}

		results = append(results, expInfo)
	}

	return results, nil
}

// parseExperimentStatus extracts status information from a summary.md file
func parseExperimentStatus(expInfo *ExperimentInfo, summaryPath string) error {
	// Default to running
	expInfo.IsFinished = false
	expInfo.ExitStatus = -1

	// Read summary file
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return err
	}

	// Check for exit status line
	content := string(data)
	exitStatusRe := regexp.MustCompile(`\*\*Exit status:\*\*\s*(\d+)`)
	matches := exitStatusRe.FindStringSubmatch(content)

	if len(matches) == 2 {
		// Found exit status
		expInfo.IsFinished = true
		exitStatus, err := strconv.Atoi(matches[1])
		if err == nil {
			expInfo.ExitStatus = exitStatus
		}
	}

	return nil
}

// parseCutoff parses a cutoff string like "30d" to a time.Time
func parseCutoff(cutoff string) (time.Time, error) {
	if cutoff == "" {
		// Get default from config if not provided
		cfg := config.GetConfig()
		cutoff = cfg.Archive.OlderThan
	}

	// Parse the duration string
	re := regexp.MustCompile(`^(\d+)([dhm])$`)
	matches := re.FindStringSubmatch(cutoff)
	if len(matches) != 3 {
		return time.Time{}, fmt.Errorf("invalid duration format: %s (expected 30d, 12h, etc.)", cutoff)
	}

	// Convert value to int
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration value: %s", matches[1])
	}

	// Calculate duration based on unit
	var duration time.Duration
	switch matches[2] {
	case "d":
		duration = time.Duration(value) * 24 * time.Hour
	case "h":
		duration = time.Duration(value) * time.Hour
	case "m":
		duration = time.Duration(value) * time.Minute
	default:
		return time.Time{}, fmt.Errorf("invalid duration unit: %s", matches[2])
	}

	// Calculate cutoff time
	return time.Now().Add(-duration), nil
}

// confirmArchive asks the user to confirm the archive operation
func confirmArchive() bool {
	fmt.Print("Do you want to proceed with archiving? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// archiveDirectory handles the actual archiving process
func archiveDirectory(srcDir, destPath, format string) error {
	switch format {
	case "tar.gz":
		return archiveToTarGz(srcDir, destPath)
	case "zip":
		return archiveToZip(srcDir, destPath)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

// archiveToTarGz creates a tar.gz archive of a directory
func archiveToTarGz(srcDir, destPath string) error {
	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(destFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk through all files in source directory
	baseDir := filepath.Base(srcDir)
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Set header name relative to source directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(baseDir, relPath)

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Skip if directory
		if info.IsDir() {
			return nil
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

// archiveToZip creates a zip archive of a directory
func archiveToZip(srcDir, destPath string) error {
	// Create zip file
	zipFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through all files in source directory
	baseDir := filepath.Base(srcDir)
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories directly
		if info.IsDir() {
			return nil
		}

		// Create zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set header name relative to source directory
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(baseDir, relPath)
		header.Method = zip.Deflate

		// Create file entry in zip
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// updateArchiveIndex creates or updates an index of archived experiments
func updateArchiveIndex(destDir string, exps []ExperimentInfo, format string, deleted bool) error {
	// Create or open index file
	indexPath := filepath.Join(destDir, "archive_index.md")

	// Check if file exists
	var indexContent string

	data, err := os.ReadFile(indexPath)
	if err == nil {
		indexContent = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Open file for writing
	file, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// If no content, create header
	if indexContent == "" {
		indexContent = "# Moco Experiment Archive Index\n\n"
		indexContent += "| Archive File | Original Directory | Timestamp | Branch | Status | Archived On | Original Deleted |\n"
		indexContent += "|-------------|-------------------|-----------|--------|--------|------------|------------------|\n"
	}

	// Add entries for newly archived experiments
	now := time.Now().Format("2006-01-02 15:04:05")

	for _, exp := range exps {
		status := "Running"
		if exp.IsFinished {
			if exp.ExitStatus == 0 {
				status = "Success"
			} else {
				status = fmt.Sprintf("Failed (%d)", exp.ExitStatus)
			}
		}

		// Use proper Go syntax for conditional values
		deletedStr := "No"
		if deleted {
			deletedStr = "Yes"
		}

		// Create a new row for this experiment
		newRow := fmt.Sprintf("| %s.%s | %s | %s | %s | %s | %s | %s |\n",
			exp.Name, format,
			exp.Name,
			exp.Timestamp.Format("2006-01-02 15:04:05"),
			exp.Branch,
			status,
			now,
			deletedStr)

		// Add to content
		indexContent += newRow
	}

	// Write to file
	_, err = file.WriteString(indexContent)
	return err
}
