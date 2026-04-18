// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"tae/internal/vcs"
	"fmt"
	"path/filepath"
	"tae/internal/storage"
)

// resolveTagPaths consulta o metadado da tag e resolve os caminhos com base no escopo
func resolveTagPaths(tagName string, targets []string) ([]string, error) {
	meta, err := storage.GetTagMeta(tagName)
	if err != nil {
		return nil, err
	}

	var resolved []string
	if meta.Type == storage.TagTypeGit {
		if !vcs.IsInsideRepo() {
			return nil, fmt.Errorf("a tag '%s' pertence ao Git, mas você não está em um repositório", tagName)
		}
		currentRepoID := vcs.GetRepoID()
		if meta.RepoID != "" && meta.RepoID != currentRepoID {
			displayName := meta.RepoName
			if displayName == "" { displayName = meta.RepoID } // Fallback
			return nil, fmt.Errorf("a tag '%s' pertence a outro repositório Git (%s)", tagName, displayName)
		}

		for _, t := range targets {
			relPath, err := vcs.GetRelativePath(t)
			if err != nil {
				return nil, fmt.Errorf("falha no alvo '%s': %w", t, err)
			}
			resolved = append(resolved, relPath)
		}
	} else {
		for _, t := range targets {
			absPath, err := filepath.Abs(t)
			if err != nil {
				return nil, fmt.Errorf("caminho inválido '%s': %w", t, err)
			}
			resolved = append(resolved, absPath)
		}
	}
	return resolved, nil
}

// restorePathsForDisk converte caminhos lidos do banco em caminhos absolutos testáveis no disco físico.
// Executável de qualquer lugar do sistema.
func restorePathsForDisk(tagName string, paths []string) ([]string, error) {
	meta, err := storage.GetTagMeta(tagName)
	if err != nil {
		return nil, err
	}

	if meta.Type == storage.TagTypeGit {
		gitRoot := meta.GitRoot

		// Fallback de retrocompatibilidade para tags criadas antes desta correção
		if gitRoot == "" {
			if !vcs.IsInsideRepo() || vcs.GetRepoID() != meta.RepoID {
				displayName := meta.RepoName
				if displayName == "" { displayName = meta.RepoID }
				return nil, fmt.Errorf("esta tag não possui a raiz do Git salva. Execute este comando dentro do repositório [%s] uma vez para que ela seja lida", displayName)
			}
			gitRoot = vcs.GetRoot()
		}

		var absPaths []string
		for _, p := range paths {
			absPaths = append(absPaths, filepath.ToSlash(filepath.Join(gitRoot, p)))
		}
		return absPaths, nil
	}

	return paths, nil
}
