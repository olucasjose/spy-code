// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppSettings mapeia as configurações globais do usuário no disco
type AppSettings struct {
	Editor          string   `json:"editor"`
	StatsIgnoreDirs []string `json:"stats_ignore_dirs"`
}

func getSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("falha ao localizar diretório home: %w", err)
	}

	dir := filepath.Join(home, ".tae")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório base de configuração: %w", err)
	}

	return filepath.Join(dir, "config.json"), nil
}

// LoadSettings lê as configurações do disco. Cria um arquivo padrão se não existir.
func LoadSettings() (*AppSettings, error) {
	path, err := getSettingsPath()
	if err != nil {
		return nil, err
	}

	cfg := &AppSettings{}

	// Verifica se o arquivo existe; se não, faz o bootstrap
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg.Editor = "" // Mantemos vazio para permitir fallback dinâmico ($VISUAL/$EDITOR) se o usuário não preencher
		cfg.StatsIgnoreDirs = []string{".git", "node_modules"}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("erro ao gerar json padrão de configurações: %w", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, fmt.Errorf("erro ao salvar configurações no disco: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro de I/O ao ler arquivo de configurações: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("JSON inválido no arquivo ~/.tae/config.json: %w", err)
	}

	// Retrocompatibilidade para arquivos antigos que não possuem o campo StatsIgnoreDirs
	if cfg.StatsIgnoreDirs == nil {
		cfg.StatsIgnoreDirs = []string{".git", "node_modules"}
		
		// Opcional: Re-escrever o arquivo com os defaults aplicados
		updatedData, _ := json.MarshalIndent(cfg, "", "  ")
		os.WriteFile(path, updatedData, 0644)
	}

	return cfg, nil
}

// GetEditor resolve a hierarquia do editor que interceptará o terminal.
// Ordem de precedência: ~/.tae/config.json -> $VISUAL -> $EDITOR -> nano
func GetEditor() string {
	cfg, err := LoadSettings()
	if err == nil && strings.TrimSpace(cfg.Editor) != "" {
		return strings.TrimSpace(cfg.Editor)
	}

	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	return "nano"
}
