// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import "encoding/json"

const (
	TagTypeLocal = "local"
	TagTypeGit   = "git"
)

// TagMeta define o schema do valor salvo no bucket BucketTags
type TagMeta struct {
	Type   string `json:"type"`
	RepoID string `json:"repo_id,omitempty"`
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
