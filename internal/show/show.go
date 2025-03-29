package show

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bicycle1885/moco/internal/config"
	"github.com/charmbracelet/glamour"
)

func Main(run string) error {
	cfg := config.Get()

	var summaryPath string
	stat, err := os.Stat(run)
	if err == nil {
		if stat.IsDir() {
			// Assume run is a directory containing a summary file
			summaryPath = filepath.Join(run, cfg.SummaryFile)
		} else {
			// Assume run is a summary file
			summaryPath = run
		}
	} else {
		return err
	}

	// Read the markdown file
	content, err := os.ReadFile(summaryPath)
	if err != nil {
		return err
	}

	if !cfg.Show.Raw {
		// Render the markdown content
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(100),
		)
		if err != nil {
			return err
		}
		content, err = renderer.RenderBytes(content)
		if err != nil {
			return err
		}
	}

	return pipeToPager(string(content))
}

func pipeToPager(content string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	// Find the path to the less command
	lessPath, err := exec.LookPath(pager)
	if err != nil {
		// Fall back to just printing if less is not available
		fmt.Print(content)
		return nil
	}

	// Set up the less command with appropriate flags
	// -R: Process ANSI color sequences
	// -F: Quit if entire file fits on first screen
	// -X: Don't clear screen on exit
	cmd := exec.Command(lessPath, "-R", "-F", "-X")
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
