// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"path/filepath"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	convertToGit bool
	convertToTae bool
)

var convertCmd = &cobra.Command{
	Use:   "convert <nome da tag>",
	Short: "Converte uma tag entre os escopos Local e Git",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if convertToGit == convertToTae {
			return fmt.Errorf("use --git (-g) OU --tae (-t) para definir o destino da conversão")
		}

		tagName := args[0]

		allTags, err := storage.GetAllTagsWithMeta()
		if err != nil {
			return fmt.Errorf("erro ao acessar banco de dados: %w", err)
		}

		meta, exists := allTags[tagName]
		if !exists {
			return fmt.Errorf("a tag '%s' não existe", tagName)
		}

		filesKeys, ignoredKeys, err := storage.GetTagRawKeys(tagName)
		if err != nil {
			return fmt.Errorf("erro ao ler chaves: %w", err)
		}

		swapFiles := make(map[string]string)
		swapIgnored := make(map[string]string)

		if convertToGit {
			if meta.Type == storage.TagTypeGit {
				return fmt.Errorf("a tag '%s' já pertence ao Git")
			}
			if !isInsideGitRepo() {
				return fmt.Errorf("você precisa estar dentro de um repositório Git para converter esta tag")
			}

			repoID := getGitRepoID()

			for _, k := range filesKeys {
				relPath, err := getGitRelativePath(k)
				if err != nil {
					return fmt.Errorf("o caminho '%s' está fora do repositório. Mova ou remova-o da tag antes de converter", k)
				}
				if k != relPath {
					swapFiles[k] = relPath
				}
			}
			for _, k := range ignoredKeys {
				relPath, err := getGitRelativePath(k)
				if err != nil {
					return fmt.Errorf("o caminho da denylist '%s' está fora do repositório", k)
				}
				if k != relPath {
					swapIgnored[k] = relPath
				}
			}

			meta.Type = storage.TagTypeGit
			meta.RepoID = repoID
			meta.RepoName = getGitRepoName()
			meta.GitRoot = getGitRoot()

		} else {
			if meta.Type == storage.TagTypeLocal {
				return fmt.Errorf("a tag '%s' já é Local")
			}
			if !isInsideGitRepo() || getGitRepoID() != meta.RepoID {
				return fmt.Errorf("você precisa estar dentro do repositório Git original (%s) para reverter esta tag", meta.RepoID)
			}

			gitRoot := getGitRoot()

			for _, k := range filesKeys {
				absPath := filepath.ToSlash(filepath.Join(gitRoot, k))
				swapFiles[k] = absPath
			}
			for _, k := range ignoredKeys {
				absPath := filepath.ToSlash(filepath.Join(gitRoot, k))
				swapIgnored[k] = absPath
			}

			meta.Type = storage.TagTypeLocal
			meta.RepoID = ""
			meta.RepoName = ""
			meta.GitRoot = ""
		}

		if err := storage.UpdateTagScope(tagName, meta, swapFiles, swapIgnored); err != nil {
			return fmt.Errorf("operação abortada. O banco não foi modificado. Erro: %w", err)
		}

		if convertToGit {
			fmt.Printf("Sucesso: A tag '%s' foi convertida para escopo Git.\n", tagName)
		} else {
			fmt.Printf("Sucesso: A tag '%s' foi convertida para escopo Local (Tae).\n", tagName)
		}
		return nil
	},
}

func init() {
	convertCmd.Flags().BoolVarP(&convertToGit, "git", "g", false, "Converte uma tag Local para Git")
	convertCmd.Flags().BoolVarP(&convertToTae, "tae", "t", false, "Converte uma tag Git para Local (Tae)")
	rootCmd.AddCommand(convertCmd)
}
