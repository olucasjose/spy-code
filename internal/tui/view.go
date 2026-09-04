// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"sort"
)

var (
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dirStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	trackedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	ignoredStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	untrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("227")).Bold(true).MarginBottom(1)
	footerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1)
	sizeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Align(lipgloss.Right)
	promptStyle    = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(0, 1).
			MarginTop(1).
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("52")).Bold(true)
)

func formatBytes(size int64) string {
	if size < 1000 {
		return fmt.Sprintf("%d B", size)
	}

	units := []string{"B", "kB", "MB", "GB", "TB", "PB"}
	value := float64(size)
	unitIndex := 0

	for value >= 1000 && unitIndex < len(units)-1 {
		value /= 1000.0
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", value, units[unitIndex])
}


// View constrói o buffer de renderização do terminal frame a frame
func (m Model) View() string {
	if m.Err != nil {
		return fmt.Sprintf("Erro fatal: %v\n", m.Err)
	}
	if m.Quitting {
		return ""
	}
	if m.Calculating || m.CalculatingStats {
		var s strings.Builder
		s.WriteString(headerStyle.Render(fmt.Sprintf("TUI Manager | Tag: %s | Caminho: %s", m.TagName, m.CurrentDir.Path)))
		if m.Calculating && m.CalculatingStats {
			s.WriteString("\n\n  Aguardando cálculo de tamanhos e estatísticas...\n\n")
		} else if m.Calculating {
			s.WriteString("\n\n  Aguardando cálculo de tamanhos...\n\n")
		} else {
			s.WriteString("\n\n  Aguardando cálculo de estatísticas...\n\n")
		}
		s.WriteString(footerStyle.Render("Ctrl+c / q: Cancelar e Sair"))
		return s.String()
	}
	if m.ShowHelp {
		var s strings.Builder
		s.WriteString(headerStyle.Render(fmt.Sprintf("TUI Manager | Tag: %s | Ajuda", m.TagName)))
		s.WriteString("\n\n")
		s.WriteString("  Atalhos do Teclado:\n")
		s.WriteString("  ↑/↓    : Mover cursor para cima/baixo\n")
		s.WriteString("  →      : Entrar no diretório selecionado\n")
		s.WriteString("  ←      : Voltar para o diretório pai\n")
		s.WriteString("  Espaço : Alternar estado de Rastreamento (T)\n")
		s.WriteString("  i / I  : Alternar estado de Denylist (I)\n")
		s.WriteString("  c / C  : Calcular tamanho da pasta/arquivo selecionado\n")
		s.WriteString("  Ctrl+s : Salvar alterações no banco de dados\n")
		s.WriteString("  q      : Sair\n")
		s.WriteString("  alt+e  : Mostrar/Ocultar Estatísticas\n")
		s.WriteString("  ?      : Fechar este menu de ajuda\n\n")
		s.WriteString(footerStyle.Render("Pressione ? ou esc para voltar"))
		return s.String()
	}

	var s strings.Builder

	m.AdjustScroll()

	visibleHeight := m.TerminalHeight - 5
	if visibleHeight <= 0 {
		visibleHeight = 1
	}
	childrenCount := len(m.CurrentDir.Children)
	endIndex := m.ScrollOffset + visibleHeight
	if endIndex > childrenCount {
		endIndex = childrenCount
	}

	headerText := fmt.Sprintf("TUI Manager | Tag: %s | Caminho: %s", m.TagName, m.CurrentDir.Path)
	if childrenCount > visibleHeight {
		headerText += fmt.Sprintf(" (Itens %d-%d de %d)", m.ScrollOffset+1, endIndex, childrenCount)
	}

	s.WriteString(headerStyle.Render(headerText))
	s.WriteString("\n")

	if childrenCount == 0 {
		s.WriteString(untrackedStyle.Render("  (Diretório vazio)\n"))
	}

	for i := m.ScrollOffset; i < endIndex; i++ {
		child := m.CurrentDir.Children[i]
		cursor := "  "
		if m.CursorIndex == i {
			cursor = cursorStyle.Render("> ")
		}

		name := child.Name
		if child.IsDir {
			name = dirStyle.Render(name + "/")
		}

		stateMarker := untrackedStyle.Render("[ ]")
		if child.State == StateTracked {
			stateMarker = trackedStyle.Render("[T]")
		} else if child.State == StateIgnored {
			stateMarker = ignoredStyle.Render("[I]")
		}

		sizeStr := "[?]"
		if child.SizeCalculated {
			sizeStr = formatBytes(child.Size)
		}
		// Formatado para %11s para acomodar o crescimento da string com as casas decimais (ex: "11.35 kB")
		formattedSize := sizeStyle.Render(fmt.Sprintf("%11s", sizeStr))

		line := fmt.Sprintf("%s%s %s %s", cursor, stateMarker, formattedSize, name)
		if m.CursorIndex == i {
			bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))
			line = bgStyle.Render(line)
			bgCode := strings.TrimSuffix(bgStyle.Render(""), "\x1b[0m")
			line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgCode) + "\x1b[0m"
		}
		s.WriteString(line + "\n")
	}

	if m.PromptingExit {
		s.WriteString(promptStyle.Render("⚠ ALTERAÇÕES NÃO SALVAS DETECTADAS!\nDeseja salvar antes de sair? [s] Sim / [n] Não / [esc] Cancelar"))
	} else {
		help := "Pressione ? para ajuda • q: Sair • alt+e: Estatísticas"
		if m.UnsavedChanges {
			help += lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(" (Alterações Pendentes)")
		}
		s.WriteString(footerStyle.Render(help))

		if m.StatsEnabled && m.StatsData != nil {
			s.WriteString("\n\n")
			s.WriteString(headerStyle.Render("Estatísticas da Tag"))
			s.WriteString("\n\n")

			s.WriteString(dirStyle.Render("Binários"))
			s.WriteString("\n")
			if len(m.StatsData.Binary) == 0 {
				s.WriteString("  Nenhum arquivo binário\n")
			} else {
				for _, ec := range sortMap(m.StatsData.Binary) {
					s.WriteString(fmt.Sprintf("  %s - %d\n", ec.Ext, ec.Count))
				}
			}
			s.WriteString("\n")
			
			s.WriteString(dirStyle.Render("Não Binários"))
			s.WriteString("\n")
			if len(m.StatsData.NonBinary) == 0 {
				s.WriteString("  Nenhum arquivo não binário\n")
			} else {
				for _, ec := range sortMap(m.StatsData.NonBinary) {
					s.WriteString(fmt.Sprintf("  %s - %d\n", ec.Ext, ec.Count))
				}
			}
		}
	}

	return s.String()
}

type extCount struct {
	Ext   string
	Count int
}

func sortMap(m map[string]int) []extCount {
	var lst []extCount
	for k, v := range m {
		lst = append(lst, extCount{k, v})
	}
	sort.Slice(lst, func(i, j int) bool {
		if lst[i].Count == lst[j].Count {
			return lst[i].Ext < lst[j].Ext
		}
		return lst[i].Count > lst[j].Count
	})
	return lst

}
