// internal/experiment/list.go
package experiment

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bicycle1885/moco/internal/config"
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

// ExperimentInfo holds information about an experiment
type ExperimentInfo struct {
	Path         string    `json:"path"`
	Directory    string    `json:"directory"`
	Timestamp    time.Time `json:"timestamp"`
	Branch       string    `json:"branch"`
	CommitHash   string    `json:"commit_hash"`
	Command      string    `json:"command"`
	ExitStatus   int       `json:"exit_status"`
	Duration     string    `json:"duration"`
	DurationSecs float64   `json:"duration_secs"`
	Interrupted  bool      `json:"interrupted"`
	IsRunning    bool      `json:"is_running"`
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
func findExperiments(baseDir string) ([]ExperimentInfo, error) {
	var experiments []ExperimentInfo

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

		// Parse timestamp from directory name
		timestamp, err := time.Parse("2006-01-02T15:04:05.000", matches[1])
		if err != nil {
			continue // Invalid timestamp format
		}

		// Create experiment info
		exp := ExperimentInfo{
			Path:       filepath.Join(baseDir, name),
			Directory:  name,
			Timestamp:  timestamp,
			Branch:     matches[2],
			CommitHash: matches[3],
			IsRunning:  true, // Assume running by default
		}

		// Parse summary file
		summaryPath := filepath.Join(exp.Path, cfg.Paths.SummaryFile)
		if err := parseSummary(&exp, summaryPath); err != nil {
			// If we can't parse summary, use defaults
			exp.Command = "Unknown"
		}

		experiments = append(experiments, exp)
	}

	return experiments, nil
}

// parseSummary extracts information from a summary.md file
func parseSummary(exp *ExperimentInfo, summaryPath string) error {
	// Open summary file
	file, err := os.Open(summaryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Scan for relevant information
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Extract command
		if strings.Contains(line, "**Command:**") {
			parts := strings.SplitN(line, "`", 3)
			if len(parts) >= 2 {
				exp.Command = parts[1]
			}
		}

		// Check for exit status
		if strings.Contains(line, "**Exit status:**") {
			exp.IsRunning = false
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				status, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					exp.ExitStatus = status
				}
			}
		}

		// Extract duration
		if strings.Contains(line, "**Execution time:**") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				exp.Duration = strings.TrimSpace(parts[1])
				// Try to parse duration in seconds for sorting
				exp.DurationSecs = parseDurationToSeconds(exp.Duration)
			}
		}

		// Check if interrupted
		if strings.Contains(line, "**Terminated by user**") {
			exp.Interrupted = true
		}
	}

	return scanner.Err()
}

// parseDurationToSeconds converts a duration string like "1h 30m 45s" to seconds
func parseDurationToSeconds(duration string) float64 {
	// Pattern to extract hours, minutes, seconds
	reHours := regexp.MustCompile(`(\d+)h`)
	reMinutes := regexp.MustCompile(`(\d+)m`)
	reSeconds := regexp.MustCompile(`(\d+)s`)

	var total float64

	// Extract hours
	if matches := reHours.FindStringSubmatch(duration); len(matches) > 1 {
		if hours, err := strconv.Atoi(matches[1]); err == nil {
			total += float64(hours) * 3600
		}
	}

	// Extract minutes
	if matches := reMinutes.FindStringSubmatch(duration); len(matches) > 1 {
		if minutes, err := strconv.Atoi(matches[1]); err == nil {
			total += float64(minutes) * 60
		}
	}

	// Extract seconds
	if matches := reSeconds.FindStringSubmatch(duration); len(matches) > 1 {
		if seconds, err := strconv.Atoi(matches[1]); err == nil {
			total += float64(seconds)
		}
	}

	return total
}

// filterExperiments applies filters to experiment results
func filterExperiments(experiments []ExperimentInfo, opts ListOptions) ([]ExperimentInfo, error) {
	var filtered []ExperimentInfo

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
		if !sinceTime.IsZero() && exp.Timestamp.Before(sinceTime) {
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
func sortExperiments(experiments []ExperimentInfo, sortBy string, reverse bool) {
	// Define sort function based on criteria
	var sortFunc func(i, j int) bool

	switch sortBy {
	case "branch":
		sortFunc = func(i, j int) bool {
			return experiments[i].Branch < experiments[j].Branch
		}
	case "status":
		sortFunc = func(i, j int) bool {
			// Sort by running/completed, then by exit status
			if experiments[i].IsRunning != experiments[j].IsRunning {
				return experiments[j].IsRunning // Running experiments first
			}
			return experiments[i].ExitStatus < experiments[j].ExitStatus
		}
	case "duration":
		sortFunc = func(i, j int) bool {
			return experiments[i].DurationSecs < experiments[j].DurationSecs
		}
	default: // "date" or any other value defaults to date
		sortFunc = func(i, j int) bool {
			return experiments[i].Timestamp.After(experiments[j].Timestamp)
		}
	}

	// Apply reverse if requested
	if reverse {
		originalFunc := sortFunc
		sortFunc = func(i, j int) bool {
			return !originalFunc(i, j)
		}
	}

	// Sort the slice
	sort.Slice(experiments, sortFunc)
}

// outputTable formats and displays experiments as a table
func outputTable(experiments []ExperimentInfo) error {
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
func outputJSON(experiments []ExperimentInfo) error {
	// Create output structure
	output := struct {
		Experiments []ExperimentInfo `json:"experiments"`
		Count       int              `json:"count"`
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
func outputCSV(experiments []ExperimentInfo) error {
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
		timestamp := exp.Timestamp.Format("2006-01-02 15:04:05")

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
