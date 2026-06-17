// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"strings"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var importTargetsCmd = &cobra.Command{
	Use:   "import-targets <tag_origem> <tag_destino>",
	Short: "Importa os arquivos rastreados e ignorados de uma tag para outra tag existente",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		oldTag := args[0]
		newTag := args[1]

		if strings.ToLower(newTag) == "denylist" {
			return fmt.Errorf("'denylist' é uma palavra reservada e não pode ser usada como nome de tag")
		}

		importedCount, skipped, err := storage.ImportTargets(oldTag, newTag)
		if err != nil {
			return fmt.Errorf("erro ao importar alvos: %w", err)
		}

		fmt.Printf("%d alvo(s) importado(s) com sucesso da tag '%s' para a tag '%s'.\n", importedCount, oldTag, newTag)

		if len(skipped) > 0 {
			fmt.Printf("\nAviso: Os seguintes alvos não existem no projeto da tag '%s' e não foram importados:\n", newTag)
			for _, alvo := range skipped {
				fmt.Printf("%s\n", alvo)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(importTargetsCmd)
}
