// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"strings"

	"tae/internal/filter"
	"tae/internal/fs"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var ignorePattern string

var trackCmd = &cobra.Command{
	Use:   "track <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Adiciona um ou mais arquivos/diretórios ao monitoramento de uma tag",
	Args:  cobra.MinimumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tagName := args[len(args)-1]
		rawTargets := args[:len(args)-1]

		var ignorePatterns []string
		if ignorePattern != "" {
			ignorePatterns = strings.Split(ignorePattern, "|")
		}

		var validTargets []string
		for _, target := range rawTargets {
			if filter.MatchPattern(target, ignorePatterns) {
				fmt.Printf("Ignorando alvo via filtro regex: %s\n", target)
				continue
			}

			if _, err := os.Stat(target); os.IsNotExist(err) {
				fmt.Printf("Aviso: O alvo '%s' não existe no disco. Ignorando.\n", target)
				continue
			}
			validTargets = append(validTargets, target)
		}

		if len(validTargets) == 0 {
			fmt.Println("Nenhum alvo válido para rastrear.")
			return nil
		}

		resolvedTargets, err := fs.ResolveTagPaths(tagName, validTargets)
		if err != nil {
			return fmt.Errorf("erro de resolução: %w", err)
		}

		// Envia em lote para reconciliação e transação única no banco
		if err := storage.TrackPaths(tagName, resolvedTargets); err != nil {
			return fmt.Errorf("erro ao rastrear lote: %w", err)
		}

		fmt.Printf("%d alvo(s) rastreado(s) com sucesso na tag '%s'.\n", len(validTargets), tagName)
		return nil
	},
}

func init() {
	trackCmd.Flags().StringVarP(&ignorePattern, "ignore", "i", "", "Padrões para ignorar arquivos apenas na pasta atual (ex: \"node_modules|*.kt\")")
	rootCmd.AddCommand(trackCmd)
}
