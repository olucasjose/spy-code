// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/vcs"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Agrupa comandos relacionados a operações do repositório Git",
	Long:  "Comandos utilitários para integração com o Git, permitindo listar, exportar e gerar diffs empacotados.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !vcs.IsInsideRepo() {
			return fmt.Errorf("o diretório atual não pertence a um repositório Git. Navegue até a raiz ou subdiretório de um repositório válido antes de usar os comandos 'tae git'")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
}
