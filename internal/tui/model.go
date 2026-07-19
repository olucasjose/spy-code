// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Model gerencia o estado da aplicação TUI no padrão Elm Architecture
type Model struct {
	TagName        string
	BaseRoot       string
	Root           *Node
	CurrentDir     *Node
	CursorIndex    int
	UnsavedChanges bool
	PromptingExit  bool
	Quitting       bool
	Err            error

	TrackedMap map[string]bool
	IgnoredMap map[string]bool
	DirSizes   map[string]int64
}

// InitialModel inicializa a máquina injetando o contexto do banco de dados
func InitialModel(tagName, baseRoot string, trackedMap, ignoredMap map[string]bool) Model {
	root := NewRootNode(baseRoot)
	dirSizes := make(map[string]int64)

	// Carrega apenas o primeiro nível (Lazy Loading) passando o cache de tamanhos
	_ = root.LoadChildren(baseRoot, trackedMap, ignoredMap, dirSizes)

	return Model{
		TagName:    tagName,
		BaseRoot:   baseRoot,
		Root:       root,
		CurrentDir: root,
		TrackedMap: trackedMap,
		IgnoredMap: ignoredMap,
		DirSizes:   dirSizes,
	}
}

// Init cumpre a interface do Bubble Tea
func (m Model) Init() tea.Cmd {
	return nil
}
