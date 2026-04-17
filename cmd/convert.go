// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
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
	Run: func(cmd *cobra.Command, args []string) {
		if convertToGit == convertToTae {
			fmt.Fprintln(os.Stderr, "Erro: Use --git (-g) OU --tae (-t) para definir o destino da conversão.")
			os.Exit(1)
		}

		tagName := args[0]

		allTags, err := storage.GetAllTagsWithMeta()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao acessar banco de dados: %v\n", err)
			os.Exit(1)
		}
		
		meta, exists := allTags[tagName]
		if !exists {
			fmt.Fprintf(os.Stderr, "Erro: a tag '%s' não existe.\n", tagName)
			os.Exit(1)
		}

		filesKeys, ignoredKeys, err := storage.GetTagRawKeys(tagName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao ler chaves: %v\n", err)
			os.Exit(1)
		}

		swapFiles := make(map[string]string)
		swapIgnored := make(map[string]string)

		if convertToGit {
			if meta.Type == storage.TagTypeGit {
				fmt.Fprintf(os.Stderr, "Erro: a tag '%s' já pertence ao Git.\n", tagName)
				os.Exit(1)
			}
			if !isInsideGitRepo() {
				fmt.Fprintln(os.Stderr, "Erro: você precisa estar dentro de um repositório Git para converter esta tag.")
				os.Exit(1)
			}

			repoID := getGitRepoID()

			for _, k := range filesKeys {
				relPath, err := getGitRelativePath(k)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Erro: o caminho '%s' está fora do repositório. Mova ou remova-o da tag antes de converter.\n", k)
					os.Exit(1)
				}
				if k != relPath {
					swapFiles[k] = relPath
				}
			}
			for _, k := range ignoredKeys {
				relPath, err := getGitRelativePath(k)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Erro: o caminho da denylist '%s' está fora do repositório.\n", k)
					os.Exit(1)
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
				fmt.Fprintf(os.Stderr, "Erro: a tag '%s' já é Local.\n", tagName)
				os.Exit(1)
			}
			if !isInsideGitRepo() || getGitRepoID() != meta.RepoID {
				fmt.Fprintf(os.Stderr, "Erro: você precisa estar dentro do repositório Git original (%s) para reverter esta tag.\n", meta.RepoID)
				os.Exit(1)
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
			fmt.Fprintf(os.Stderr, "Operação abortada. O banco não foi modificado.\nErro: %v\n", err)
			os.Exit(1)
		}

		if convertToGit {
			fmt.Printf("Sucesso: A tag '%s' foi convertida para escopo Git.\n", tagName)
		} else {
			fmt.Printf("Sucesso: A tag '%s' foi convertida para escopo Local (Tae).\n", tagName)
		}
	},
}

func init() {
	convertCmd.Flags().BoolVarP(&convertToGit, "git", "g", false, "Converte uma tag Local para Git")
	convertCmd.Flags().BoolVarP(&convertToTae, "tae", "t", false, "Converte uma tag Git para Local (Tae)")
	rootCmd.AddCommand(convertCmd)
}
