// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"tae/internal/fs"
	"tae/internal/render"
	"tae/internal/storage"
	"tae/internal/vcs"

	"github.com/spf13/cobra"
)

var (
	listTree        bool
	listDepth       int
	listIgnore      string
	listAbsolute    bool
	listExpand      bool
	listIgnored     bool
	listDetails       bool
	listGroup         bool
	listCurrentRepo   bool
	listGitrootStatus bool
	listInvalidOnly   bool
)

var listCmd = &cobra.Command{
	Use:   "list [nome da tag]",
	Short: "Lista todas as tags ou os arquivos rastreados em um alvo específico",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			tagsMeta, err := storage.GetAllTagsWithMeta()
			if err != nil {
				return fmt.Errorf("erro ao carregar tags: %w", err)
			}

			if listCurrentRepo {
				if !vcs.IsInsideRepo() {
					return fmt.Errorf("a flag --current-repo exige que o comando seja executado dentro de um repositório Git")
				}
				currentRepoID := vcs.GetRepoID()
				for tag, meta := range tagsMeta {
					if meta.Type != storage.TagTypeGit || meta.RepoID != currentRepoID {
						delete(tagsMeta, tag)
					}
				}
			}

			if listGitrootStatus {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "TAG\tREPOSITÓRIO\tGIT ROOT\tSTATUS")

				var tagNames []string
				for t := range tagsMeta {
					tagNames = append(tagNames, t)
				}
				sort.Strings(tagNames)

				for _, tagName := range tagNames {
					meta := tagsMeta[tagName]
					status := "N/A"
					repoName := "-"
					gitRoot := "-"

					if meta.Type == storage.TagTypeGit {
						repoName = meta.RepoName
						if repoName == "" {
							repoName = meta.RepoID
						}
						gitRoot = meta.GitRoot
						if gitRoot == "" {
							status = "SEM GITROOT"
						} else {
							if _, err := os.Stat(gitRoot); err != nil {
								status = "INVÁLIDO (Não encontrado)"
							} else {
								status = "VÁLIDO"
							}
						}
					}

					if listInvalidOnly && (status == "VÁLIDO" || status == "N/A" || status == "SEM GITROOT") {
						continue
					}

					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", tagName, repoName, gitRoot, status)
				}
				w.Flush()
				return nil
			}

			if listGroup {
				groups := make(map[string][]string)

				for tag, meta := range tagsMeta {
					repo := "No repo"
					if meta.Type == storage.TagTypeGit {
						repo = meta.RepoName
						if repo == "" {
							repo = meta.RepoID
						}
					}
					groups[repo] = append(groups[repo], tag)
				}

				var repos []string
				for r := range groups {
					if r != "No repo" {
						repos = append(repos, r)
					}
				}
				sort.Strings(repos)

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

				for i, r := range repos {
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
				return nil
			}

			if !listDetails {
				fmt.Println("Tags cadastradas:")
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			if listDetails {
				fmt.Fprintln(w, "TAG\tTIPO\tREPOSITÓRIO")
			}

			var tagNames []string
			for t := range tagsMeta {
				tagNames = append(tagNames, t)
			}
			sort.Strings(tagNames)

			for _, tagName := range tagNames {
				meta := tagsMeta[tagName]
				if listDetails {
					if meta.Type == storage.TagTypeGit {
						repoName := meta.RepoName
						if repoName == "" {
							repoName = meta.RepoID
						}
						fmt.Fprintf(w, "%s\tGit\t%s\n", tagName, repoName)
					} else {
						fmt.Fprintf(w, "%s\tGlobal\t\n", tagName)
					}
				} else {
					fmt.Printf("%s\n", tagName)
				}
			}

			if listDetails {
				w.Flush()
			}
			return nil
		}

		tagName := args[0]

		meta, err := storage.GetTagMeta(tagName)
		if err != nil {
			if errors.Is(err, storage.ErrTagNotFound) {
				return fmt.Errorf("a tag '%s' não existe", tagName)
			}
			return fmt.Errorf("erro ao obter metadados da tag: %w", err)
		}

		if listIgnored {
			ignoredMap, err := storage.GetIgnoredPaths(tagName)
			if err != nil {
				return fmt.Errorf("erro ao ler Exclusion Index: %w", err)
			}

			if len(ignoredMap) == 0 {
				fmt.Printf("A denylist em '%s' está vazia.\n", tagName)
				return nil
			}

			fmt.Printf("Exclusion Index (Denylist) em '%s':\n", tagName)
			for path := range ignoredMap {
				fmt.Printf("%s\n", path)
			}
			return nil
		}

		files, err := storage.GetFilesByTag(tagName)
		if err != nil {
			return fmt.Errorf("erro ao consultar arquivos: %w", err)
		}

		if len(files) == 0 {
			fmt.Printf("Alvos rastreados em '%s':\n  (Vazio ou não inicializado)\n", tagName)
			return nil
		}

		if err := fs.ValidateGitRoot(meta); err != nil {
			fmt.Printf("\n\033[33m[!] AVISO: %v\033[0m\n\n", err)
		}

		resolvedFiles, err := fs.RestorePathsForDisk(tagName, meta, files)
		if err != nil {
			return fmt.Errorf("erro de escopo estrutural: %w", err)
		}
		files = resolvedFiles

		if listExpand {
			ignoredMap, _ := storage.GetIgnoredPaths(tagName)

			restoredIgnored := make(map[string]bool)
			var igPaths []string
			for p := range ignoredMap {
				igPaths = append(igPaths, p)
			}
			if resIgPaths, err := fs.RestorePathsForDisk(tagName, meta, igPaths); err == nil {
				for _, p := range resIgPaths {
					restoredIgnored[p] = true
				}
			}

			files = fs.ExpandPathsToFiles(files, restoredIgnored)
		}

		fmt.Printf("Alvos rastreados em '%s':\n", tagName)

		if listAbsolute {
			for _, f := range files {
				fmt.Printf("%s\n", f)
			}
			return nil
		}

		basePrefix := render.GetCommonPrefix(files)
		var ignorePatterns []string
		if listIgnore != "" {
			ignorePatterns = strings.Split(listIgnore, "|")
		}

		fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)

		if listTree {
			rootNode := render.BuildVisualTree(files, basePrefix)
			render.PrintTree(os.Stdout, rootNode, "", 0, listDepth, ignorePatterns)
		} else {
			for _, f := range files {
				relPath := strings.TrimPrefix(f, basePrefix)
				relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
				if relPath == "" {
					relPath = filepath.Base(f)
				}
				fmt.Printf("%s\n", relPath)
			}
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listTree, "tree", "t", false, "Exibe os caminhos em formato de árvore")
	listCmd.Flags().IntVarP(&listDepth, "level", "L", 0, "Profundidade máxima da árvore (0 = infinito)")
	listCmd.Flags().StringVarP(&listIgnore, "ignore", "I", "", "Padrões para ignorar na exibição (ex: \"node_modules|*.go\")")
	listCmd.Flags().BoolVarP(&listAbsolute, "absolute", "A", false, "Exibe os caminhos absolutos originais sem truncar")
	listCmd.Flags().BoolVarP(&listExpand, "expand", "e", false, "Expande diretórios lendo o disco físico antes de listar")
	listCmd.Flags().BoolVarP(&listIgnored, "ignored", "i", false, "Exibe apenas os arquivos na denylist permanente da tag")
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "Exibe os metadados das tags em colunas, indicando se são Global ou Git")
	listCmd.Flags().BoolVarP(&listGroup, "group", "g", false, "Agrupa a exibição de tags por repositório (com suporte a cores)")
	listCmd.Flags().BoolVarP(&listCurrentRepo, "current-repo", "c", false, "Lista apenas as tags Git atreladas ao repositório atual")
	listCmd.Flags().BoolVarP(&listGitrootStatus, "gitroot-status", "", false, "Exibe o status de validade da raiz (git root) das tags")
	listCmd.Flags().BoolVarP(&listInvalidOnly, "invalid-only", "", false, "Filtra e exibe apenas as tags com git roots inválidos (uso com --gitroot-status)")
	rootCmd.AddCommand(listCmd)
}
