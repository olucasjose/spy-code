// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tae/internal/render"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	gitListTree    bool
	gitListDepth   int
	gitListIgnore  string
	gitListIgnored bool
)

var gitListCmd = &cobra.Command{
	Use:   "list [commit]",
	Short: "Lista arquivos de um commit ou a denylist do repositório atual",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Interceptação isolada para listar a denylist do repositório
		if gitListIgnored {
			repoID := getGitRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao ler a denylist do repositório: %v\n", err)
				os.Exit(1)
			}

			if len(ignoredMap) == 0 {
				fmt.Println("A denylist do repositório atual está vazia.")
				return
			}

			fmt.Println("Exclusion Index (Denylist) do repositório atual:")
			for path := range ignoredMap {
				fmt.Printf("  - %s\n", path)
			}
			return
		}

		// Validação Fail-Fast: se não for para ver a denylist, um commit é obrigatório
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Erro: Informe um <commit> para listar ou use a flag --ignored (-i) para ver a denylist.")
			os.Exit(1)
		}

		commit := args[0]
		out, err := exec.Command("git", "ls-tree", "-r", "--name-only", commit).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao consultar Git:\n%s\n", string(out))
			os.Exit(1)
		}

		var files []string
		for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if f != "" {
				files = append(files, f)
			}
		}

		if len(files) == 0 {
			fmt.Println("Nenhum arquivo encontrado neste commit.")
			return
		}

		var ignorePatterns []string
		if gitListIgnore != "" {
			ignorePatterns = strings.Split(gitListIgnore, "|")
		}

		if gitListTree {
			rootNode := render.BuildVisualTree(files, "")
			render.PrintTree(rootNode, "", 0, gitListDepth, ignorePatterns)
		} else {
			for _, f := range files {
				if !renderIsIgnored(f, ignorePatterns) {
					fmt.Printf("  - %s\n", f)
				}
			}
		}
	},
}

func renderIsIgnored(path string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
	}
	return false
}

func init() {
	gitListCmd.Flags().BoolVarP(&gitListTree, "tree", "t", false, "Exibe os caminhos em formato de árvore")
	gitListCmd.Flags().IntVarP(&gitListDepth, "level", "L", 0, "Profundidade máxima da árvore (0 = infinito)")
	gitListCmd.Flags().StringVarP(&gitListIgnore, "ignore", "I", "", "Padrões para ignorar na exibição (ex: \"*.go\")")
	gitListCmd.Flags().BoolVarP(&gitListIgnored, "ignored", "i", false, "Exibe os arquivos na denylist do repositório")
	gitCmd.AddCommand(gitListCmd)
}
