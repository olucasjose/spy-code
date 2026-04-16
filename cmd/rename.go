// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"strings"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <tag_antiga> <tag_nova>",
	Short: "Renomeia uma tag existente e transfere todo o seu rastreamento",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Sugere nomes apenas para o primeiro argumento (a tag existente)
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		oldTag := args[0]
		newTag := args[1]

		if strings.ToLower(newTag) == "denylist" {
			fmt.Fprintln(os.Stderr, "Erro: 'denylist' é uma palavra reservada e não pode ser usada como nome de tag.")
			os.Exit(1)
		}

		fmt.Printf("Tag '%s' renomeada para '%s' com sucesso.\n", oldTag, newTag)
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
