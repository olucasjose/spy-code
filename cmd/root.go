package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "spycode",
	Short: "Spycode é um utilitário CLI para extração e empacotamento de código",
	Long:  `Uma ferramenta modular para gerenciar, rastrear e extrair arquivos de projetos de forma inteligente.`,
}

// Execute adiciona todos os comandos filhos ao comando raiz e prepara as flags.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
