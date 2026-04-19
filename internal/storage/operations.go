// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
)

// GetTagRawKeys retorna as chaves do banco exatamente como estão gravadas.
func GetTagRawKeys(tagName string) (files []string, ignored []string, err error) {
	db, err := GetDB()
	if err != nil {
		return nil, nil, err
	}

	fRows, err := db.Query("SELECT path FROM files_tracked WHERE tag_name = ?", tagName)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao consultar arquivos rastreados: %w", err)
	}
	for fRows.Next() {
		var p string
		if err := fRows.Scan(&p); err != nil {
			fRows.Close()
			return nil, nil, fmt.Errorf("erro ao escanear arquivo rastreado: %w", err)
		}
		files = append(files, p)
	}
	fRows.Close()

	iRows, err := db.Query("SELECT path FROM files_ignored WHERE tag_name = ?", tagName)
	if err != nil {
		return nil, nil, fmt.Errorf("erro ao consultar arquivos ignorados: %w", err)
	}
	for iRows.Next() {
		var p string
		if err := iRows.Scan(&p); err != nil {
			iRows.Close()
			return nil, nil, fmt.Errorf("erro ao escanear arquivo ignorado: %w", err)
		}
		ignored = append(ignored, p)
	}
	iRows.Close()

	return files, ignored, nil
}

// RemoveKeysFromTag deleta chaves específicas dos buckets de uma tag (usado pelo prune).
func RemoveKeysFromTag(tagName string, filesToRemove, ignoredToRemove []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(filesToRemove) > 0 {
		stmt, err := tx.Prepare("DELETE FROM files_tracked WHERE tag_name = ? AND path = ?")
		if err != nil {
			return fmt.Errorf("erro ao preparar remoção de files_tracked: %w", err)
		}
		for _, f := range filesToRemove {
			if _, err := stmt.Exec(tagName, f); err != nil {
				stmt.Close()
				return fmt.Errorf("erro ao remover arquivo rastreado '%s': %w", f, err)
			}
		}
		stmt.Close()
	}

	if len(ignoredToRemove) > 0 {
		stmt, err := tx.Prepare("DELETE FROM files_ignored WHERE tag_name = ? AND path = ?")
		if err != nil {
			return fmt.Errorf("erro ao preparar remoção de files_ignored: %w", err)
		}
		for _, i := range ignoredToRemove {
			if _, err := stmt.Exec(tagName, i); err != nil {
				stmt.Close()
				return fmt.Errorf("erro ao remover arquivo ignorado '%s': %w", i, err)
			}
		}
		stmt.Close()
	}

	return tx.Commit()
}

// UpdateTagScope reescreve os caminhos de uma tag resolvendo a troca de contexto (Local <-> Git).
func UpdateTagScope(tagName string, meta TagMeta, swapFiles, swapIgnored map[string]string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec("UPDATE tags SET type = ?, repo_id = ?, repo_name = ?, git_root = ? WHERE name = ?",
		meta.Type, meta.RepoID, meta.RepoName, meta.GitRoot, tagName)
	if err != nil {
		return fmt.Errorf("erro ao atualizar metadados da tag: %w", err)
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("a tag '%s' não existe", tagName)
	}

	if len(swapFiles) > 0 {
		delStmt, err := tx.Prepare("DELETE FROM files_tracked WHERE tag_name = ? AND path = ?")
		if err != nil {
			return fmt.Errorf("erro ao preparar deleção em files_tracked: %w", err)
		}
		insStmt, err := tx.Prepare("INSERT INTO files_tracked (tag_name, path) VALUES (?, ?)")
		if err != nil {
			delStmt.Close()
			return fmt.Errorf("erro ao preparar inserção em files_tracked: %w", err)
		}
		for oldKey, newKey := range swapFiles {
			if _, err := delStmt.Exec(tagName, oldKey); err != nil {
				delStmt.Close()
				insStmt.Close()
				return fmt.Errorf("erro ao deletar path antigo '%s': %w", oldKey, err)
			}
			if _, err := insStmt.Exec(tagName, newKey); err != nil {
				delStmt.Close()
				insStmt.Close()
				return fmt.Errorf("erro ao inserir path novo '%s': %w", newKey, err)
			}
		}
		delStmt.Close()
		insStmt.Close()
	}

	if len(swapIgnored) > 0 {
		delStmt, err := tx.Prepare("DELETE FROM files_ignored WHERE tag_name = ? AND path = ?")
		if err != nil {
			return fmt.Errorf("erro ao preparar deleção em files_ignored: %w", err)
		}
		insStmt, err := tx.Prepare("INSERT INTO files_ignored (tag_name, path) VALUES (?, ?)")
		if err != nil {
			delStmt.Close()
			return fmt.Errorf("erro ao preparar inserção em files_ignored: %w", err)
		}
		for oldKey, newKey := range swapIgnored {
			if _, err := delStmt.Exec(tagName, oldKey); err != nil {
				delStmt.Close()
				insStmt.Close()
				return fmt.Errorf("erro ao deletar path ignorado antigo '%s': %w", oldKey, err)
			}
			if _, err := insStmt.Exec(tagName, newKey); err != nil {
				delStmt.Close()
				insStmt.Close()
				return fmt.Errorf("erro ao inserir path ignorado novo '%s': %w", newKey, err)
			}
		}
		delStmt.Close()
		insStmt.Close()
	}

	return tx.Commit()
}
