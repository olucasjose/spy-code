// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"path/filepath"

	"tae/internal/render"

	"github.com/spf13/cobra"
)

var (
	gitListTree   bool
	gitListDepth  int
	gitListIgnore string
)

var gitListCmd = &cobra.Command{
	Use:   "list <commit>",
	Short: "Lista recursivamente os arquivos presentes em um commit",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
			// O Git já retorna caminhos relativos à raiz do repositório,
			// então usamos "" como basePrefix para a árvore.
			rootNode := render.BuildVisualTree(files, "")
			render.PrintTree(rootNode, "", 0, gitListDepth, ignorePatterns)
		} else {
			for _, f := range files {
				// Reaproveita a lógica de ignore do renderizador
				// simulando a verificação do nome final
				if !renderIsIgnored(f, ignorePatterns) {
					fmt.Printf("  - %s\n", f)
				}
			}
		}
	},
}

// renderIsIgnored copia a checagem básica para a listagem plana
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
	gitCmd.AddCommand(gitListCmd)
}
