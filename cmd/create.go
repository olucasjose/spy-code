// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"strings"
	"tae/internal/vcs"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var createGlobal bool

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
		isGit := !createGlobal
		if isGit {
			if !vcs.IsInsideRepo() {
				return fmt.Errorf("o escopo padrão de tags exige um repositório Git. Use --global (-g) para criar uma tag global")
			}
			repoID = vcs.GetRepoID()
			repoName = vcs.GetRepoName()
		}

		meta := storage.TagMeta{Type: storage.TagTypeGlobal}
		if isGit {
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

		if isGit {
			fmt.Printf("Tag(s) Git criada(s) com sucesso e atreladas ao repositório [%s]: %v\n", repoName, args)
		} else {
			fmt.Printf("Tag(s) Global(is) criada(s) com sucesso: %v\n", args)
		}
		return nil
	},
}

func init() {
	createCmd.Flags().BoolVarP(&createGlobal, "global", "g", false, "Cria a tag com escopo global, em vez de atrelar ao repositório Git atual")
	rootCmd.AddCommand(createCmd)
}
