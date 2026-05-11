// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/editor"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <nome da tag>",
	Short: "Abre o editor configurado para modificação manual do rastreamento e denylist da tag",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Autocompletar fornece as tags disponíveis no banco
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tagName := args[0]

		// Validação Fail-Fast: a tag precisa existir
		allTags, err := storage.GetAllTagsWithMeta()
		if err != nil {
			return fmt.Errorf("erro ao acessar banco de dados: %w", err)
		}
		if _, exists := allTags[tagName]; !exists {
			return fmt.Errorf("a tag '%s' não existe", tagName)
		}

		// 1. Extração do estado atual
		files, ignored, err := storage.GetTagRawKeys(tagName)
		if err != nil {
			return fmt.Errorf("erro ao ler chaves atuais da tag: %w", err)
		}

		// 2. Construção do rascunho textual
		draftBytes := editor.BuildDraft(files, ignored)

		// 3. Invocação do subprocesso do editor (Bloqueante)
		modifiedBytes, err := editor.RunEditor(draftBytes, tagName)
		if err != nil {
			return fmt.Errorf("operação abortada durante edição: %w", err)
		}

		// 4. Parser de reconciliação em memória
		newFiles, newIgnored, err := editor.ParseDraft(modifiedBytes)
		if err != nil {
			return fmt.Errorf("falha de sintaxe no rascunho: %w", err)
		}

		// 5. Persistência atômica
		if err := storage.ReplaceTagState(tagName, newFiles, newIgnored); err != nil {
			return fmt.Errorf("operação abortada. O banco não foi modificado. Erro: %w", err)
		}

		fmt.Printf("Sucesso: Estado da tag '%s' atualizado via editor.\n", tagName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
