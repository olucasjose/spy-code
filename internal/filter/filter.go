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

// GetIgnoredRule retorna a regra exata (do mapa de exclusão) que bloqueou o caminho alvo. Se não houver, retorna string vazia.
func GetIgnoredRule(target string, ignoredMap map[string]bool) string {
	if ignoredMap[target] {
		return target
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
			return current
		}
	}
	return ""
}

// IsPathIgnoredByMap verifica se o caminho exato ou seus diretórios pai estão no mapa de exclusão
func IsPathIgnoredByMap(target string, ignoredMap map[string]bool) bool {
	return GetIgnoredRule(target, ignoredMap) != ""
}

// GetActiveIgnoredTargets processa todos os arquivos, retorna o mapa apenas com as regras que incidiram neles e o total de arquivos bloqueados.
func GetActiveIgnoredTargets(files []string, ignoredMap map[string]bool) (map[string]bool, int) {
	activeTargets := make(map[string]bool)
	ignoredFilesCount := 0

	for _, f := range files {
		rule := GetIgnoredRule(f, ignoredMap)
		if rule != "" {
			activeTargets[rule] = true
			ignoredFilesCount++
		}
	}

	return activeTargets, ignoredFilesCount
}
