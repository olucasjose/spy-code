// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/storage"
	"tae/internal/vcs"

	"github.com/spf13/cobra"
)

var gitRootFixCmd = &cobra.Command{
	Use:   "root-fix",
	Short: "Atualiza o git_root de todas as tags atreladas a este repositório caso ele tenha sido movido",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !vcs.IsInsideRepo() {
			return fmt.Errorf("o comando root-fix exige que seja executado dentro de um repositório Git")
		}

		repoID := vcs.GetRepoID()
		newRoot := vcs.GetRoot()
		repoName := vcs.GetRepoName()

		affected, err := storage.UpdateGitRootForRepo(repoID, newRoot, repoName)
		if err != nil {
			return fmt.Errorf("erro ao atualizar o git root no banco de dados: %w", err)
		}

		if affected == 0 {
			fmt.Printf("Nenhuma tag encontrada para o repositório '%s'.\n", repoName)
		} else {
			fmt.Printf("✔ %d tag(s) atualizada(s) para a nova raiz: %s\n", affected, newRoot)
		}

		return nil
	},
}

func init() {
	gitCmd.AddCommand(gitRootFixCmd)
}
