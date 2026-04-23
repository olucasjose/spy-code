// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva
package exporter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tae/internal/config"
	"tae/internal/vcs"
)

// ExportSingleFile consolida todos os arquivos monitorados em um único arquivo texto plano otimizado para LLMs.
// Removemos listagens redundantes no topo para priorizar a densidade de informação útil por token.
func ExportSingleFile(destPath string, files []string, opts ExportOptions) error {
	filter, err := config.LoadFilter()
	if err != nil {
		return fmt.Errorf("falha na camada de configuração: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("erro de I/O ao criar diretório base: %w", err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo unificado: %w", err)
	}
	defer outFile.Close()

	var br *vcs.BatchReader
	if opts.GitCommit != "" {
		gitRoot := vcs.GetRoot()
		var errGit error
		br, errGit = vcs.NewBatchReader(gitRoot)
		if errGit != nil {
			return fmt.Errorf("falha crítica no motor do Git: %w", errGit)
		}
		defer br.Close()
	}

	// Ordenação Hierárquica em Memória para consistência no dump
	sort.Slice(files, func(i, j int) bool {
		relI := resolveRelPath(files[i], opts.BasePrefix, opts.FlattenMap)
		relJ := resolveRelPath(files[j], opts.BasePrefix, opts.FlattenMap)
		dirI := filepath.Dir(relI)
		dirJ := filepath.Dir(relJ)

		if dirI == dirJ {
			return filepath.Base(relI) < filepath.Base(relJ)
		}
		return dirI < dirJ
	})

	// 1. Cabeçalho Minimalista (VP/CEO optimization)
	fmt.Fprintln(outFile, "———")
	fmt.Fprintln(outFile, " TAE Export - Single File")
	if opts.GitCommit != "" {
		fmt.Fprintf(outFile, " Commit Original: %s\n", opts.GitCommit)
	}
	fmt.Fprintln(outFile, "———")

	reader := bufio.NewReader(os.Stdin)

	// 2. Despeja o conteúdo de cada arquivo sequencialmente
	for _, path := range files {
		relPath := resolveRelPath(path, opts.BasePrefix, opts.FlattenMap)
		if opts.AppendTxt {
			relPath += ".txt"
		}

		// Identificador de arquivo (Gerente optimization: o caminho já está aqui)
		fmt.Fprintln(outFile, "\n———")
		fmt.Fprintf(outFile, "File: %s\n", relPath)
		fmt.Fprintln(outFile, "———")

		ext := strings.ToLower(filepath.Ext(path))
		skip := false

		if ext != "" {
			if filter.Blocked[ext] {
				skip = true
			} else if !filter.Allowed[ext] {
				if opts.Quiet {
					skip = true
				} else {
					fmt.Printf("\n[?] A extensão '%s' do arquivo '%s' é desconhecida.\n", ext, relPath)
					fmt.Printf("Deseja incluir seu conteúdo e PERMITIR essa extensão no futuro? [s/N]: ")
					response, _ := reader.ReadString('\n')
					response = strings.TrimSpace(strings.ToLower(response))
					if response == "s" || response == "y" {
						if err := filter.LearnExtension(ext, false); err != nil {
							fmt.Printf("Aviso: Falha ao salvar regra de permissão: %v\n", err)
						}
						skip = false
					} else {
						if err := filter.LearnExtension(ext, true); err != nil {
							fmt.Printf("Aviso: Falha ao salvar regra de bloqueio: %v\n", err)
						}
						skip = true
					}
				}
			}
		} else {
			if opts.Quiet {
				skip = true
			} else {
				fmt.Printf("\n[?] O arquivo '%s' não possui extensão.\n", relPath)
				fmt.Printf("Deseja incluir seu conteúdo nesta exportação? [s/N]: ")
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				skip = !(response == "s" || response == "y")
			}
		}

		if skip {
			if !opts.Quiet {
				fmt.Printf("  -> Omitido: %s\n", relPath)
			}
			continue
		}

		err := writeContent(path, opts.GitCommit, outFile, br)
		if err != nil {
			fmt.Fprintf(outFile, "[Erro de I/O ao ler conteúdo deste arquivo: %v]\n", err)
			if !opts.Quiet {
				fmt.Printf("Aviso: Falha ao ler '%s': %v\n", relPath, err)
			}
		} else {
			fmt.Fprintln(outFile, "")
		}

		if !opts.Quiet {
			fmt.Printf("  -> Anexado: %s\n", relPath)
		}
	}

	return nil
}
