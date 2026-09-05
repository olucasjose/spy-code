// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"tae/internal/config"
	"tae/internal/fs"
	"tae/internal/stats"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	infoGitRoot         bool
	infoCountItems      bool
	infoCountDenylist   bool
	infoCountExportable bool
	infoStats           bool
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

		if !infoGitRoot && !infoCountItems && !infoCountDenylist && !infoCountExportable && !infoStats {
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
				if err := fs.ValidateGitRoot(meta); err != nil {
					fmt.Printf("\n\033[33m[!] AVISO: %v\033[0m\n\n", err)
				}
				fmt.Printf("Git Root: %s\n", meta.GitRoot)
			} else {
				fmt.Printf("Tipo: Global\n")
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

		if infoStats {
			var baseRoot string
			if meta.Type == storage.TagTypeGit {
				baseRoot = meta.GitRoot
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("falha ao determinar diretório de trabalho atual: %w", err)
				}
				baseRoot = cwd
			}

			cfg, _ := config.LoadSettings()
			var statsIgnoreDirs []string
			if cfg != nil {
				statsIgnoreDirs = cfg.StatsIgnoreDirs
			}

			data := stats.Calculate(baseRoot, statsIgnoreDirs)

			fmt.Println("\nEstatísticas da Tag")
			fmt.Println("\nBinários")
			if len(data.Binary) == 0 {
				fmt.Println("  Nenhum arquivo binário")
			} else {
				for _, ec := range sortMap(data.Binary) {
					fmt.Printf("  %s - %d\n", ec.Ext, ec.Count)
				}
			}

			fmt.Println("\nNão Binários")
			if len(data.NonBinary) == 0 {
				fmt.Println("  Nenhum arquivo não binário")
			} else {
				for _, ec := range sortMap(data.NonBinary) {
					fmt.Printf("  %s - %d\n", ec.Ext, ec.Count)
				}
			}
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

type extCount struct {
	Ext   string
	Count int
}

func sortMap(m map[string]int) []extCount {
	var lst []extCount
	for k, v := range m {
		lst = append(lst, extCount{k, v})
	}
	sort.Slice(lst, func(i, j int) bool {
		if lst[i].Count == lst[j].Count {
			return lst[i].Ext < lst[j].Ext
		}
		return lst[i].Count > lst[j].Count
	})
	return lst
}

func init() {
	infoCmd.Flags().BoolVar(&infoGitRoot, "gitroot", false, "Exibe apenas o caminho raiz do repositório Git associado")
	infoCmd.Flags().BoolVar(&infoCountItems, "count-itens", false, "Exibe apenas a quantidade de itens rastreados")
	infoCmd.Flags().BoolVar(&infoCountDenylist, "count-denylist", false, "Exibe apenas a quantidade de itens na denylist")
	infoCmd.Flags().BoolVar(&infoCountExportable, "count-exportable", false, "Exibe apenas a quantidade total de arquivos que serão efetivamente exportados")
	infoCmd.Flags().BoolVar(&infoStats, "stats", false, "Exibe estatísticas detalhadas de tipos de arquivos associados ao repositório")
	rootCmd.AddCommand(infoCmd)
}
