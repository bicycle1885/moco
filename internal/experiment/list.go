package experiment

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/bicycle1885/moco/internal/utils"
	"golang.org/x/exp/slices"
)

// ListOptions defines filtering and display options
type ListOptions struct {
	Format  string // Output format (table, json, csv)
	SortBy  string // Sort field (date, branch, status, duration)
	Reverse bool   // Reverse sort order
	Branch  string // Filter by branch name
	Status  string // Filter by status (success, failure, running)
	Since   string // Filter by date (e.g., "7d" for last 7 days)
	Command string // Filter by command pattern
	Limit   int    // Limit number of results
}

// List displays and filters experiments
func List(opts ListOptions) error {
	// Get config
	cfg := config.GetConfig()

	// Find all experiments
	experiments, err := findExperiments(cfg.Paths.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to find experiments: %w", err)
	}

	if len(experiments) == 0 {
		fmt.Println("No experiments found.")
		return nil
	}

	// Apply filters
	filtered, err := filterExperiments(experiments, opts)
	if err != nil {
		return fmt.Errorf("failed to apply filters: %w", err)
	}

	if len(filtered) == 0 {
		fmt.Println("No experiments match the specified criteria.")
		return nil
	}

	// Sort experiments
	sortExperiments(filtered, opts.SortBy, opts.Reverse)

	// Apply limit if specified
	if opts.Limit > 0 && opts.Limit < len(filtered) {
		filtered = filtered[:opts.Limit]
	}

	// Output in the requested format
	switch opts.Format {
	case "json":
		return outputJSON(filtered)
	case "csv":
		return outputCSV(filtered)
	default: // table
		return outputTable(filtered)
	}
}

// findExperiments scans the base directory for experiment directories
func findExperiments(baseDir string) ([]utils.RunInfo, error) {
	var experiments []utils.RunInfo

	// Ensure base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return experiments, nil // Return empty slice if directory doesn't exist
	}

	// Pattern for experiment directories
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3})_(.+)_([a-f0-9]{7})$`)

	// Read all entries in base directory
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	// Get configuration
	cfg := config.GetConfig()

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

		// Parse summary file
		summaryPath := filepath.Join(baseDir, name, cfg.Paths.SummaryFile)
		runInfo, err := utils.ParseRunInfo(summaryPath)
		if err != nil {
			// TODO: Log error and continue
			return nil, fmt.Errorf("failed to parse summary file: %w", err)
		}

		experiments = append(experiments, runInfo)
	}

	return experiments, nil
}

// filterExperiments applies filters to experiment results
func filterExperiments(experiments []utils.RunInfo, opts ListOptions) ([]utils.RunInfo, error) {
	var filtered []utils.RunInfo

	// Parse 'since' filter if provided
	var sinceTime time.Time
	if opts.Since != "" {
		duration, err := parseDuration(opts.Since)
		if err != nil {
			return nil, fmt.Errorf("invalid 'since' format: %w", err)
		}
		sinceTime = time.Now().Add(-duration)
	}

	// Compile command regex if provided
	var commandRegex *regexp.Regexp
	if opts.Command != "" {
		var err error
		commandRegex, err = regexp.Compile(opts.Command)
		if err != nil {
			return nil, fmt.Errorf("invalid command pattern: %w", err)
		}
	}

	// Filter each experiment
	for _, exp := range experiments {
		// Filter by branch
		if opts.Branch != "" && !strings.Contains(exp.Branch, opts.Branch) {
			continue
		}

		// Filter by status
		if opts.Status != "" {
			if opts.Status == "success" && (exp.IsRunning || exp.ExitStatus != 0) {
				continue
			}
			if opts.Status == "failure" && (exp.IsRunning || exp.ExitStatus == 0) {
				continue
			}
			if opts.Status == "running" && !exp.IsRunning {
				continue
			}
		}

		// Filter by date
		if !sinceTime.IsZero() && exp.StartTime.Before(sinceTime) {
			continue
		}

		// Filter by command
		if commandRegex != nil && !commandRegex.MatchString(exp.Command) {
			continue
		}

		filtered = append(filtered, exp)
	}

	return filtered, nil
}

// parseDuration parses a duration string like "7d" or "24h"
func parseDuration(s string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)([dhm])$`)
	matches := re.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid duration format: %s (expected 7d, 24h, etc.)", s)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", matches[1])
	}

	var multiplier time.Duration
	switch matches[2] {
	case "d":
		multiplier = 24 * time.Hour
	case "h":
		multiplier = time.Hour
	case "m":
		multiplier = time.Minute
	default:
		return 0, fmt.Errorf("invalid duration unit: %s", matches[2])
	}

	return time.Duration(value) * multiplier, nil
}

// sortExperiments sorts experiments based on criteria
func sortExperiments(experiments []utils.RunInfo, sortBy string, reverse bool) {
	// Define sort function based on criteria
	var sortFunc func(i, j utils.RunInfo) int

	switch sortBy {
	case "branch":
		sortFunc = func(a, b utils.RunInfo) int {
			return strings.Compare(a.Branch, b.Branch)
		}
	case "status":
		sortFunc = func(a, b utils.RunInfo) int {
			// Sort by running/completed, then by exit status
			if a.IsRunning {
				if b.IsRunning {
					return 0
				}
				return -1
			} else if b.IsRunning {
				return 1
			}
			return compareInt(a.ExitStatus, b.ExitStatus)
		}
	case "duration":
		sortFunc = func(a, b utils.RunInfo) int {
			return compareDuration(a.EndTime.Sub(a.StartTime), b.EndTime.Sub(b.StartTime))
		}
	default: // "date" or any other value defaults to date
		sortFunc = func(a, b utils.RunInfo) int {
			return compareTime(a.StartTime, b.StartTime)
		}
	}

	// Apply reverse if requested
	if reverse {
		originalFunc := sortFunc
		sortFunc = func(a, b utils.RunInfo) int {
			return -originalFunc(a, b)
		}
	}

	// Sort the slice
	slices.SortStableFunc(experiments, sortFunc)
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareDuration(a, b time.Duration) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

// outputTable formats and displays experiments as a table
func outputTable(experiments []utils.RunInfo) error {
	// Create a tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Write header
	fmt.Fprintln(w, "DIRECTORY\tBRANCH\tSTATUS\tDURATION\tCOMMAND")

	// Write each experiment
	for _, exp := range experiments {
		// Format status
		status := "Running"
		if !exp.IsRunning {
			if exp.ExitStatus == 0 {
				status = "Success"
			} else {
				status = fmt.Sprintf("Failed (%d)", exp.ExitStatus)
				if exp.Interrupted {
					status = "Interrupted"
				}
			}
		}

		// Format duration
		duration := exp.Duration
		if duration == "" {
			duration = "-"
		}

		// Write the row
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			exp.Directory,
			exp.Branch,
			status,
			duration,
			exp.Command,
		)
	}

	return nil
}

// outputJSON formats and displays experiments as JSON
func outputJSON(experiments []utils.RunInfo) error {
	// Create output structure
	output := struct {
		Experiments []utils.RunInfo `json:"experiments"`
		Count       int             `json:"count"`
	}{
		Experiments: experiments,
		Count:       len(experiments),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to stdout
	fmt.Println(string(data))
	return nil
}

// outputCSV formats and displays experiments as CSV
func outputCSV(experiments []utils.RunInfo) error {
	// Create a CSV writer
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	header := []string{"Directory", "Timestamp", "Branch", "CommitHash", "Status", "Duration", "Command"}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write each experiment
	for _, exp := range experiments {
		// Format status
		status := "Running"
		if !exp.IsRunning {
			if exp.ExitStatus == 0 {
				status = "Success"
			} else {
				status = fmt.Sprintf("Failed (%d)", exp.ExitStatus)
				if exp.Interrupted {
					status = "Interrupted"
				}
			}
		}

		// Format timestamp
		timestamp := exp.StartTime.Format("2006-01-02 15:04:05")

		// Format duration
		duration := exp.Duration
		if duration == "" {
			duration = "-"
		}

		// Create the record
		record := []string{
			exp.Directory,
			timestamp,
			exp.Branch,
			exp.CommitHash,
			status,
			duration,
			exp.Command,
		}

		// Write the record
		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}
