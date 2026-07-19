// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package tui

import (
	"strings"
	"tae/internal/storage"

	tea "github.com/charmbracelet/bubbletea"
)

// Update é a malha assíncrona de interceptação de inputs e mutação de estado
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.PromptingExit {
			return m.handlePromptKeys(msg)
		}
		return m.handleNavKeys(msg)
	case error:
		m.Err = msg
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handlePromptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S":
		if err := m.saveState(); err != nil {
			m.Err = err
		}
		m.Quitting = true
		return m, tea.Quit
	case "n", "N":
		m.Quitting = true
		return m, tea.Quit
	case "esc":
		m.PromptingExit = false
		return m, nil
	}
	return m, nil
}

func (m Model) handleNavKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	childrenCount := len(m.CurrentDir.Children)

	switch msg.String() {
	case "ctrl+c":
		m.Quitting = true
		return m, tea.Quit

	case "ctrl+q":
		if m.UnsavedChanges {
			m.PromptingExit = true
			return m, nil
		}
		m.Quitting = true
		return m, tea.Quit

	case "ctrl+s":
		if err := m.saveState(); err != nil {
			m.Err = err
			m.Quitting = true
			return m, tea.Quit
		}
		m.UnsavedChanges = false
		return m, nil

	case "up":
		if m.CursorIndex > 0 {
			m.CursorIndex--
		}
	case "down":
		if m.CursorIndex < childrenCount-1 {
			m.CursorIndex++
		}
	case "left":
		if m.CurrentDir.Parent != nil {
			m.CurrentDir = m.CurrentDir.Parent
			m.CursorIndex = 0
		}
	case "right":
		if childrenCount > 0 {
			selected := m.CurrentDir.Children[m.CursorIndex]
			if selected.IsDir {
				// Continua o Lazy Loading propagando o cache do banco
				_ = selected.LoadChildren(m.BaseRoot, m.TrackedMap, m.IgnoredMap)
				m.CurrentDir = selected
				m.CursorIndex = 0
			}
		}
	case " ":
		if childrenCount > 0 {
			m.CurrentDir.Children[m.CursorIndex].ToggleState(StateTracked)
			m.UnsavedChanges = true
		}
	case "i", "I":
		if childrenCount > 0 {
			m.CurrentDir.Children[m.CursorIndex].ToggleState(StateIgnored)
			m.UnsavedChanges = true
		}
	}

	return m, nil
}

// saveState compila a árvore virtual com as chaves intocadas do banco (Resolução Plana)
func (m Model) saveState() error {
	var tracked, ignored []string
	memoryNodes := make(map[string]*Node)

	// 1. Snapshot da memória (Apenas galhos visitados)
	var traverse func(n *Node)
	traverse = func(n *Node) {
		memoryNodes[n.Path] = n
		if n.IsLoaded {
			for _, c := range n.Children {
				traverse(c)
			}
		}
	}
	traverse(m.Root)

	// 2. Extração explícita
	for path, n := range memoryNodes {
		if path == "." || path == "" {
			continue
		}
		if n.State == StateTracked {
			tracked = append(tracked, path)
		} else if n.State == StateIgnored {
			ignored = append(ignored, path)
		}
	}

	// 3. Reconciliação (Previne deleção em cascata de itens não carregados no Lazy Load)
	isDBKeyValid := func(dbPath string) bool {
		parts := strings.Split(dbPath, "/")
		currentPath := ""
		
		for _, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}
			
			if n, exists := memoryNodes[currentPath]; exists {
				// Override detectado na cadeia de ancestrais
				if n.State != StateUntracked {
					return false
				}
			}
		}
		
		// Nó final exato foi carregado (Regra já dita na etapa 2)
		if _, exists := memoryNodes[dbPath]; exists {
			return false
		}
		
		return true
	}

	for path := range m.TrackedMap {
		if isDBKeyValid(path) {
			tracked = append(tracked, path)
		}
	}

	for path := range m.IgnoredMap {
		if isDBKeyValid(path) {
			ignored = append(ignored, path)
		}
	}

	return storage.ReplaceTagState(m.TagName, tracked, ignored)
}
