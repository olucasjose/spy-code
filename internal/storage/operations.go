// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"go.etcd.io/bbolt"
)

// GetTagRawKeys retorna as chaves do banco exatamente como estão gravadas (sem resolução de caminhos).
func GetTagRawKeys(tagName string) (files []string, ignored []string, err error) {
	db, err := Open()
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	err = db.View(func(tx *bbolt.Tx) error {
		if b := tx.Bucket([]byte(BucketFiles)); b != nil {
			if pb := b.Bucket([]byte(tagName)); pb != nil {
				_ = pb.ForEach(func(k, v []byte) error {
					files = append(files, string(k))
					return nil
				})
			}
		}
		if b := tx.Bucket([]byte(BucketIgnored)); b != nil {
			if pb := b.Bucket([]byte(tagName)); pb != nil {
				_ = pb.ForEach(func(k, v []byte) error {
					ignored = append(ignored, string(k))
					return nil
				})
			}
		}
		return nil
	})
	return files, ignored, err
}

// RemoveKeysFromTag deleta chaves específicas dos buckets de uma tag (usado pelo prune).
func RemoveKeysFromTag(tagName string, filesToRemove, ignoredToRemove []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		remove := func(bucketName string, keys []string) error {
			if len(keys) == 0 {
				return nil
			}
			b := tx.Bucket([]byte(bucketName))
			if b == nil {
				return nil
			}
			pb := b.Bucket([]byte(tagName))
			if pb == nil {
				return nil
			}
			for _, k := range keys {
				if err := pb.Delete([]byte(k)); err != nil {
					return err
				}
			}
			return nil
		}

		if err := remove(BucketFiles, filesToRemove); err != nil {
			return err
		}
		return remove(BucketIgnored, ignoredToRemove)
	})
}

// UpdateTagScope reescreve os caminhos de uma tag resolvendo a troca de contexto (Local <-> Git).
// swapFiles e swapIgnored são mapas onde a chave é o path antigo e o valor é o novo path calculado pelo CLI.
func UpdateTagScope(tagName string, meta TagMeta, swapFiles, swapIgnored map[string]string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		tagsBucket := tx.Bucket([]byte(BucketTags))
		if tagsBucket.Get([]byte(tagName)) == nil {
			return fmt.Errorf("a tag '%s' não existe", tagName)
		}
		if err := tagsBucket.Put([]byte(tagName), EncodeTagMeta(meta)); err != nil {
			return err
		}

		updateBucket := func(bucketName string, swaps map[string]string) error {
			if len(swaps) == 0 {
				return nil
			}
			b := tx.Bucket([]byte(bucketName))
			if b == nil {
				return nil
			}
			pb := b.Bucket([]byte(tagName))
			if pb == nil {
				return nil
			}
			for oldKey, newKey := range swaps {
				if err := pb.Delete([]byte(oldKey)); err != nil {
					return err
				}
				if err := pb.Put([]byte(newKey), []byte("1")); err != nil {
					return err
				}
			}
			return nil
		}

		if err := updateBucket(BucketFiles, swapFiles); err != nil {
			return err
		}
		return updateBucket(BucketIgnored, swapIgnored)
	})
}
