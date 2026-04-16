// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

// BackupSchema define a estrutura do JSON exportado
type BackupSchema struct {
	RepoID       string               `json:"repo_id"`
	RepoName     string               `json:"repo_name,omitempty"` // Adicionado para UX
	RepoDenylist []string             `json:"repo_denylist,omitempty"`
	Tags         map[string]TagBackup `json:"tags,omitempty"`
}

type TagBackup struct {
	Meta    storage.TagMeta `json:"meta"`
	Files   []string        `json:"files,omitempty"`
	Ignored []string        `json:"ignored,omitempty"`
}

var (
	backupExport string
	backupImport string
	backupAll    bool
	backupDeny   bool
	backupTags   bool
	backupOnly   []string
)

var gitBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Gerencia a exportação e importação de tags e denylists atreladas ao repositório Git",
	Run: func(cmd *cobra.Command, args []string) {
		if backupExport == "" && backupImport == "" {
			fmt.Fprintln(os.Stderr, "Erro: Especifique a operação com --export (-e) ou --import (-i).")
			os.Exit(1)
		}
		if backupExport != "" && backupImport != "" {
			fmt.Fprintln(os.Stderr, "Erro: Não é possível exportar e importar simultaneamente.")
			os.Exit(1)
		}

		repoID := getGitRepoID()

		if backupExport != "" {
			executeExport(repoID, backupExport)
		} else {
			executeImport(repoID, backupImport)
		}
	},
}

func executeExport(repoID, destFile string) {
	if !backupAll && !backupDeny && !backupTags && len(backupOnly) == 0 {
		fmt.Fprintln(os.Stderr, "Erro: Para exportar, defina o escopo usando --all, --denylist, --tag ou --only.")
		os.Exit(1)
	}

	db, err := storage.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	backup := BackupSchema{
		RepoID:   repoID,
		RepoName: getGitRepoName(), // Injeta o nome amigável no JSON
		Tags:     make(map[string]TagBackup),
	}

	err = db.View(func(tx *bbolt.Tx) error {
		// 1. Exporta Denylist do Repositório
		if backupAll || backupDeny || containsString(backupOnly, "denylist") {
			gitIgnoredBucket := tx.Bucket([]byte(storage.BucketGitIgnored))
			if gitIgnoredBucket != nil {
				if repoBucket := gitIgnoredBucket.Bucket([]byte(repoID)); repoBucket != nil {
					_ = repoBucket.ForEach(func(k, v []byte) error {
						backup.RepoDenylist = append(backup.RepoDenylist, string(k))
						return nil
					})
				}
			}
		}

		// 2. Exporta Tags
		if backupAll || backupTags || len(backupOnly) > 0 {
			tagsBucket := tx.Bucket([]byte(storage.BucketTags))
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			ignoredBucket := tx.Bucket([]byte(storage.BucketIgnored))

			if tagsBucket != nil {
				_ = tagsBucket.ForEach(func(k, v []byte) error {
					tagName := string(k)
					meta := storage.ParseTagMeta(v)

					// Filtro de Segurança: Só exporta tags do Git atreladas a este repositório
					if meta.Type != storage.TagTypeGit || meta.RepoID != repoID {
						return nil
					}

					// Filtro de Escopo: Se --only foi usado (e não é --all/--tag), ignora se não estiver na lista
					if len(backupOnly) > 0 && !backupAll && !backupTags && !containsString(backupOnly, tagName) {
						return nil
					}

					tb := TagBackup{Meta: meta}

					if filesBucket != nil {
						if projFiles := filesBucket.Bucket(k); projFiles != nil {
							_ = projFiles.ForEach(func(fk, fv []byte) error {
								tb.Files = append(tb.Files, string(fk))
								return nil
							})
						}
					}

					if ignoredBucket != nil {
						if projIgnored := ignoredBucket.Bucket(k); projIgnored != nil {
							_ = projIgnored.ForEach(func(ik, iv []byte) error {
								tb.Ignored = append(tb.Ignored, string(ik))
								return nil
							})
						}
					}

					backup.Tags[tagName] = tb
					return nil
				})
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao montar leitura do backup: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro estrutural ao serializar backup: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erro de I/O ao salvar exportação: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sucesso: Backup do Git exportado para '%s' (Denylist: %d, Tags: %d).\n", destFile, len(backup.RepoDenylist), len(backup.Tags))
}

func executeImport(currentRepoID, srcFile string) {
	data, err := os.ReadFile(srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro de I/O ao ler o arquivo de backup: %v\n", err)
		os.Exit(1)
	}

	var backup BackupSchema
	if err := json.Unmarshal(data, &backup); err != nil {
		fmt.Fprintf(os.Stderr, "Falha no parse. Formato JSON inválido: %v\n", err)
		os.Exit(1)
	}

	// Barreira de Proteção: Validação de RepoID
	if backup.RepoID != currentRepoID {
		origem := backup.RepoName
		if origem == "" {
			origem = backup.RepoID // Fallback para backups antigos
		}
		fmt.Fprintf(os.Stderr, "Erro Fatal: O arquivo de backup pertence ao repositório [%s], mas você está tentando importá-lo no repositório atual. Operação bloqueada para evitar corrupção de rastreamento.\n", origem)
		os.Exit(1)
	}

	db, err := storage.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Update(func(tx *bbolt.Tx) error {
		// Importa Denylist do Repo
		if len(backup.RepoDenylist) > 0 {
			gitIgnoredBucket := tx.Bucket([]byte(storage.BucketGitIgnored))
			repoBucket, err := gitIgnoredBucket.CreateBucketIfNotExists([]byte(currentRepoID))
			if err != nil {
				return err
			}
			for _, p := range backup.RepoDenylist {
				if err := repoBucket.Put([]byte(p), []byte("1")); err != nil {
					return err
				}
			}
		}

		// Importa Tags
		if len(backup.Tags) > 0 {
			tagsBucket := tx.Bucket([]byte(storage.BucketTags))
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			ignoredBucket := tx.Bucket([]byte(storage.BucketIgnored))

			// Na importação, garantimos que o GitRoot seja atualizado para a máquina alvo
			currentGitRoot := getGitRoot()

			for tagName, tagData := range backup.Tags {
				meta := tagData.Meta
				meta.GitRoot = currentGitRoot

				if err := tagsBucket.Put([]byte(tagName), storage.EncodeTagMeta(meta)); err != nil {
					return err
				}

				projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(tagName))
				if err != nil {
					return err
				}
				for _, p := range tagData.Files {
					if err := projFiles.Put([]byte(p), []byte("1")); err != nil {
						return err
					}
				}

				projIgnored, err := ignoredBucket.CreateBucketIfNotExists([]byte(tagName))
				if err != nil {
					return err
				}
				for _, p := range tagData.Ignored {
					if err := projIgnored.Put([]byte(p), []byte("1")); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro fatal durante a transação de importação: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sucesso: Backup importado com segurança (Denylist: %d, Tags: %d).\n", len(backup.RepoDenylist), len(backup.Tags))
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func init() {
	gitBackupCmd.Flags().StringVarP(&backupExport, "export", "e", "", "Arquivo JSON de destino para exportar o backup")
	gitBackupCmd.Flags().StringVarP(&backupImport, "import", "i", "", "Arquivo JSON fonte para importar o backup")
	gitBackupCmd.Flags().BoolVarP(&backupAll, "all", "a", false, "Exporta tudo: denylist do repo e todas as tags do git (Requer -e)")
	gitBackupCmd.Flags().BoolVarP(&backupDeny, "denylist", "d", false, "Exporta a denylist do repositório (Requer -e)")
	gitBackupCmd.Flags().BoolVarP(&backupTags, "tag", "t", false, "Exporta todas as tags do git e suas denylists (Requer -e)")
	gitBackupCmd.Flags().StringSliceVarP(&backupOnly, "only", "o", []string{}, "Exporta apenas as tags listadas ou a 'denylist' (Requer -e. Ex: -o tag1,tag2,denylist)")

	gitCmd.AddCommand(gitBackupCmd)

	// Registra o Autocomplete apenas para a flag --only
	gitBackupCmd.RegisterFlagCompletionFunc("only", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		tags = append(tags, "denylist") // Adiciona a palavra reservada como sugestão
		return tags, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})
}
