// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"strings"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var duplicateCmd = &cobra.Command{
	Use:   "duplicate <tag_origem> <tag_destino>",
	Short: "Cria uma cópia exata de uma tag, incluindo seus metadados e arquivos rastreados",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Sugere nomes apenas para o primeiro argumento (a tag origem existente)
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		oldTag := args[0]
		newTag := args[1]

		if strings.ToLower(newTag) == "denylist" {
			return fmt.Errorf("'denylist' é uma palavra reservada e não pode ser usada como nome de tag")
		}

		if err := storage.DuplicateTag(oldTag, newTag); err != nil {
			return fmt.Errorf("erro ao duplicar tag: %w", err)
		}

		fmt.Printf("Tag '%s' duplicada para '%s' com sucesso.\n", oldTag, newTag)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(duplicateCmd)
}
