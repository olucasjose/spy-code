// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"go.etcd.io/bbolt"
)

// TrackPaths recebe caminhos já processados pelo resolver e insere no rastreamento.
func TrackPaths(tagName string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		projBucket := tx.Bucket([]byte(BucketTags))
		if projBucket.Get([]byte(tagName)) == nil {
			meta := EncodeTagMeta(TagMeta{Type: TagTypeLocal})
			if err := projBucket.Put([]byte(tagName), meta); err != nil {
				return fmt.Errorf("falha ao criar tag '%s' automaticamente: %w", tagName, err)
			}
		}

		filesBucket := tx.Bucket([]byte(BucketFiles))
		projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return err
		}

		ignoredBucket := tx.Bucket([]byte(BucketIgnored))
		projIgnored, err := ignoredBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return err
		}

		for _, path := range targets {
			if projIgnored.Get([]byte(path)) != nil {
				if err := projIgnored.Delete([]byte(path)); err != nil {
					return err
				}
			}
			if err := projFiles.Put([]byte(path), []byte("1")); err != nil {
				return err
			}
		}
		return nil
	})
}

// UntrackPath remove um caminho pré-processado da tag.
func UntrackPath(tagName, targetPath string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		filesBucket := tx.Bucket([]byte(BucketFiles))
		if filesBucket == nil {
			return nil
		}
		projFiles := filesBucket.Bucket([]byte(tagName))
		if projFiles == nil {
			return fmt.Errorf("tag '%s' não possui arquivos rastreados", tagName)
		}

		if projFiles.Get([]byte(targetPath)) == nil {
			return fmt.Errorf("o alvo '%s' não está rastreado diretamente", targetPath)
		}

		return projFiles.Delete([]byte(targetPath))
	})
}
