// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"go.etcd.io/bbolt"
)

// IgnorePaths move alvos pré-processados para a denylist.
func IgnorePaths(tagName string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		tagsBucket := tx.Bucket([]byte(BucketTags))
		if tagsBucket.Get([]byte(tagName)) == nil {
			meta := EncodeTagMeta(TagMeta{Type: TagTypeLocal})
			if err := tagsBucket.Put([]byte(tagName), meta); err != nil {
				return fmt.Errorf("falha ao inicializar tag: %w", err)
			}
		}

		filesBucket := tx.Bucket([]byte(BucketFiles))
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))

		projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return err
		}

		projIgnored, err := ignoredBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return err
		}

		for _, path := range targets {
			if projFiles.Get([]byte(path)) != nil {
				if err := projFiles.Delete([]byte(path)); err != nil {
					return err
				}
			}
			if err := projIgnored.Put([]byte(path), []byte("1")); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetIgnoredPaths carrega o Exclusion Index da tag em um mapa O(1).
func GetIgnoredPaths(tagName string) (map[string]bool, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ignored := make(map[string]bool)
	err = db.View(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))
		if ignoredBucket == nil {
			return nil
		}
		projIgnored := ignoredBucket.Bucket([]byte(tagName))
		if projIgnored == nil {
			return nil
		}
		return projIgnored.ForEach(func(k, v []byte) error {
			ignored[string(k)] = true
			return nil
		})
	})
	return ignored, err
}

// UnignorePaths remove alvos pré-processados da denylist.
func UnignorePaths(tagName string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))
		if ignoredBucket == nil {
			return nil
		}
		projIgnored := ignoredBucket.Bucket([]byte(tagName))
		if projIgnored == nil {
			return nil
		}

		for _, path := range targets {
			if err := projIgnored.Delete([]byte(path)); err != nil {
				return err
			}
		}
		return nil
	})
}
