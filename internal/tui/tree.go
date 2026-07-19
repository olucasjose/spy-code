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
	Name     string
	Path     string // Caminho relativo (como armazenado no BD)
	AbsPath  string // Caminho absoluto (usado para I/O em disco)
	IsDir    bool
	State    NodeState
	IsLoaded bool
	Children []*Node
	Parent   *Node
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
// Ele injeta o estado base via BD na primeira leitura ou herda do pai.
func (n *Node) LoadChildren(baseRoot string, trackedMap, ignoredMap map[string]bool) error {
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
			State:   n.State, // Herança padrão do estado do nó pai
		}

		// Overrides baseados no estado salvo no banco de dados (Apenas no load inicial)
		if trackedMap[relPath] {
			child.State = StateTracked
		} else if ignoredMap[relPath] {
			child.State = StateIgnored
		}

		n.Children = append(n.Children, child)
	}

	// Ordenação alfabética, priorizando diretórios
	sort.Slice(n.Children, func(i, j int) bool {
		if n.Children[i].IsDir == n.Children[j].IsDir {
			return strings.ToLower(n.Children[i].Name) < strings.ToLower(n.Children[j].Name)
		}
		return n.Children[i].IsDir
	})

	n.IsLoaded = true
	return nil
}

// ToggleState aplica a mutação de estado baseada no input do usuário (Espaço ou i).
// Possui efeito cascata sobre os filhos já carregados na memória.
func (n *Node) ToggleState(newState NodeState) {
	// Se a tecla aplicar o mesmo estado que o nó já tem, atua como um 'desmarcar'
	if n.State == newState {
		if n.State != StateUntracked {
			n.State = StateUntracked
		}
	} else {
		n.State = newState
	}

	n.propagateStateToLoadedChildren()
}

func (n *Node) propagateStateToLoadedChildren() {
	if !n.IsLoaded {
		return
	}
	for _, child := range n.Children {
		child.State = n.State
		child.propagateStateToLoadedChildren()
	}
}

// CollectStates percorre a árvore compilando as fatias de persistência (Tracked e Ignored).
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
