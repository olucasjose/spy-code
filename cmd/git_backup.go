// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"tae/internal/vcs"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	backupAll  bool
	backupDeny bool
	backupTags bool
	backupOnly []string
)

var gitBackupSaveCmd = &cobra.Command{
	Use:   "backup-save [diretorio_destino]",
	Short: "Exporta as tags e denylists do repositório Git para um arquivo JSON",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		destDir := "."
		if len(args) == 1 {
			destDir = args[0]
		}

		info, err := os.Stat(destDir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("o destino '%s' não é um diretório válido ou não existe", destDir)
		}

		repoID := vcs.GetRepoID()
		repoName := vcs.GetRepoName()
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s_%s_tae-backup.json", repoName, timestamp)
		destFile := filepath.Join(destDir, filename)

		return executeExport(repoID, repoName, destFile)
	},
}

var gitBackupRestoreCmd = &cobra.Command{
	Use:   "backup-restore <arquivo_backup.json>",
	Short: "Importa tags e denylists de um arquivo de backup para o repositório Git atual",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcFile := args[0]
		repoID := vcs.GetRepoID()
		
		return executeImport(repoID, srcFile)
	},
}

func executeExport(repoID, repoName, destFile string) error {
	if !backupAll && !backupDeny && !backupTags && len(backupOnly) == 0 {
		return fmt.Errorf("para exportar, defina o escopo usando --all, --denylist, --tag ou --only")
	}

	fullDump, err := storage.DumpGitRepositoryData(repoID)
	if err != nil {
		return fmt.Errorf("erro ao consultar base de dados: %w", err)
	}

	if fullDump.RepoName == "" {
		fullDump.RepoName = repoName
	}

	filteredDump := storage.BackupSchema{
		RepoID:   fullDump.RepoID,
		RepoName: fullDump.RepoName,
		Tags:     make(map[string]storage.TagBackup),
	}

	if backupAll || backupDeny || containsString(backupOnly, "denylist") {
		filteredDump.RepoDenylist = fullDump.RepoDenylist
	}

	if backupAll || backupTags || len(backupOnly) > 0 {
		for tagName, tagData := range fullDump.Tags {
			if len(backupOnly) > 0 && !backupAll && !backupTags && !containsString(backupOnly, tagName) {
				continue
			}
			filteredDump.Tags[tagName] = tagData
		}
	}

	data, err := json.MarshalIndent(filteredDump, "", "  ")
	if err != nil {
		return fmt.Errorf("erro estrutural ao serializar backup: %w", err)
	}

	if err := os.WriteFile(destFile, data, 0644); err != nil {
		return fmt.Errorf("erro de I/O ao salvar exportação: %w", err)
	}

	fmt.Printf("Sucesso: Backup do Git exportado para '%s' (Denylist: %d, Tags: %d).\n", destFile, len(filteredDump.RepoDenylist), len(filteredDump.Tags))
	return nil
}

func executeImport(currentRepoID, srcFile string) error {
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("erro de I/O ao ler o arquivo de backup: %w", err)
	}

	var backup storage.BackupSchema
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("falha no parse. Formato JSON inválido: %w", err)
	}

	if backup.RepoID != currentRepoID {
		origem := backup.RepoName
		if origem == "" {
			origem = backup.RepoID
		}
		return fmt.Errorf("o arquivo de backup pertence ao repositório [%s], mas você está tentando importá-lo no repositório atual", origem)
	}

	currentGitRoot := vcs.GetRoot()
	if err := storage.RestoreGitRepositoryData(currentGitRoot, backup); err != nil {
		return fmt.Errorf("erro fatal durante a restauração: %w", err)
	}

	fmt.Printf("Sucesso: Backup importado com segurança (Denylist: %d, Tags: %d).\n", len(backup.RepoDenylist), len(backup.Tags))
	return nil
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
	gitBackupSaveCmd.Flags().BoolVarP(&backupAll, "all", "a", false, "Exporta tudo: denylist do repo e todas as tags do git")
	gitBackupSaveCmd.Flags().BoolVarP(&backupDeny, "denylist", "d", false, "Exporta a denylist do repositório")
	gitBackupSaveCmd.Flags().BoolVarP(&backupTags, "tag", "t", false, "Exporta todas as tags do git e suas denylists")
	gitBackupSaveCmd.Flags().StringSliceVarP(&backupOnly, "only", "o", []string{}, "Exporta apenas as tags listadas ou a 'denylist' (Ex: -o tag1,tag2,denylist)")

	gitCmd.AddCommand(gitBackupSaveCmd)
	gitCmd.AddCommand(gitBackupRestoreCmd)

	gitBackupSaveCmd.RegisterFlagCompletionFunc("only", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		tags = append(tags, "denylist")
		return tags, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})
}
