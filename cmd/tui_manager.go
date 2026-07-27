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
	"tae/internal/vcs"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiSizeMode string
var tuiNoBinary bool

var tuiManagerCmd = &cobra.Command{
	Use:   "tui-manager <tag>",
	Short: "Abre a interface visual iterativa para gerenciar arquivos rastreados e denylist",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if tuiSizeMode != "only-files" && tuiSizeMode != "all-files" && tuiSizeMode != "filtered" {
			return fmt.Errorf("valor inválido para --size-mode: '%s'. Valores aceitos: only-files, all-files, filtered", tuiSizeMode)
		}

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

		var gitIgnoredMap map[string]bool
		if meta.Type == storage.TagTypeGit {
			var err error
			gitIgnoredMap, err = vcs.GetIgnoredPaths(baseRoot)
			if err != nil {
				gitIgnoredMap = make(map[string]bool)
			}
		} else {
			gitIgnoredMap = make(map[string]bool)
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

		model := tui.InitialModel(tagName, baseRoot, trackedMap, ignoredMap, tuiSizeMode, gitIgnoredMap, tuiNoBinary)
		
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("a malha de execução do TUI foi abortada: %w", err)
		}

		return nil
	},
}

func init() {
	tuiManagerCmd.Flags().StringVarP(&tuiSizeMode, "size-mode", "s", "all-files", "Modo de cálculo de tamanho (all-files, filtered, only-files)")
	tuiManagerCmd.Flags().BoolVar(&tuiNoBinary, "no-binary", false, "Ocultar arquivos binários da interface e não os contabilizar no peso")
	rootCmd.AddCommand(tuiManagerCmd)
}

