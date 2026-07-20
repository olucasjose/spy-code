// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package tui

import (
	"io/fs"
	"path/filepath"
	"strings"

	"tae/internal/filter"

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

// isPathGitIgnored verifica se o caminho relativo ou algum de seus diretórios pai está ignorado no gitignoredMap.
func isPathGitIgnored(relPath string, gitIgnoredMap map[string]bool) bool {
	if len(gitIgnoredMap) == 0 {
		return false
	}
	parts := strings.Split(relPath, "/")
	current := ""
	for i := 0; i < len(parts); i++ {
		if current == "" {
			current = parts[i]
		} else {
			current = current + "/" + parts[i]
		}
		if gitIgnoredMap[current] {
			return true
		}
	}
	return false
}

// calculateAllSizesCmd calcula os tamanhos de todos os subdiretórios a partir de baseRoot em background.
func calculateAllSizesCmd(baseRoot string, calcMode string, ignoredMap map[string]bool, gitIgnoredMap map[string]bool) tea.Cmd {
	return func() tea.Msg {
		sizes := make(map[string]int64)

		_ = filepath.WalkDir(baseRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Calcula o caminho relativo para verificação no gitignore/denylist
			relPath, err := filepath.Rel(baseRoot, path)
			if err != nil {
				return nil
			}
			relPath = filepath.ToSlash(relPath)

			if calcMode == "filtered" {
				// Verifica a denylist específica da tag
				if filter.IsPathIgnoredByMap(relPath, ignoredMap) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				// Verifica o gitignore da tag
				if isPathGitIgnored(relPath, gitIgnoredMap) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			fileSize := info.Size()
			dir := filepath.Dir(path)

			// Propagação Bottom-Up: soma o peso em todos os subdiretórios pais até atingir o baseRoot inicial
			for {
				sizes[dir] += fileSize
				if dir == baseRoot || dir == "." || dir == "/" || dir == filepath.VolumeName(dir)+"\\" {
					break
				}
				dir = filepath.Dir(dir)
			}

			return nil
		})

		return dirSizesMsg(sizes)
	}
}

