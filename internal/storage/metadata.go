// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"encoding/json"

	"go.etcd.io/bbolt"
)

const (
	TagTypeLocal = "local"
	TagTypeGit   = "git"
)

// TagMeta define o schema do valor salvo no bucket BucketTags
type TagMeta struct {
	Type    string `json:"type"`
	RepoID  string `json:"repo_id,omitempty"`
	RepoName string `json:"repo_name,omitempty"`
	GitRoot string `json:"git_root,omitempty"` // Novo campo para viabilizar operações globais
}

// ParseTagMeta lê os metadados de uma tag com suporte a retrocompatibilidade
func ParseTagMeta(data []byte) TagMeta {
	var meta TagMeta
	if len(data) == 0 || string(data) == "{}" {
		return TagMeta{Type: TagTypeLocal}
	}
	_ = json.Unmarshal(data, &meta)
	if meta.Type == "" {
		meta.Type = TagTypeLocal
	}
	return meta
}

// EncodeTagMeta converte a struct para persistência no bbolt
func EncodeTagMeta(meta TagMeta) []byte {
	data, _ := json.Marshal(meta)
	return data
}

// GetTagMeta recupera os metadados de uma tag.
// Retorna fallback local se a tag não existir, mantendo a retrocompatibilidade e a auto-criação.
func GetTagMeta(tagName string) (TagMeta, error) {
	db, err := Open()
	if err != nil {
		return TagMeta{}, err
	}
	defer db.Close()

	var meta TagMeta
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketTags))
		if b == nil {
			return nil
		}
		data := b.Get([]byte(tagName))
		if data == nil {
			meta = TagMeta{Type: TagTypeLocal}
			return nil
		}
		meta = ParseTagMeta(data)
		return nil
	})
	return meta, err
}
