// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"errors"
	"fmt"

	"tae/internal/fs"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	infoGitRoot         bool
	infoCountItems      bool
	infoCountDenylist   bool
	infoCountExportable bool
)

var infoCmd = &cobra.Command{
	Use:   "info [nome da tag]",
	Short: "Exibe informações e estatísticas detalhadas sobre uma tag",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
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
			return fmt.Errorf("erro ao obter metadados da tag: %w", err)
		}

		if !infoGitRoot && !infoCountItems && !infoCountDenylist && !infoCountExportable {
			trackedCount, err := storage.CountTrackedFiles(tagName)
			if err != nil {
				return err
			}

			ignoredCount, err := storage.CountIgnoredFiles(tagName)
			if err != nil {
				return err
			}

			fmt.Printf("Tag: %s\n", tagName)
			if meta.Type == storage.TagTypeGit {
				fmt.Printf("Tipo: Git\n")
				repoName := meta.RepoName
				if repoName == "" {
					repoName = meta.RepoID
				}
				fmt.Printf("Repositório: %s\n", repoName)
				fmt.Printf("Git Root: %s\n", meta.GitRoot)
			} else {
				fmt.Printf("Tipo: Local\n")
			}

			fmt.Printf("Alvos Monitorados: %d\n", trackedCount)
			fmt.Printf("Alvos Ignorados: %d\n", ignoredCount)

			exportableCount, err := countExportableFiles(tagName, meta)
			if err == nil {
				fmt.Printf("Arquivos Exportáveis: %d\n", exportableCount)
			}

			return nil
		}

		if infoGitRoot {
			fmt.Println(meta.GitRoot)
		}

		if infoCountItems {
			trackedCount, err := storage.CountTrackedFiles(tagName)
			if err != nil {
				return err
			}
			fmt.Println(trackedCount)
		}

		if infoCountDenylist {
			ignoredCount, err := storage.CountIgnoredFiles(tagName)
			if err != nil {
				return err
			}
			fmt.Println(ignoredCount)
		}

		if infoCountExportable {
			exportableCount, err := countExportableFiles(tagName, meta)
			if err != nil {
				return err
			}
			fmt.Println(exportableCount)
		}

		return nil
	},
}

func countExportableFiles(tagName string, meta storage.TagMeta) (int, error) {
	files, err := storage.GetFilesByTag(tagName)
	if err != nil {
		return 0, err
	}

	resolvedFiles, err := fs.RestorePathsForDisk(tagName, meta, files)
	if err != nil {
		return 0, err
	}

	ignoredMap, _ := storage.GetIgnoredPaths(tagName)
	restoredIgnored := make(map[string]bool)
	var igPaths []string
	for p := range ignoredMap {
		igPaths = append(igPaths, p)
	}
	if resIgPaths, err := fs.RestorePathsForDisk(tagName, meta, igPaths); err == nil {
		for _, p := range resIgPaths {
			restoredIgnored[p] = true
		}
	}

	expanded := fs.ExpandPathsToFiles(resolvedFiles, restoredIgnored)
	return len(expanded), nil
}

func init() {
	infoCmd.Flags().BoolVar(&infoGitRoot, "gitroot", false, "Exibe apenas o caminho raiz do repositório Git associado")
	infoCmd.Flags().BoolVar(&infoCountItems, "count-itens", false, "Exibe apenas a quantidade de itens rastreados")
	infoCmd.Flags().BoolVar(&infoCountDenylist, "count-denylist", false, "Exibe apenas a quantidade de itens na denylist")
	infoCmd.Flags().BoolVar(&infoCountExportable, "count-exportable", false, "Exibe apenas a quantidade total de arquivos que serão efetivamente exportados")
	rootCmd.AddCommand(infoCmd)
}
