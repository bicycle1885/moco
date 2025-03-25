package utils

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func RenderRunInfos(runInfos []RunInfo) string {
	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	headerStyle := cellStyle.Bold(true).Align(lipgloss.Left)
	t := table.New().
		// Enable the header border only
		BorderHeader(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(false).
		BorderRow(false).
		BorderColumn(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			} else if col == 2 {
				return cellStyle.Align(lipgloss.Right)
			} else {
				return cellStyle
			}
		}).
		Headers("Directory", "Status", "Duration", "Command")
	for _, run := range runInfos {
		t.Row(run.Directory, statusString(run), run.Duration(), run.Command)
	}
	return t.Render()
}

func statusString(run RunInfo) string {
	if run.IsRunning {
		return "Running"
	} else if run.ExitStatus == 0 {
		return "Success"
	} else if run.Interrupted {
		return "Interrupted"
	} else {
		return fmt.Sprintf("Failed (exit: %d)", run.ExitStatus)
	}
}
