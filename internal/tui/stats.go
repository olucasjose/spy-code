package tui

import (
	"tae/internal/stats"
	tea "github.com/charmbracelet/bubbletea"
)

type statsMsg struct {
	Data *stats.Data
}

func calculateStatsCmd(baseRoot string, ignoreDirs []string) tea.Cmd {
	return func() tea.Msg {
		data := stats.Calculate(baseRoot, ignoreDirs)
		return statsMsg{Data: data}
	}
}
