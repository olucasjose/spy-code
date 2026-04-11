// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// TrackPaths recebe múltiplos caminhos, reconcilia o Exclusion Index (apagando chaves se necessário) e insere no rastreamento.
func TrackPaths(tagName string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	var absTargets []string
	for _, t := range targets {
		absPath, err := filepath.Abs(t)
		if err != nil {
			return fmt.Errorf("caminho inválido '%s': %w", t, err)
		}
		absTargets = append(absTargets, absPath)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		projBucket := tx.Bucket([]byte(BucketTags))
		if projBucket.Get([]byte(tagName)) == nil {
			if err := projBucket.Put([]byte(tagName), []byte("{}")); err != nil {
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

		for _, absPath := range absTargets {
			// Reconciliação via Exact Match: se estava banido, deixa de estar
			if projIgnored.Get([]byte(absPath)) != nil {
				if err := projIgnored.Delete([]byte(absPath)); err != nil {
					return err
				}
			}
			// Persiste como alvo rastreado
			if err := projFiles.Put([]byte(absPath), []byte("1")); err != nil {
				return err
			}
		}
		return nil
	})
}

// UntrackPath remove um caminho do sub-bucket da tag. Utiliza abordagem Fail-Fast.
func UntrackPath(tagName, targetPath string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("caminho inválido: %w", err)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		filesBucket := tx.Bucket([]byte(BucketFiles))
		if filesBucket == nil {
			return nil
		}
		projFiles := filesBucket.Bucket([]byte(tagName))
		if projFiles == nil {
			return fmt.Errorf("tag '%s' não possui arquivos rastreados ou não existe", tagName)
		}

		// Fail-Fast: Impede falso positivo verificando a chave explícita
		if projFiles.Get([]byte(absPath)) == nil {
			return fmt.Errorf("o alvo '%s' não está rastreado diretamente na tag. Ele pode ser herdado de um diretório pai.\nPara omiti-lo da operação, utilize o comando:\n  tae ignore %s %s", targetPath, targetPath, tagName)
		}

		return projFiles.Delete([]byte(absPath))
	})
}
