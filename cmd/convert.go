// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var (
	convertToGit bool
	convertToTae bool
)

var convertCmd = &cobra.Command{
	Use:   "convert <nome da tag>",
	Short: "Converte uma tag entre os escopos Local e Git",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if convertToGit == convertToTae {
			fmt.Fprintln(os.Stderr, "Erro: Use --git (-g) OU --tae (-t) para definir o destino da conversão.")
			os.Exit(1)
		}

		tagName := args[0]

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		// Transação atômica
		err = db.Update(func(tx *bbolt.Tx) error {
			tagsBucket := tx.Bucket([]byte(storage.BucketTags))
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			ignoredBucket := tx.Bucket([]byte(storage.BucketIgnored))

			tagData := tagsBucket.Get([]byte(tagName))
			if tagData == nil {
				return fmt.Errorf("a tag '%s' não existe", tagName)
			}
			meta := storage.ParseTagMeta(tagData)

			projFiles := filesBucket.Bucket([]byte(tagName))
			projIgnored := ignoredBucket.Bucket([]byte(tagName))

			if convertToGit {
				if meta.Type == storage.TagTypeGit {
					return fmt.Errorf("a tag '%s' já pertence ao Git", tagName)
				}
				if !isInsideGitRepo() {
					return fmt.Errorf("você precisa estar dentro de um repositório Git para converter esta tag")
				}

				repoID := getGitRepoID()

				if err := convertBucketToGit(projFiles); err != nil {
					return fmt.Errorf("arquivos rastreados: %w", err)
				}
				if err := convertBucketToGit(projIgnored); err != nil {
					return fmt.Errorf("denylist: %w", err)
				}

				meta.Type = storage.TagTypeGit
				meta.RepoID = repoID
				meta.GitRoot = getGitRoot()
				return tagsBucket.Put([]byte(tagName), storage.EncodeTagMeta(meta))

			} else {
				if meta.Type == storage.TagTypeLocal {
					return fmt.Errorf("a tag '%s' já é Local", tagName)
				}
				if !isInsideGitRepo() || getGitRepoID() != meta.RepoID {
					return fmt.Errorf("você precisa estar dentro do repositório Git original (%s) para reverter esta tag", meta.RepoID)
				}

				gitRoot := getGitRoot()

				if err := convertBucketToLocal(projFiles, gitRoot); err != nil {
					return fmt.Errorf("arquivos rastreados: %w", err)
				}
				if err := convertBucketToLocal(projIgnored, gitRoot); err != nil {
					return fmt.Errorf("denylist: %w", err)
				}

				meta.Type = storage.TagTypeLocal
				meta.RepoID = ""
				return tagsBucket.Put([]byte(tagName), storage.EncodeTagMeta(meta))
			}
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Operação abortada. O banco não foi modificado.\nErro: %v\n", err)
			os.Exit(1)
		}

		if convertToGit {
			fmt.Printf("Sucesso: A tag '%s' foi convertida para escopo Git.\n", tagName)
		} else {
			fmt.Printf("Sucesso: A tag '%s' foi convertida para escopo Local (Tae).\n", tagName)
		}
	},
}

func init() {
	convertCmd.Flags().BoolVarP(&convertToGit, "git", "g", false, "Converte uma tag Local para Git")
	convertCmd.Flags().BoolVarP(&convertToTae, "tae", "t", false, "Converte uma tag Git para Local (Tae)")
	rootCmd.AddCommand(convertCmd)
}

func convertBucketToGit(b *bbolt.Bucket) error {
	if b == nil {
		return nil
	}

	var toAdd [][]byte
	var toDelete [][]byte

	err := b.ForEach(func(k, v []byte) error {
		relPath, err := getGitRelativePath(string(k))
		if err != nil {
			return fmt.Errorf("o caminho '%s' está fora do repositório. Mova ou remova-o da tag antes de converter", string(k))
		}
		if string(k) != relPath {
			toDelete = append(toDelete, k)
			toAdd = append(toAdd, []byte(relPath))
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, k := range toDelete {
		_ = b.Delete(k)
	}
	for _, k := range toAdd {
		_ = b.Put(k, []byte("1"))
	}

	return nil
}

func convertBucketToLocal(b *bbolt.Bucket, gitRoot string) error {
	if b == nil {
		return nil
	}

	var toAdd [][]byte
	var toDelete [][]byte

	err := b.ForEach(func(k, v []byte) error {
		absPath := filepath.ToSlash(filepath.Join(gitRoot, string(k)))
		toDelete = append(toDelete, k)
		toAdd = append(toAdd, []byte(absPath))
		return nil
	})
	if err != nil {
		return err
	}

	for _, k := range toDelete {
		_ = b.Delete(k)
	}
	for _, k := range toAdd {
		_ = b.Put(k, []byte("1"))
	}

	return nil
}
