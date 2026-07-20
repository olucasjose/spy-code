// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NodeState int

const (
	StateUntracked NodeState = iota
	StateTracked
	StateIgnored
)

// Node representa um arquivo ou diretório na interface iterativa
type Node struct {
	Name           string
	Path           string
	AbsPath        string
	IsDir          bool
	State          NodeState
	IsLoaded       bool
	Size           int64
	SizeCalculated bool
	Children       []*Node
	Parent         *Node
}

// NewRootNode inicializa a raiz da árvore virtual baseada na raiz do repositório/tag
func NewRootNode(absRoot string) *Node {
	return &Node{
		Name:     filepath.Base(absRoot),
		Path:     ".",
		AbsPath:  absRoot,
		IsDir:    true,
		State:    StateUntracked,
		IsLoaded: false,
	}
}

// LoadChildren escaneia o disco sob demanda (Lazy Loading).
// Ele resolve instantaneamente o tamanho de arquivos e consulta o cache para diretórios.
func (n *Node) LoadChildren(baseRoot string, trackedMap, ignoredMap map[string]bool, dirSizes map[string]int64, calcMode string, gitIgnoredMap map[string]bool) error {
	if n.IsLoaded || !n.IsDir {
		return nil
	}

	entries, err := os.ReadDir(n.AbsPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childAbs := filepath.Join(n.AbsPath, entry.Name())
		relPath, _ := filepath.Rel(baseRoot, childAbs)
		relPath = filepath.ToSlash(relPath)

		child := &Node{
			Name:    entry.Name(),
			AbsPath: childAbs,
			Path:    relPath,
			IsDir:   entry.IsDir(),
			Parent:  n,
			State:   StateUntracked,
		}

		if trackedMap[relPath] {
			child.State = StateTracked
		} else if ignoredMap[relPath] {
			child.State = StateIgnored
		}

		// Se for modo "filtered" e estiver na denylist ou no gitignore, não calcula automaticamente
		isExcluded := false
		if calcMode == "filtered" {
			if child.State == StateIgnored || isPathGitIgnored(relPath, gitIgnoredMap) {
				isExcluded = true
			}
		}

		if !child.IsDir {
			if !isExcluded {
				if info, err := entry.Info(); err == nil {
					child.Size = info.Size()
					child.SizeCalculated = true
				}
			}
		} else {
			if !isExcluded {
				if size, exists := dirSizes[child.AbsPath]; exists {
					child.Size = size
					child.SizeCalculated = true
				}
			}
		}

		n.Children = append(n.Children, child)
	}

	n.SortChildren()

	n.IsLoaded = true
	return nil
}


// ToggleState aplica a mutação de estado baseada no input do usuário
func (n *Node) ToggleState(newState NodeState) {
	if n.State == newState {
		if n.State != StateUntracked {
			n.State = StateUntracked
		}
	} else {
		n.State = newState
	}
}

// CollectStates percorre a árvore compilando as fatias de persistência
func (n *Node) CollectStates(tracked, ignored *[]string) {
	if n.Path != "." && n.Path != "" {
		if n.State == StateTracked {
			*tracked = append(*tracked, n.Path)
		} else if n.State == StateIgnored {
			*ignored = append(*ignored, n.Path)
		}
	}

	if n.IsLoaded {
		for _, child := range n.Children {
			child.CollectStates(tracked, ignored)
		}
	}
}

// SortChildren aplica as regras de negócio em cascata para ordenação visual
func (n *Node) SortChildren() {
	sort.Slice(n.Children, func(i, j int) bool {
		a := n.Children[i]
		b := n.Children[j]

		// 1. Prioridade absoluta para nós que já tiveram o peso calculado
		if a.SizeCalculated != b.SizeCalculated {
			return a.SizeCalculated
		}

		// 2. Ordem de grandeza: decrescente por tamanho
		if a.SizeCalculated && b.SizeCalculated && a.Size != b.Size {
			return a.Size > b.Size
		}

		// 3. Fallback estrutural: Diretórios antes de arquivos, depois nome alfabético
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}
