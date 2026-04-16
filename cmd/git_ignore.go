// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var gitIgnoreRemove bool

var gitIgnoreCmd = &cobra.Command{
	Use:   "ignore <arquivo1...>",
	Short: "Gerencia a denylist (Exclusion Index) atrelada ao repositório Git atual",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoID := getGitRepoID()

		var validTargets []string
		for _, target := range args {
			relPath, err := getGitRelativePath(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Aviso: %v. Ignorando alvo.\n", err)
				continue
			}
			validTargets = append(validTargets, relPath)
		}

		if len(validTargets) == 0 {
			fmt.Println("Erro: Nenhum alvo válido pertencente a este repositório foi fornecido.")
			os.Exit(1)
		}

		if gitIgnoreRemove {
			if err := storage.UnignoreGitPaths(repoID, validTargets); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao remover alvos da denylist do repositório: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%d alvo(s) removido(s) da denylist do repositório.\n", len(validTargets))
			return
		}

		if err := storage.GitIgnorePaths(repoID, validTargets); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao adicionar alvos à denylist do repositório: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%d alvo(s) adicionado(s) à denylist do repositório.\n", len(validTargets))
	},
}

func init() {
	gitIgnoreCmd.Flags().BoolVarP(&gitIgnoreRemove, "remove", "r", false, "Remove os alvos da denylist do repositório")
	gitCmd.AddCommand(gitIgnoreCmd)
}
