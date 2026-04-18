// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"tae/internal/vcs"
	"fmt"
	"strings"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var createGit bool

var createCmd = &cobra.Command{
	Use:   "create <nome1> [nome2...]",
	Short: "Cria uma ou mais tags de rastreamento no banco de dados",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, tagName := range args {
			if strings.ToLower(tagName) == "denylist" {
				return fmt.Errorf("'denylist' é uma palavra reservada do sistema e não pode ser usada como nome de tag")
			}
		}

		var repoID, repoName string
		if createGit {
			if !vcs.IsInsideRepo() {
				return fmt.Errorf("a flag --git exige que o comando seja executado dentro de um repositório Git")
			}
			repoID = vcs.GetRepoID()
			repoName = vcs.GetRepoName()
		}

		meta := storage.TagMeta{Type: storage.TagTypeLocal}
		if createGit {
			meta = storage.TagMeta{
				Type:     storage.TagTypeGit,
				RepoID:   repoID,
				RepoName: repoName,
				GitRoot:  vcs.GetRoot(),
			}
		}

		if err := storage.CreateTags(args, meta); err != nil {
			return fmt.Errorf("erro na transação: %w", err)
		}

		if createGit {
			fmt.Printf("Tag(s) Git criada(s) com sucesso e atreladas ao repositório [%s]: %v\n", repoName, args)
		} else {
			fmt.Printf("Tag(s) Local(is) criada(s) com sucesso: %v\n", args)
		}
		return nil
	},
}

func init() {
	createCmd.Flags().BoolVarP(&createGit, "git", "g", false, "Cria a tag com escopo amarrado ao repositório Git atual")
	rootCmd.AddCommand(createCmd)
}
