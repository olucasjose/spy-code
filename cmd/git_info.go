// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/filter"
	"tae/internal/storage"
	"tae/internal/vcs"

	"github.com/spf13/cobra"
)

var (
	gitInfoType            bool
	gitInfoHash            bool
	gitInfoCountDenylist   bool
	gitInfoCountExportable bool
	gitInfoCountTotal      bool
)

var gitInfoCmd = &cobra.Command{
	Use:   "info <target>",
	Short: "Exibe informações e estatísticas sobre um alvo Git (branch, tag, commit)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		hash, _, err := vcs.GetCommitInfo(target)
		if err != nil {
			return err
		}

		targetType := vcs.GetTargetType(target)

		if !gitInfoType && !gitInfoHash && !gitInfoCountDenylist && !gitInfoCountExportable && !gitInfoCountTotal {
			// Buscar quantidade de arquivos na tree original do Git
			rawFiles, err := vcs.ListTree(target)
			if err != nil {
				return fmt.Errorf("erro ao ler árvore do Git para o alvo %s: %w", target, err)
			}

			// Buscar regras de exclusão (denylist) do repositório
			repoID := vcs.GetRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				ignoredMap = make(map[string]bool)
			}

			// Filtrar arquivos
			var exportableCount int
			for _, f := range rawFiles {
				if !filter.IsPathIgnoredByMap(f, ignoredMap) {
					exportableCount++
				}
			}

			fmt.Printf("Nome: %s\n", target)
			fmt.Printf("Tipo: %s\n", targetType)
			fmt.Printf("Hash: %s\n", hash)
			fmt.Printf("Total na Tree: %d\n", len(rawFiles))
			fmt.Printf("Alvos Ignorados: %d\n", len(ignoredMap))
			fmt.Printf("Arquivos Exportáveis: %d\n", exportableCount)

			return nil
		}

		if gitInfoType {
			fmt.Println(targetType)
		}

		if gitInfoHash {
			fmt.Println(hash)
		}

		if gitInfoCountDenylist || gitInfoCountExportable || gitInfoCountTotal {
			repoID := vcs.GetRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				ignoredMap = make(map[string]bool)
			}

			if gitInfoCountDenylist {
				fmt.Println(len(ignoredMap))
			}

			if gitInfoCountTotal || gitInfoCountExportable {
				rawFiles, err := vcs.ListTree(target)
				if err != nil {
					return fmt.Errorf("erro ao ler árvore do Git para o alvo %s: %w", target, err)
				}

				if gitInfoCountTotal {
					fmt.Println(len(rawFiles))
				}

				if gitInfoCountExportable {
					var exportableCount int
					for _, f := range rawFiles {
						if !filter.IsPathIgnoredByMap(f, ignoredMap) {
							exportableCount++
						}
					}
					fmt.Println(exportableCount)
				}
			}
		}

		return nil
	},
}

func init() {
	gitInfoCmd.Flags().BoolVar(&gitInfoType, "type", false, "Exibe apenas o tipo do alvo (Commit, Tag, Branch)")
	gitInfoCmd.Flags().BoolVar(&gitInfoHash, "hash", false, "Exibe apenas o hash completo associado ao alvo")
	gitInfoCmd.Flags().BoolVar(&gitInfoCountDenylist, "count-denylist", false, "Exibe apenas a quantidade de alvos na denylist")
	gitInfoCmd.Flags().BoolVar(&gitInfoCountExportable, "count-exportable", false, "Exibe apenas a quantidade de arquivos que serão efetivamente exportados")
	gitInfoCmd.Flags().BoolVar(&gitInfoCountTotal, "count-total", false, "Exibe apenas a quantidade total de arquivos na árvore do Git")
	gitCmd.AddCommand(gitInfoCmd)
}
