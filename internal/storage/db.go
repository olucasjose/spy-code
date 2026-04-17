// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// getDBPath resolve o caminho seguro para o banco de dados global
func getDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("falha ao localizar diretório home: %w", err)
	}

	dir := filepath.Join(home, ".tae")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório base: %w", err)
	}

	return filepath.Join(dir, "tae.db"), nil
}

// Open inicia a conexão com o SQLite e garante a existência das tabelas
func Open() (*sql.DB, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, err
	}

	// _pragma=foreign_keys(1) garante que o SQLite respeite CASCADE e restrições
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)", dbPath)
	
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("falha ao conectar no banco sqlite: %w", err)
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tags (
		name TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		repo_id TEXT,
		repo_name TEXT,
		git_root TEXT
	);

	CREATE TABLE IF NOT EXISTS files_tracked (
		tag_name TEXT,
		path TEXT,
		PRIMARY KEY (tag_name, path),
		FOREIGN KEY (tag_name) REFERENCES tags(name) ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS files_ignored (
		tag_name TEXT,
		path TEXT,
		PRIMARY KEY (tag_name, path),
		FOREIGN KEY (tag_name) REFERENCES tags(name) ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS git_ignored (
		repo_id TEXT,
		path TEXT,
		PRIMARY KEY (repo_id, path)
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("falha ao criar tabelas internas: %w", err)
	}
	return nil
}

// GetAllTags retorna uma lista com os nomes de todas as tags cadastradas no banco
func GetAllTags() ([]string, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM tags")
	if err != nil {
		return nil, fmt.Errorf("falha ao listar tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
