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
	gitInfoCountIgnored    bool
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

		if !gitInfoType && !gitInfoHash && !gitInfoCountIgnored && !gitInfoCountExportable && !gitInfoCountTotal {
			rawFiles, err := vcs.ListTree(target)
			if err != nil {
				return fmt.Errorf("erro ao ler árvore do Git para o alvo %s: %w", target, err)
			}

			repoID := vcs.GetRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				ignoredMap = make(map[string]bool)
			}

			activeTargets, ignoredFilesCount := filter.GetActiveIgnoredTargets(rawFiles, ignoredMap)
			exportableCount := len(rawFiles) - ignoredFilesCount

			fmt.Printf("Nome: %s\n", target)
			fmt.Printf("Tipo: %s\n", targetType)
			fmt.Printf("Hash: %s\n", hash)
			fmt.Printf("Total na Tree: %d\n", len(rawFiles))
			fmt.Printf("Alvos Ignorados: %d\n", len(activeTargets))
			fmt.Printf("Arquivos Exportáveis: %d\n", exportableCount)

			return nil
		}

		if gitInfoType {
			fmt.Println(targetType)
		}

		if gitInfoHash {
			fmt.Println(hash)
		}

		if gitInfoCountIgnored || gitInfoCountTotal || gitInfoCountExportable {
			rawFiles, err := vcs.ListTree(target)
			if err != nil {
				return fmt.Errorf("erro ao ler árvore do Git para o alvo %s: %w", target, err)
			}

			if gitInfoCountTotal {
				fmt.Println(len(rawFiles))
			}

			if gitInfoCountIgnored || gitInfoCountExportable {
				repoID := vcs.GetRepoID()
				ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
				if err != nil {
					ignoredMap = make(map[string]bool)
				}

				activeTargets, ignoredFilesCount := filter.GetActiveIgnoredTargets(rawFiles, ignoredMap)

				if gitInfoCountIgnored {
					fmt.Println(len(activeTargets))
				}

				if gitInfoCountExportable {
					fmt.Println(len(rawFiles) - ignoredFilesCount)
				}
			}
		}

		return nil
	},
}

func init() {
	gitInfoCmd.Flags().BoolVar(&gitInfoType, "type", false, "Exibe apenas o tipo do alvo (Commit, Tag, Branch)")
	gitInfoCmd.Flags().BoolVar(&gitInfoHash, "hash", false, "Exibe apenas o hash completo associado ao alvo")
	gitInfoCmd.Flags().BoolVar(&gitInfoCountIgnored, "count-ignored", false, "Exibe apenas a quantidade de alvos que incidiram no alvo")
	gitInfoCmd.Flags().BoolVar(&gitInfoCountExportable, "count-exportable", false, "Exibe apenas a quantidade de arquivos que serão efetivamente exportados")
	gitInfoCmd.Flags().BoolVar(&gitInfoCountTotal, "count-total", false, "Exibe apenas a quantidade total de arquivos na árvore do Git")
	gitCmd.AddCommand(gitInfoCmd)
}
