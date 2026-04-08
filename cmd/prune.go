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

var pruneAll bool

var pruneCmd = &cobra.Command{
	Use:   "prune [nome1] [nome2...]",
	Short: "Remove do rastreamento os arquivos que não existem mais no disco",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && !pruneAll {
			fmt.Fprintln(os.Stderr, "Erro: Informe pelo menos uma tag ou use a flag --all (-a) para atualizar todas.")
			os.Exit(1)
		}

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.Update(func(tx *bbolt.Tx) error {
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			if filesBucket == nil {
				return nil
			}

			var targetTags []string
			if pruneAll {
				tagsBucket := tx.Bucket([]byte(storage.BucketTags))
				if tagsBucket != nil {
					tagsBucket.ForEach(func(k, v []byte) error {
						targetTags = append(targetTags, string(k))
						return nil
					})
				}
			} else {
				targetTags = args
			}

			totalPruned := 0

			for _, tagName := range targetTags {
				projFiles := filesBucket.Bucket([]byte(tagName))
				if projFiles == nil {
					if !pruneAll {
						fmt.Printf("Aviso: Tag '%s' não encontrada ou sem arquivos rastreados.\n", tagName)
					}
					continue
				}

				var keysToDelete [][]byte

				// Identificação Fail-Fast: os.Stat acusa IsNotExist
				projFiles.ForEach(func(k, v []byte) error {
					path := string(k)
					if _, err := os.Stat(path); os.IsNotExist(err) {
						keysToDelete = append(keysToDelete, k)
					}
					return nil
				})

				// Deleção segura e isolada da iteração
				for _, k := range keysToDelete {
					if err := projFiles.Delete(k); err != nil {
						return fmt.Errorf("falha ao remover chave interna '%s': %w", string(k), err)
					}
					totalPruned++
				}
				
				if len(keysToDelete) > 0 {
					fmt.Printf("Tag '%s': %d arquivo(s) fantasma(s) removido(s).\n", tagName, len(keysToDelete))
				}
			}

			if totalPruned == 0 {
				fmt.Println("Nenhum arquivo fantasma encontrado. O rastreamento está atualizado.")
			} else {
				fmt.Printf("\nTotal podado do banco de dados: %d arquivo(s).\n", totalPruned)
			}

			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro fatal na transação de limpeza: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&pruneAll, "all", "a", false, "Aplica a verificação e limpeza em todas as tags cadastradas")
	rootCmd.AddCommand(pruneCmd)
}
