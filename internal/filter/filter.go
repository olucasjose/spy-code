// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package filter

import (
	"path/filepath"
	"strings"
)

// MatchPattern avalia se o caminho alvo bate com algum padrão fornecido (Glob)
func MatchPattern(target string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		matched, err := filepath.Match(p, target)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// IsPathIgnoredByMap verifica se o caminho exato ou seus diretórios pai estão no mapa de exclusão
func IsPathIgnoredByMap(target string, ignoredMap map[string]bool) bool {
	if ignoredMap[target] {
		return true
	}
	parts := strings.Split(target, "/")
	current := ""
	for i := 0; i < len(parts)-1; i++ {
		if current == "" {
			current = parts[i]
		} else {
			current = current + "/" + parts[i]
		}
		if ignoredMap[current] {
			return true
		}
	}
	return false
}
