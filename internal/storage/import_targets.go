// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// ImportTargets copia alvos de uma tag para outra que já exista, ignorando caminhos que não existam fisicamente no contexto da tag de destino.
func ImportTargets(oldName, newName string) (int, []string, error) {
	db, err := GetDB()
	if err != nil {
		return 0, nil, err
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	// Validação Fail-Fast: A tag de destino existe?
	var destType string
	var destGitRoot sql.NullString
	err = tx.QueryRow("SELECT type, git_root FROM tags WHERE name = ?", newName).Scan(&destType, &destGitRoot)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil, fmt.Errorf("a tag destino '%s' não existe. Crie a tag primeiro antes de importar alvos para ela", newName)
		}
		return 0, nil, err
	}

	// Validar se a tag de origem existe
	var exists int
	err = tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", oldName).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil, fmt.Errorf("a tag origem '%s' não existe", oldName)
		}
		return 0, nil, err
	}

	// Recuperar alvos da tag origem (tracked)
	trackedRows, err := tx.Query("SELECT path FROM files_tracked WHERE tag_name = ?", oldName)
	if err != nil {
		return 0, nil, err
	}
	defer trackedRows.Close()

	var sourceTracked []string
	for trackedRows.Next() {
		var p string
		if err := trackedRows.Scan(&p); err != nil {
			return 0, nil, err
		}
		sourceTracked = append(sourceTracked, p)
	}

	// Recuperar alvos ignorados da tag origem
	ignoredRows, err := tx.Query("SELECT path FROM files_ignored WHERE tag_name = ?", oldName)
	if err != nil {
		return 0, nil, err
	}
	defer ignoredRows.Close()

	var sourceIgnored []string
	for ignoredRows.Next() {
		var p string
		if err := ignoredRows.Scan(&p); err != nil {
			return 0, nil, err
		}
		sourceIgnored = append(sourceIgnored, p)
	}

	var skipped []string
	importedCount := 0

	insertTracked, err := tx.Prepare("INSERT OR IGNORE INTO files_tracked (tag_name, path) VALUES (?, ?)")
	if err != nil {
		return 0, nil, err
	}
	defer insertTracked.Close()

	insertIgnored, err := tx.Prepare("INSERT OR IGNORE INTO files_ignored (tag_name, path) VALUES (?, ?)")
	if err != nil {
		return 0, nil, err
	}
	defer insertIgnored.Close()

	// Função auxiliar para testar a existência física baseada no contexto da tag destino
	testPath := func(relPath string) bool {
		checkPath := relPath
		if destType == TagTypeGit && destGitRoot.Valid && destGitRoot.String != "" {
			checkPath = filepath.Join(destGitRoot.String, relPath)
		} else if destType == TagTypeLocal {
			absPath, err := filepath.Abs(relPath)
			if err == nil {
				checkPath = absPath
			}
		}

		_, err := os.Stat(checkPath)
		return err == nil
	}

	// Inserir tracked válidos
	for _, p := range sourceTracked {
		if testPath(p) {
			res, err := insertTracked.Exec(newName, p)
			if err != nil {
				return 0, nil, err
			}
			affected, _ := res.RowsAffected()
			importedCount += int(affected)
		} else {
			skipped = append(skipped, p)
		}
	}

	// Inserir ignored válidos
	for _, p := range sourceIgnored {
		if testPath(p) {
			res, err := insertIgnored.Exec(newName, p)
			if err != nil {
				return 0, nil, err
			}
			affected, _ := res.RowsAffected()
			importedCount += int(affected)
		} else {
			skipped = append(skipped, p)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, err
	}

	return importedCount, skipped, nil
}
