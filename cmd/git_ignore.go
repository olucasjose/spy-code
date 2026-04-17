// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var gitIgnoreRemove bool

var gitIgnoreCmd = &cobra.Command{
	Use:   "ignore <arquivo1...>",
	Short: "Gerencia a denylist (Exclusion Index) atrelada ao repositório Git atual",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoID := getGitRepoID()

		var validTargets []string
		for _, target := range args {
			relPath, err := getGitRelativePath(target)
			if err != nil {
				fmt.Printf("Aviso: %v. Ignorando alvo.\n", err)
				continue
			}
			validTargets = append(validTargets, relPath)
		}

		if len(validTargets) == 0 {
			return fmt.Errorf("nenhum alvo válido pertencente a este repositório foi fornecido")
		}

		if gitIgnoreRemove {
			if err := storage.UnignoreGitPaths(repoID, validTargets); err != nil {
				return fmt.Errorf("erro ao remover alvos da denylist do repositório: %w", err)
			}
			fmt.Printf("%d alvo(s) removido(s) da denylist do repositório.\n", len(validTargets))
			return nil
		}

		if err := storage.GitIgnorePaths(repoID, validTargets); err != nil {
			return fmt.Errorf("erro ao adicionar alvos à denylist do repositório: %w", err)
		}

		fmt.Printf("%d alvo(s) adicionado(s) à denylist do repositório.\n", len(validTargets))
		return nil
	},
}

func init() {
	gitIgnoreCmd.Flags().BoolVarP(&gitIgnoreRemove, "remove", "r", false, "Remove os alvos da denylist do repositório")
	gitCmd.AddCommand(gitIgnoreCmd)
}
