package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/romain/sshselector/internal/model"
)

// -----------------------------------------------------------------------
// Styles
// -----------------------------------------------------------------------

var (
	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("236"))

	styleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	styleDimmed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	styleBadgeSSH = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	styleBadgeTS = lipgloss.NewStyle().
			Foreground(lipgloss.Color("135")).
			Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1)

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// -----------------------------------------------------------------------
// Model
// -----------------------------------------------------------------------

// SelectedHost is returned by Run when the user picks a host.
type SelectedHost = model.Host

// Model is the Bubbletea application model.
type Model struct {
	allHosts     []model.Host
	filtered     []model.Host
	cursor       int
	input        textinput.Model
	chosen       *model.Host
	quitting     bool
	width        int
	height       int
	maxVisible   int
	scrollOffset int
}

// New creates a new TUI model loaded with the given host list.
func New(hosts []model.Host) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter hosts…"
	ti.Focus()
	ti.CharLimit = 128
	ti.Width = 50

	m := Model{
		allHosts:   hosts,
		filtered:   hosts,
		input:      ti,
		maxVisible: 12,
	}
	return m
}

// -----------------------------------------------------------------------
// Bubbletea interface
// -----------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Recompute how many list rows fit; leave room for header + input + help.
		available := msg.Height - 10
		if available < 3 {
			available = 3
		}
		m.maxVisible = available

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				chosen := m.filtered[m.cursor]
				m.chosen = &chosen
				m.quitting = true
				return m, tea.Quit
			}

		case tea.KeyUp, tea.KeyCtrlP:
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollOffset {
					m.scrollOffset--
				}
			}

		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor >= m.scrollOffset+m.maxVisible {
					m.scrollOffset++
				}
			}

		default:
			// Any other key is handled by the text input for filtering.
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.applyFilter()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.input.Value()))
	if query == "" {
		m.filtered = m.allHosts
		m.cursor = 0
		m.scrollOffset = 0
		return
	}

	var out []model.Host
	for _, h := range m.allHosts {
		haystack := strings.ToLower(h.Name + " " + h.Addr + " " + h.User)
		if strings.Contains(haystack, query) {
			out = append(out, h)
		}
	}
	m.filtered = out
	m.cursor = 0
	m.scrollOffset = 0
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	// Title
	sb.WriteString(styleTitle.Render("  SSH Selector"))
	sb.WriteString("\n")

	// Search input
	sb.WriteString(m.input.View())
	sb.WriteString("\n\n")

	// Host list
	if len(m.filtered) == 0 {
		sb.WriteString(styleDimmed.Render("  No hosts match your query.\n"))
	} else {
		end := m.scrollOffset + m.maxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := m.scrollOffset; i < end; i++ {
			h := m.filtered[i]
			line := m.renderRow(h, i == m.cursor)
			sb.WriteString(line)
			sb.WriteString("\n")
		}

		// Scroll indicator
		total := len(m.filtered)
		if total > m.maxVisible {
			indicator := fmt.Sprintf("  %d–%d of %d", m.scrollOffset+1, end, total)
			sb.WriteString(styleDimmed.Render(indicator))
			sb.WriteString("\n")
		}
	}

	// Help bar
	sb.WriteString(styleHelp.Render("  ↑/↓ navigate   enter connect   esc quit"))

	return sb.String()
}

func (m Model) renderRow(h model.Host, selected bool) string {
	// Source badge
	var badge string
	switch h.Source {
	case model.SourceSSHConfig:
		badge = styleBadgeSSH.Render("[ssh]")
	case model.SourceTailscale:
		badge = styleBadgeTS.Render("[ts] ")
	default:
		badge = styleDimmed.Render("[?]  ")
	}

	// Build the main label
	name := h.Name
	meta := h.Addr
	if h.User != "" {
		meta = h.User + "@" + h.Addr
	}
	if h.Port != "" && h.Port != "22" {
		meta += ":" + h.Port
	}

	var row string
	if selected {
		arrow := "▶ "
		row = styleSelected.Render(fmt.Sprintf("%s%s  %s  %s", arrow, badge, name, styleDimmed.Render(meta)))
	} else {
		row = styleNormal.Render(fmt.Sprintf("  %s  %s  %s", badge, name, styleDimmed.Render(meta)))
	}
	return row
}

// -----------------------------------------------------------------------
// Public entry point
// -----------------------------------------------------------------------

// Run starts the TUI and returns the host chosen by the user, or nil if
// the user quit without selecting anything. The second return is any
// non-fatal startup error (e.g. no hosts found).
func Run(hosts []model.Host) (*model.Host, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts found — add entries to ~/.ssh/config or connect to a Tailscale network")
	}

	m := New(hosts)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("tui error: %w", err)
	}

	fm := finalModel.(Model)
	return fm.chosen, nil
}

// Suppress unused import of styleError (reserved for future inline error display).
var _ = styleError
