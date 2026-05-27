// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva
package exporter

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tae/internal/config"
	"tae/internal/filter"
	"tae/internal/vcs"
)

// ExportSingleFile consolida todos os arquivos monitorados em um único arquivo texto plano otimizado para LLMs.
// Bifurca o tratamento em modo interativo (baseado em JSON) ou detecção automatizada de binários via byte stream.
func ExportSingleFile(destPath string, files []string, opts ExportOptions) error {
	var cfgFilter *config.ExtensionFilter
	if opts.Interactive {
		var err error
		cfgFilter, err = config.LoadFilter()
		if err != nil {
			return fmt.Errorf("falha na camada de configuração interativa: %w", err)
		}
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

	fmt.Fprintln(outFile, "———")
	fmt.Fprintln(outFile, " TAE Export - Single File")
	if opts.GitCommit != "" {
		fmt.Fprintf(outFile, " Commit Original: %s\n", opts.GitCommit)
	}
	fmt.Fprintln(outFile, "———")

	var reader *bufio.Reader
	if opts.Interactive {
		reader = bufio.NewReader(os.Stdin)
	}

	for _, path := range files {
		relPath := resolveRelPath(path, opts.BasePrefix, opts.FlattenMap)
		if opts.AppendTxt {
			relPath += ".txt"
		}

		// Pré-renderizamos o cabeçalho em memória para injetarmos apenas se o conteúdo passar nos filtros
		var headerBuf bytes.Buffer
		fmt.Fprintln(&headerBuf, "\n———")
		fmt.Fprintf(&headerBuf, "File: %s\n", relPath)
		fmt.Fprintln(&headerBuf, "———")

		if opts.Interactive {
			ext := strings.ToLower(filepath.Ext(path))
			skip := false

			if ext != "" {
				if cfgFilter.Blocked[ext] {
					skip = true
				} else if !cfgFilter.Allowed[ext] {
					if opts.Quiet {
						skip = true
					} else {
						fmt.Printf("\n[?] A extensão '%s' do arquivo '%s' é desconhecida.\n", ext, relPath)
						fmt.Printf("Deseja incluir seu conteúdo e PERMITIR essa extensão no futuro? [s/N]: ")
						response, _ := reader.ReadString('\n')
						response = strings.TrimSpace(strings.ToLower(response))
						if response == "s" || response == "y" {
							if err := cfgFilter.LearnExtension(ext, false); err != nil {
								fmt.Printf("Aviso: Falha ao salvar regra de permissão: %v\n", err)
							}
							skip = false
						} else {
							if err := cfgFilter.LearnExtension(ext, true); err != nil {
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

			outFile.Write(headerBuf.Bytes())
			err := writeContent(path, opts.GitCommit, outFile, br)
			if err != nil {
				fmt.Fprintf(outFile, "[Erro de I/O ao ler conteúdo deste arquivo: %v]\n", err)
				if !opts.Quiet {
					fmt.Printf("Aviso: Falha ao ler '%s': %v\n", relPath, err)
				}
			} else {
				fmt.Fprintln(outFile, "")
				if !opts.Quiet {
					fmt.Printf("  -> Anexado: %s\n", relPath)
				}
			}

		} else {
			// Rota Autônoma: Detecção profunda de binários via I/O
			if opts.GitCommit == "" {
				isBin, err := filter.IsBinaryFile(path)
				if err != nil {
					outFile.Write(headerBuf.Bytes())
					fmt.Fprintf(outFile, "[Erro ao verificar binário físico: %v]\n", err)
					if !opts.Quiet {
						fmt.Printf("Aviso: Falha ao ler '%s': %v\n", relPath, err)
					}
					continue
				}

				if isBin {
					if !opts.Quiet {
						fmt.Printf("  -> Omitido (Binário automático detectado no disco): %s\n", relPath)
					}
					continue
				}

				outFile.Write(headerBuf.Bytes())
				if err := writeContent(path, opts.GitCommit, outFile, br); err != nil {
					fmt.Fprintf(outFile, "[Erro de I/O ao ler conteúdo físico: %v]\n", err)
					if !opts.Quiet {
						fmt.Printf("Aviso: Falha de I/O na montagem do arquivo '%s': %v\n", relPath, err)
					}
				} else {
					fmt.Fprintln(outFile, "")
					if !opts.Quiet {
						fmt.Printf("  -> Anexado: %s\n", relPath)
					}
				}
			} else {
				// Rota de Fluxo Git: Injeção do buffer interceptador (Lazy write)
				fw := &filter.BinaryFilterWriter{
					Out:     outFile,
					Header:  headerBuf.Bytes(),
					Quiet:   opts.Quiet,
					RelPath: relPath,
				}

				err := writeContent(path, opts.GitCommit, fw, br)
				fw.Flush() // Gatilho obrigatório para arquivos menores que a tolerância de 512 bytes

				if err != nil && !fw.IsBinary {
					if !fw.Determined {
						outFile.Write(headerBuf.Bytes())
					}
					fmt.Fprintf(outFile, "\n[Erro de I/O na extração Git: %v]\n", err)
					if !opts.Quiet {
						fmt.Printf("Aviso: Falha estrutural ao extrair o path Git '%s': %v\n", relPath, err)
					}
				} else if !fw.IsBinary {
					fmt.Fprintln(outFile, "")
					if !opts.Quiet {
						fmt.Printf("  -> Anexado: %s\n", relPath)
					}
				}
			}
		}
	}

	return nil
}
