// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package editor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"tae/internal/config"
)

const (
	MarkerTracked = "[RASTREADOS]"
	MarkerIgnored = "[DENYLIST]"
	Instructions  = "# Edite os caminhos monitorados abaixo. Um por linha.\n# Linhas vazias ou começando com '#' serão ignoradas.\n# CUIDADO: Não altere ou remova as seções [RASTREADOS] e [DENYLIST]."
)

// BuildDraft gera o conteúdo textual inicial que será apresentado no editor.
func BuildDraft(files, ignored []string) []byte {
	var buf bytes.Buffer

	buf.WriteString(Instructions + "\n\n")

	buf.WriteString(MarkerTracked + "\n")
	for _, f := range files {
		if f == "" {
			buf.WriteString(".\n")
		} else {
			buf.WriteString(f + "\n")
		}
	}

	buf.WriteString("\n" + MarkerIgnored + "\n")
	for _, i := range ignored {
		if i == "" {
			buf.WriteString(".\n")
		} else {
			buf.WriteString(i + "\n")
		}
	}

	return buf.Bytes()
}

// RunEditor cria o rascunho no disco, invoca o editor de texto interativo e retorna o conteúdo modificado.
func RunEditor(content []byte, tagName string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("tae_edit_%s_*.ini", tagName))
	if err != nil {
		return nil, fmt.Errorf("falha de I/O ao criar rascunho temporário: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Limpeza garantida após a saída do editor ou em caso de pânico
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("falha ao gravar estado inicial no rascunho: %w", err)
	}
	tmpFile.Close()

	editorBin := config.GetEditor()

	// Prepara a invocação. Passar os ponteiros de I/O padrão é o que permite
	// o editor "sequestrar" a interface do terminal do nosso CLI.
	cmd := exec.Command(editorBin, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("o processo do editor '%s' encerrou com erro: %w", editorBin, err)
	}

	modifiedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("falha de I/O ao ler as modificações do rascunho: %w", err)
	}

	return modifiedContent, nil
}
