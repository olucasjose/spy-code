// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
)

// DuplicateTag realiza a cópia atômica dos metadados e listas de rastreamento de uma tag para um novo nome.
func DuplicateTag(oldName, newName string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Validação Fail-Fast: O destino já existe?
	var exists int
	err = tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", newName).Scan(&exists)
	if err == nil {
		return fmt.Errorf("a tag destino '%s' já existe. Operação abortada", newName)
	} else if err != sql.ErrNoRows {
		return err
	}

	// Etapa 1: Copia os metadados base da tag original
	res, err := tx.Exec(`
		INSERT INTO tags (name, type, repo_id, repo_name, git_root)
		SELECT ?, type, repo_id, repo_name, git_root 
		FROM tags WHERE name = ?`, newName, oldName)
	if err != nil {
		return fmt.Errorf("erro ao copiar metadados da tag: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("a tag origem '%s' não existe", oldName)
	}

	// Etapa 2: Copia a lista de arquivos rastreados (files_tracked)
	_, err = tx.Exec(`
		INSERT INTO files_tracked (tag_name, path)
		SELECT ?, path FROM files_tracked WHERE tag_name = ?`, newName, oldName)
	if err != nil {
		return fmt.Errorf("erro ao duplicar rastreamento de arquivos: %w", err)
	}

	// Etapa 3: Copia a denylist (files_ignored)
	_, err = tx.Exec(`
		INSERT INTO files_ignored (tag_name, path)
		SELECT ?, path FROM files_ignored WHERE tag_name = ?`, newName, oldName)
	if err != nil {
		return fmt.Errorf("erro ao duplicar exclusion index (denylist): %w", err)
	}

	return tx.Commit()
}
