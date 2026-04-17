// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

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
	listIgnored  bool
	listDetails  bool
	listGroup    bool
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
		// 1. Caso sem argumentos: Lista todas as tags isoladamente
		if len(args) == 0 {
			db, err := storage.Open()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
				os.Exit(1)
			}

			// Lógica de agrupamento por repositório
			if listGroup {
				groups := make(map[string][]string)

				db.View(func(tx *bbolt.Tx) error {
					b := tx.Bucket([]byte(storage.BucketTags))
					if b == nil {
						return nil
					}
					return b.ForEach(func(k, v []byte) error {
						meta := storage.ParseTagMeta(v)
						tag := string(k)
						repo := "No repo"
						
						if meta.Type == storage.TagTypeGit {
							repo = meta.RepoName
							if repo == "" {
								repo = meta.RepoID // Fallback
							}
						}
						
						groups[repo] = append(groups[repo], tag)
						return nil
					})
				})
				
				db.Close()

				var repos []string
				for r := range groups {
					if r != "No repo" {
						repos = append(repos, r)
					}
				}
				sort.Strings(repos)

				// Imprime "No repo" primeiro
				if tags, ok := groups["No repo"]; ok {
					fmt.Println("No repo:")
					sort.Strings(tags)
					for _, t := range tags {
						fmt.Printf("\t%s\n", t)
					}
					if len(repos) > 0 {
						fmt.Println()
					}
				}

				// Imprime os repositórios Git com nome em amarelo
				for i, r := range repos {
					// \033[33m é o código ANSI para amarelo e \033[0m reseta a formatação
					fmt.Printf("\033[33m%s:\033[0m\n", r)
					tags := groups[r]
					sort.Strings(tags)
					for _, t := range tags {
						fmt.Printf("\t%s\n", t)
					}
					if i < len(repos)-1 {
						fmt.Println()
					}
				}
				return
			}

			if !listDetails {
				fmt.Println("Tags cadastradas:")
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			if listDetails {
				fmt.Fprintln(w, "TAG\tTIPO\tREPOSITÓRIO")
			}

			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(storage.BucketTags))
				if b == nil {
					return nil
				}
				return b.ForEach(func(k, v []byte) error {
					if listDetails {
						meta := storage.ParseTagMeta(v)
						if meta.Type == storage.TagTypeGit {
							repoName := meta.RepoName
							if repoName == "" {
								repoName = meta.RepoID // Fallback
							}
							fmt.Fprintf(w, "%s\tGit\t%s\n", k, repoName)
						} else {
							fmt.Fprintf(w, "%s\tLocal\t\n", k)
						}
					} else {
						fmt.Printf("  - %s\n", k)
					}
					return nil
				})
			})

			if listDetails {
				w.Flush()
			}
			
			db.Close() // Fecha e libera o lock do arquivo
			return
		}

		tagName := args[0]

		// 2. Interceptação da Denylist isolada
		if listIgnored {
			ignoredMap, err := storage.GetIgnoredPaths(tagName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao ler Exclusion Index: %v\n", err)
				os.Exit(1)
			}

			if len(ignoredMap) == 0 {
				fmt.Printf("A denylist da tag '%s' está vazia.\n", tagName)
				return
			}

			fmt.Printf("Exclusion Index (Denylist) da tag '%s':\n", tagName)
			for path := range ignoredMap {
				fmt.Printf("  - %s\n", path)
			}
			return
		}

		// 3. Busca principal isolada (Abre e fecha o banco rapidamente numa closure)
		var files []string
		err := func() error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close() // Fecha o banco no fim deste bloco

			return db.View(func(tx *bbolt.Tx) error {
				filesBucket := tx.Bucket([]byte(storage.BucketFiles))
				if filesBucket == nil {
					return nil
				}
				projFiles := filesBucket.Bucket([]byte(tagName))
				if projFiles == nil {
					return nil
				}

				return projFiles.ForEach(func(k, v []byte) error {
					files = append(files, string(k))
					return nil
				})
			})
		}()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao consultar arquivos: %v\n", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Printf("Alvos rastreados na tag '%s':\n  (Vazio ou tag não inicializada)\n", tagName)
			return
		}

		// NOVA LÓGICA: Restaura os caminhos para bater com a raiz física se for tag Git
		resolvedFiles, err := restorePathsForDisk(tagName, files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro de escopo estrutural: %v\n", err)
			os.Exit(1)
		}
		files = resolvedFiles

		// 4. Expansão
		if listExpand {
			ignoredMap, _ := storage.GetIgnoredPaths(tagName)
			
			// Restaura a denylist para a mesma base física
			restoredIgnored := make(map[string]bool)
			var igPaths []string
			for p := range ignoredMap { igPaths = append(igPaths, p) }
			if resIgPaths, err := restorePathsForDisk(tagName, igPaths); err == nil {
				for _, p := range resIgPaths { restoredIgnored[p] = true }
			}

			files = expandPathsToFiles(files, restoredIgnored)
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
	listCmd.Flags().BoolVarP(&listIgnored, "ignored", "i", false, "Exibe apenas os arquivos na denylist permanente da tag")
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "Exibe os metadados das tags em colunas, indicando se são Local ou Git")
	listCmd.Flags().BoolVarP(&listGroup, "group", "g", false, "Agrupa a exibição de tags por repositório (com suporte a cores)")
	rootCmd.AddCommand(listCmd)
}
