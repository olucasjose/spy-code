package tui

import (
	"io/fs"
	"path/filepath"
	"strings"

	"tae/internal/filter"
	tea "github.com/charmbracelet/bubbletea"
)

type StatsData struct {
	Binary    map[string]int
	NonBinary map[string]int
}

type statsMsg struct {
	Data *StatsData
}

func calculateStatsCmd(baseRoot string, ignoreDirs []string) tea.Cmd {
	return func() tea.Msg {
		ignoreMap := make(map[string]bool)
		for _, dir := range ignoreDirs {
			ignoreMap[dir] = true
		}
		data := &StatsData{
			Binary:    make(map[string]int),
			NonBinary: make(map[string]int),
		}

		_ = filepath.WalkDir(baseRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Ignorar a pasta .git para não pesar nas estatísticas desnecessariamente, 
            // ou talvez não, o usuário disse "somando tudo, não apenas tracked, ignore, etc". 
            // Mas .git pode ser muito grande, vou incluir de qualquer jeito, ou ignorar .git?
            // É melhor ignorar .git pois não é arquivo do usuário.
			if d.IsDir() {
				if ignoreMap[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			isBin, err := filter.IsBinaryFile(path)
			if err != nil {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext == "" {
				ext = "sem extensão"
			}

			if isBin {
				data.Binary[ext]++
			} else {
				data.NonBinary[ext]++
			}

			return nil
		})

		return statsMsg{Data: data}
	}
}
