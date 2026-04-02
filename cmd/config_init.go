package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var configInitCmd = &cobra.Command{
	Use:   "config init",
	Short: "Inicializa o banco de dados e a pasta root do Tae no sistema",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro fatal na inicialização: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		fmt.Println("Sucesso! Banco de dados estruturado em ~/.tae/tae.db")
	},
}

func init() {
	// Gambiarra necessária porque o Cobra precisa de um comando pai válido
	// Vamos atrelar direto ao rootCmd por enquanto, simulando "tae config-init"
	// Mas o ideal é criarmos o comando pai "config" depois.
	rootCmd.AddCommand(configInitCmd)
}
