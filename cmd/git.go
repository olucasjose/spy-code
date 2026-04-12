// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Agrupa comandos relacionados a operações do repositório Git",
	Long:  "Comandos utilitários para integração com o Git, permitindo listar, exportar e gerar diffs empacotados.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !isInsideGitRepo() {
			fmt.Fprintln(os.Stderr, "⚠️ Alerta: O diretório atual não pertence a um repositório Git.")
			fmt.Fprintln(os.Stderr, "Navegue até a raiz ou subdiretório de um repositório válido antes de usar os comandos 'tae git'.")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
}

// isInsideGitRepo verifica silenciosamente se o diretório atual é uma working tree válida.
func isInsideGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// streamGitBlob lê os bytes diretamente dos objetos internos do Git (isolado do disco rígido)
// e os jorra no io.Writer de destino (que pode ser um buffer de Zip ou um arquivo local vazio)
func streamGitBlob(commit, path string, dest io.Writer) error {
	gitPath := filepath.ToSlash(path) // Garante o padrão UNIX exigido pelo Git
	
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commit, gitPath))
	cmd.Stdout = dest // Streaming direto, zero desperdício de memória RAM
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao ler blob do git: %s (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// getGitRepoName extrai o nome do diretório raiz do repositório Git atual
func getGitRepoName() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "repo" // Fallback seguro
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}
