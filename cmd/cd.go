// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var cdCmd = &cobra.Command{
	Use:    "cd [nome da tag]",
	Short:  "Navega para o diretório de trabalho de uma tag (requer integração de shell)",
	Hidden: true, // Oculto pois a execução real é interceptada pelo tae_wrapper.sh
	Args:   cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Este código só roda se o wrapper do Bash/Zsh NÃO estiver configurado.
		return fmt.Errorf("a integração de shell não está ativa.\n" +
			"Para usar o 'tae cd', por favor configure o wrapper executando a instalação:\n" +
			"  ./install.sh\n" +
			"Ou recarregue a sessão atual:\n" +
			"  source ~/.bashrc")
	},
}

func init() {
	rootCmd.AddCommand(cdCmd)
}
