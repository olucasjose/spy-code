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
}

// InitialModel inicializa a máquina injetando o contexto do banco de dados
func InitialModel(tagName, baseRoot string, trackedMap, ignoredMap map[string]bool) Model {
	root := NewRootNode(baseRoot)
	
	// Carrega apenas o primeiro nível (Lazy Loading)
	_ = root.LoadChildren(baseRoot, trackedMap, ignoredMap)

	return Model{
		TagName:    tagName,
		BaseRoot:   baseRoot,
		Root:       root,
		CurrentDir: root,
		TrackedMap: trackedMap,
		IgnoredMap: ignoredMap,
	}
}

// Init cumpre a interface do Bubble Tea
func (m Model) Init() tea.Cmd {
	return nil
}
