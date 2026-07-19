// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"tae/internal/fs"
	"tae/internal/storage"
	"tae/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiManagerCmd = &cobra.Command{
	Use:   "tui-manager <tag>",
	Short: "Abre a interface visual iterativa para gerenciar arquivos rastreados e denylist",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tagName := args[0]

		meta, err := storage.GetTagMeta(tagName)
		if err != nil {
			if errors.Is(err, storage.ErrTagNotFound) {
				return fmt.Errorf("a tag '%s' não existe", tagName)
			}
			return fmt.Errorf("erro ao consultar banco de dados: %w", err)
		}

		var baseRoot string
		if meta.Type == storage.TagTypeGit {
			if err := fs.ValidateGitRoot(meta); err != nil {
				return err
			}
			baseRoot = meta.GitRoot
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("falha ao determinar diretório de trabalho atual: %w", err)
			}
			baseRoot = cwd
		}

		rawFiles, rawIgnored, err := storage.GetTagRawKeys(tagName)
		if err != nil {
			return fmt.Errorf("erro ao extrair estado atual do banco: %w", err)
		}

		trackedMap := make(map[string]bool)
		for _, p := range rawFiles {
			trackedMap[filepath.ToSlash(p)] = true
		}

		ignoredMap := make(map[string]bool)
		for _, p := range rawIgnored {
			ignoredMap[filepath.ToSlash(p)] = true
		}

		model := tui.InitialModel(tagName, baseRoot, trackedMap, ignoredMap)
		
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("a malha de execução do TUI foi abortada: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiManagerCmd)
}
