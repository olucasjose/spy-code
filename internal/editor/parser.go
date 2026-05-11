// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package editor

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// ParseDraft lê o texto modificado pelo usuário e reconstrói as fatias de arquivos rastreados e denylist.
func ParseDraft(content []byte) (files []string, ignored []string, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignora linhas vazias
		if line == "" {
			continue
		}

		// Ignora comentários no formato INI padrão (# ou ;)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// 1. Detecta mudança de contexto via marcadores (Match Exato)
		if line == MarkerTracked {
			currentSection = "tracked"
			continue
		} else if line == MarkerIgnored {
			currentSection = "ignored"
			continue
		}

		// 2. Heurística de Proteção: Detecta typos nos marcadores que escaparam do match exato
		upperLine := strings.ToUpper(line)
		isReservedWord := strings.Contains(upperLine, "RASTREADOS") || strings.Contains(upperLine, "DENYLIST")
		hasBrackets := strings.Contains(line, "[") || strings.Contains(line, "]")

		if isReservedWord && hasBrackets {
			return nil, nil, fmt.Errorf("marcador malformado detectado: '%s'. Use exatamente %s ou %s", line, MarkerTracked, MarkerIgnored)
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			return nil, nil, fmt.Errorf("seção não reconhecida: '%s'. Apenas %s e %s são permitidas", line, MarkerTracked, MarkerIgnored)
		}

		// 3. Sanidade: Se leu um caminho sem estar sob um marcador
		if currentSection == "" {
			return nil, nil, fmt.Errorf("o caminho '%s' foi encontrado fora de uma seção válida. Certifique-se de manter os marcadores %s e %s intactos", line, MarkerTracked, MarkerIgnored)
		}

		// Normalização preventiva de separadores para consistência com o banco
		line = strings.ReplaceAll(line, "\\", "/")

		if currentSection == "tracked" {
			files = append(files, line)
		} else if currentSection == "ignored" {
			ignored = append(ignored, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("erro inesperado durante a leitura em memória do rascunho: %w", err)
	}

	return files, ignored, nil
}
