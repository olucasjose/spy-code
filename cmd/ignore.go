// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var ignoreCmd = &cobra.Command{
	Use:   "ignore <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Adiciona arquivos ou diretórios à blacklist da tag (Exclusion Index)",
	Args:  cobra.MinimumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveDefault
	},
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[len(args)-1]
		targets := args[:len(args)-1]

		if err := storage.IgnorePaths(tagName, targets); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao ignorar alvos: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%d alvo(s) adicionado(s) à blacklist da tag '%s'.\n", len(targets), tagName)
	},
}

func init() {
	rootCmd.AddCommand(ignoreCmd)
}
