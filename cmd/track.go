package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var trackCmd = &cobra.Command{
	Use:   "track <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Adiciona um ou mais arquivos/diretórios ao monitoramento de uma tag",
	Args:  cobra.MinimumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		// ShellCompDirectiveDefault permite que o bash sugira arquivos locais E as nossas tags
		return tags, cobra.ShellCompDirectiveDefault 
	},
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[len(args)-1]
		targets := args[:len(args)-1]

		for _, target := range targets {
			if _, err := os.Stat(target); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Aviso: O alvo '%s' não existe no disco. Ignorando.\n", target)
				continue
			}

			if err := storage.TrackPath(tagName, target); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao rastrear '%s': %v\n", target, err)
			} else {
				fmt.Printf("Alvo '%s' rastreado com sucesso na tag '%s'.\n", target, tagName)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(trackCmd)
}
