// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package tui

import (
	"io/fs"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// dirSizesMsg encapsula o mapa de tamanhos computados em background para envio à malha do TUI.
type dirSizesMsg map[string]int64

// calculateDirSizeCmd executa a varredura bloqueante no disco em uma goroutine segregada,
// propagando os pesos progressivamente em direção à raiz solicitada.
func calculateDirSizeCmd(targetAbsPath string) tea.Cmd {
	return func() tea.Msg {
		sizes := make(map[string]int64)

		_ = filepath.WalkDir(targetAbsPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			fileSize := info.Size()
			dir := filepath.Dir(path)

			// Propagação Bottom-Up: soma o peso em todos os subdiretórios pais até atingir o target inicial
			for {
				sizes[dir] += fileSize
				if dir == targetAbsPath || dir == "." || dir == "/" || dir == filepath.VolumeName(dir)+"\\" {
					break
				}
				dir = filepath.Dir(dir)
			}

			return nil
		})

		return dirSizesMsg(sizes)
	}
}
