// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"go.etcd.io/bbolt"
)

// GitIgnorePaths adiciona os caminhos alvo (já processados como relativos à raiz do git)
// na denylist persistente do repositório específico.
func GitIgnorePaths(repoID string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketGitIgnored))
		
		repoBucket, err := ignoredBucket.CreateBucketIfNotExists([]byte(repoID))
		if err != nil {
			return err
		}

		for _, path := range targets {
			if err := repoBucket.Put([]byte(path), []byte("1")); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetGitIgnoredPaths retorna um hash map rápido contendo a denylist do repositório.
func GetGitIgnoredPaths(repoID string) (map[string]bool, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ignored := make(map[string]bool)
	err = db.View(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketGitIgnored))
		if ignoredBucket == nil {
			return nil
		}
		repoBucket := ignoredBucket.Bucket([]byte(repoID))
		if repoBucket == nil {
			return nil
		}
		return repoBucket.ForEach(func(k, v []byte) error {
			ignored[string(k)] = true
			return nil
		})
	})
	return ignored, err
}

// UnignoreGitPaths remove caminhos da denylist do repositório.
func UnignoreGitPaths(repoID string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketGitIgnored))
		if ignoredBucket == nil {
			return nil
		}
		repoBucket := ignoredBucket.Bucket([]byte(repoID))
		if repoBucket == nil {
			return nil
		}

		for _, path := range targets {
			if err := repoBucket.Delete([]byte(path)); err != nil {
				return err
			}
		}
		return nil
	})
}
