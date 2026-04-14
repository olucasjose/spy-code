// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	gitIgnoreRemove bool
	gitIgnoreExport string
	gitIgnoreImport string
)

var gitIgnoreCmd = &cobra.Command{
	Use:   "ignore [arquivo1...] [--export <arquivo.json>] [--import <arquivo.json>]",
	Short: "Gerencia a denylist (Exclusion Index) atrelada ao repositório Git atual",
	Args: func(cmd *cobra.Command, args []string) error {
		// Libera a exigência de argumentos posicionais se estivermos operando em lote
		if gitIgnoreExport != "" || gitIgnoreImport != "" {
			return nil
		}
		if len(args) < 1 {
			return fmt.Errorf("requer ao menos 1 argumento(s) posicional, recebido %d", len(args))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		repoID := getGitRepoID()

		// 1. Motor de Exportação (JSON)
		if gitIgnoreExport != "" {
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao ler a denylist do repositório: %v\n", err)
				os.Exit(1)
			}

			var paths []string
			for path := range ignoredMap {
				paths = append(paths, path)
			}

			data, err := json.MarshalIndent(paths, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro estrutural ao serializar denylist: %v\n", err)
				os.Exit(1)
			}

			if err := os.WriteFile(gitIgnoreExport, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Erro de I/O ao salvar exportação: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Denylist exportada com sucesso para '%s' (%d alvos).\n", gitIgnoreExport, len(paths))
			return
		}

		// 2. Motor de Importação em Lote
		if gitIgnoreImport != "" {
			data, err := os.ReadFile(gitIgnoreImport)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro de I/O ao ler o arquivo de importação: %v\n", err)
				os.Exit(1)
			}

			var importedPaths []string
			if err := json.Unmarshal(data, &importedPaths); err != nil {
				fmt.Fprintf(os.Stderr, "Falha no parse do arquivo. Formato JSON inválido: %v\n", err)
				os.Exit(1)
			}

			if len(importedPaths) == 0 {
				fmt.Println("O arquivo importado está vazio.")
				return
			}

			// Saneamento rigoroso: garante que o JSON não injete sujeira no disco atual
			var validTargets []string
			for _, target := range importedPaths {
				relPath, err := getGitRelativePath(target)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Aviso: %v. Ignorando alvo importado.\n", err)
					continue
				}
				validTargets = append(validTargets, relPath)
			}

			if len(validTargets) == 0 {
				fmt.Println("Erro: Nenhum alvo válido pertencente a este repositório foi extraído do JSON.")
				os.Exit(1)
			}

			if err := storage.GitIgnorePaths(repoID, validTargets); err != nil {
				fmt.Fprintf(os.Stderr, "Erro fatal ao injetar alvos no banco de dados: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("%d alvo(s) importado(s) com sucesso para a denylist do repositório.\n", len(validTargets))
			return
		}

		// 3. Fluxo Padrão: Adição / Remoção Manual
		var validTargets []string
		for _, target := range args {
			relPath, err := getGitRelativePath(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Aviso: %v. Ignorando alvo.\n", err)
				continue
			}
			validTargets = append(validTargets, relPath)
		}

		if len(validTargets) == 0 {
			fmt.Println("Erro: Nenhum alvo válido pertencente a este repositório foi fornecido.")
			os.Exit(1)
		}

		if gitIgnoreRemove {
			if err := storage.UnignoreGitPaths(repoID, validTargets); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao remover alvos da denylist do repositório: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%d alvo(s) removido(s) da denylist do repositório.\n", len(validTargets))
			return
		}

		if err := storage.GitIgnorePaths(repoID, validTargets); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao adicionar alvos à denylist do repositório: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%d alvo(s) adicionado(s) à denylist do repositório.\n", len(validTargets))
	},
}

func init() {
	gitIgnoreCmd.Flags().BoolVarP(&gitIgnoreRemove, "remove", "r", false, "Remove os alvos da denylist do repositório")
	gitIgnoreCmd.Flags().StringVar(&gitIgnoreExport, "export", "", "Exporta a denylist atual para o arquivo JSON especificado")
	gitIgnoreCmd.Flags().StringVar(&gitIgnoreImport, "import", "", "Importa alvos do arquivo JSON especificado para a denylist")
	gitCmd.AddCommand(gitIgnoreCmd)
}
