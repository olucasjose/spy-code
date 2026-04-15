// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var createGit bool

var createCmd = &cobra.Command{
	Use:   "create <nome1> [nome2...]",
	Short: "Cria uma ou mais tags de rastreamento no banco de dados",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var repoID string
		if createGit {
			if !isInsideGitRepo() {
				fmt.Fprintln(os.Stderr, "Erro: A flag --git exige que o comando seja executado dentro de um repositório Git.")
				os.Exit(1)
			}
			repoID = getGitRepoID()
		}

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(storage.BucketTags))
			
			// Validação Fail-Fast para evitar estado inconsistente
			for _, tagName := range args {
				if b.Get([]byte(tagName)) != nil {
					return fmt.Errorf("a tag '%s' já existe. Operação abortada", tagName)
				}
			}
			
			// Preparação do metadado
			meta := storage.TagMeta{Type: storage.TagTypeLocal}
			if createGit {
				meta = storage.TagMeta{
					Type:   storage.TagTypeGit,
					RepoID: repoID,
					GitRoot: getGitRoot(),
				}
			}
			encodedMeta := storage.EncodeTagMeta(meta)

			// Persistência
			for _, tagName := range args {
				if err := b.Put([]byte(tagName), encodedMeta); err != nil {
					return fmt.Errorf("erro ao gravar tag '%s': %w", tagName, err)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na transação: %v\n", err)
			os.Exit(1)
		}

		if createGit {
			fmt.Printf("Tag(s) Git criada(s) com sucesso e atreladas ao repositório [%s]: %v\n", repoID, args)
		} else {
			fmt.Printf("Tag(s) Local(is) criada(s) com sucesso: %v\n", args)
		}
	},
}

func init() {
	createCmd.Flags().BoolVarP(&createGit, "git", "g", false, "Cria a tag com escopo amarrado ao repositório Git atual")
	rootCmd.AddCommand(createCmd)
}
