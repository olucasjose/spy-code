// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tae/internal/render"
	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var (
	listTree     bool
	listDepth    int
	listIgnore   string
	listAbsolute bool
	listExpand   bool
)

var listCmd = &cobra.Command{
	Use:   "list [nome da tag]",
	Short: "Lista todas as tags ou os arquivos rastreados de uma tag específica",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		if len(args) == 0 {
			fmt.Println("Tags cadastradas:")
			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(storage.BucketTags))
				return b.ForEach(func(k, v []byte) error {
					fmt.Printf("  - %s\n", k)
					return nil
				})
			})
			return
		}

		tagName := args[0]
		var files []string

		db.View(func(tx *bbolt.Tx) error {
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			if filesBucket == nil { return nil }
			projFiles := filesBucket.Bucket([]byte(tagName))
			if projFiles == nil { return nil }

			return projFiles.ForEach(func(k, v []byte) error {
				files = append(files, string(k))
				return nil
			})
		})

		if len(files) == 0 {
			fmt.Printf("Alvos rastreados na tag '%s':\n  (Vazio ou tag não inicializada)\n", tagName)
			return
		}

		if listExpand {
			files = expandPathsToFiles(files)
		}

		fmt.Printf("Alvos rastreados na tag '%s':\n", tagName)

		// Exibição Absoluta (Legado)
		if listAbsolute {
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
			return
		}

		// Preparação da engine visual e caminhos relativos
		basePrefix := render.GetCommonPrefix(files)
		var ignorePatterns []string
		if listIgnore != "" {
			ignorePatterns = strings.Split(listIgnore, "|")
		}

		fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)

		if listTree {
			rootNode := render.BuildVisualTree(files, basePrefix)
			render.PrintTree(rootNode, "", 0, listDepth, ignorePatterns)
		} else {
			for _, f := range files {
				relPath := strings.TrimPrefix(f, basePrefix)
				relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
				if relPath == "" {
					relPath = filepath.Base(f)
				}
				fmt.Printf("  - %s\n", relPath)
			}
		}
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listTree, "tree", "t", false, "Exibe os caminhos em formato de árvore")
	listCmd.Flags().IntVarP(&listDepth, "level", "L", 0, "Profundidade máxima da árvore (0 = infinito)")
	listCmd.Flags().StringVarP(&listIgnore, "ignore", "I", "", "Padrões para ignorar na exibição (ex: \"node_modules|*.go\")")
	listCmd.Flags().BoolVarP(&listAbsolute, "absolute", "A", false, "Exibe os caminhos absolutos originais sem truncar")
	listCmd.Flags().BoolVarP(&listExpand, "expand", "e", false, "Expande diretórios lendo o disco físico antes de listar")
	rootCmd.AddCommand(listCmd)
}
