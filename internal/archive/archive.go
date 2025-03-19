package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/utils"
	"github.com/charmbracelet/log"
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

// Run archives experiments
func Run(runs []string, opts Options) error {
	// Get config
	cfg := config.GetConfig()

	// Validate format
	if opts.Format == "" {
		opts.Format = cfg.Archive.Format
	}
	if opts.Format != "tar.gz" && opts.Format != "zip" {
		return fmt.Errorf("unsupported archive format: %s", opts.Format)
	}

	// Parse olderThan if provided
	var cutoff time.Time
	if opts.OlderThan != "" {
		var err error
		cutoff, err = parseCutoff(opts.OlderThan)
		if err != nil {
			return fmt.Errorf("invalid olderThan format: %w", err)
		}
	} else {
		// Set the cutoff to a timepoint in the future so that all runs are archived
		cutoff = time.Now().AddDate(1, 0, 0)
	}

	// Filter runs to archive
	runInfos := filterRunsToArchive(runs, cutoff, opts.Status)
	if len(runInfos) == 0 {
		return fmt.Errorf("no runs found matching the criteria")
	}

	// Show what would be archived
	log.Infof("Found %d run(s) to archive:", len(runInfos))
	for _, runInfo := range runInfos {
		var status string
		if runInfo.ExitStatus == 0 {
			status = "Success"
		} else {
			status = "Failure"
		}
		log.Infof("  • %s - %s", runInfo.Directory, status)
	}

	if opts.DryRun {
		log.Info("Dry run completed, no files were archived")
		return nil
	}

	// Confirm with user
	if !confirmArchive() {
		log.Info("Archive operation cancelled")
		return nil
	}

	// Ensure destination directory
	destDir := opts.Destination
	if destDir == "" {
		destDir = "archives"
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Archive each run
	for _, runInfo := range runInfos {
		runDir := runInfo.Directory
		dirName := filepath.Base(filepath.Clean(runDir))
		archivePath := filepath.Join(destDir, dirName+"."+opts.Format)
		log.Infof("Archiving %s to %s", runDir, archivePath)
		if err := archiveDirectory(runDir, archivePath, opts.Format); err != nil {
			return fmt.Errorf("failed to archive %s: %w", runDir, err)
		}

		// Delete original if requested
		if opts.Delete {
			log.Infof("Deleting original directory %s", runDir)
			if err := os.RemoveAll(runDir); err != nil {
				return fmt.Errorf("failed to delete %s: %w", runDir, err)
			}
		}
	}

	log.Infof("Successfully archived %d run(s)", len(runInfos))

	return nil
}

func filterRunsToArchive(runDirs []string, cutoff time.Time, status string) []utils.RunInfo {
	var results []utils.RunInfo

	// Get configuration
	cfg := config.GetConfig()

	// Check each run directory
	for _, runDir := range runDirs {
		// Check if it's a directory
		exists, err := directoryExists(runDir)
		if err != nil {
			log.Warnf("Failed to check directory: %v", err)
			continue
		}
		if !exists {
			log.Warnf("Directory not found: %s", runDir)
			continue
		}

		// Parse timestamp from directory name
		dirName := filepath.Base(filepath.Clean(runDir))
		timestamp, err := time.Parse("2006-01-02T15:04:05.000", dirName[:23])
		if err != nil {
			log.Warnf("Failed to parse timestamp: %v", err)
			continue // Invalid timestamp format
		}

		// Apply timestamp filter; skip if it's newer than cutoff
		if !timestamp.Before(cutoff) {
			continue
		}

		// Parse summary file to check if it's finished and the exit status
		summaryPath := filepath.Join(runDir, cfg.Paths.SummaryFile)
		runInfo, err := utils.ParseRunInfo(summaryPath)
		if err != nil {
			log.Warnf("Failed to parse summary file: %v", err)
		}

		// Apply status filter
		if runInfo.IsRunning {
			continue
		}
		if status != "" && status != "all" {
			if status == "success" && runInfo.ExitStatus != 0 {
				continue
			}
			if status == "failure" && runInfo.ExitStatus == 0 {
				continue
			}
		}

		results = append(results, runInfo)
	}

	return results
}

func directoryExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// parseCutoff parses a cutoff string like "30d" to a time.Time
func parseCutoff(cutoff string) (time.Time, error) {
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
/*
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
*/
