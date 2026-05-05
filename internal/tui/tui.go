package tui

import (
	"fmt"
	"os"
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

	styleLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Width(8)

	styleConnectCmd = lipgloss.NewStyle().
			Foreground(lipgloss.Color("155")).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// -----------------------------------------------------------------------
// Screen enum
// -----------------------------------------------------------------------

type screen int

const (
	screenList   screen = iota // host picker
	screenDetail               // user / port confirmation form
)

// -----------------------------------------------------------------------
// Confirm-screen field index
// -----------------------------------------------------------------------

const (
	fieldUser = 0
	fieldPort = 1
	numFields = 2
)

// -----------------------------------------------------------------------
// Top-level model
// -----------------------------------------------------------------------

// Model is the Bubbletea application model.
type Model struct {
	// --- shared ---
	screen   screen
	quitting bool
	chosen   *model.Host
	width    int
	height   int

	// --- list screen ---
	allHosts     []model.Host
	filtered     []model.Host
	cursor       int
	searchInput  textinput.Model
	maxVisible   int
	scrollOffset int

	// --- detail screen ---
	pending     model.Host // host selected on list screen
	fields      [numFields]textinput.Model
	activeField int
}

// New creates a new TUI model loaded with the given host list.
func New(hosts []model.Host) Model {
	si := textinput.New()
	si.Placeholder = "Type to filter hosts…"
	si.Focus()
	si.CharLimit = 128
	si.Width = 50

	m := Model{
		allHosts:    hosts,
		filtered:    hosts,
		searchInput: si,
		maxVisible:  12,
	}
	return m
}

// buildDetailFields initialises the two text inputs for the detail screen,
// pre-filled from the selected host.
func (m *Model) buildDetailFields(h model.Host) {
	user := h.User
	if user == "" {
		// Default to the current OS user as a helpful hint.
		user = os.Getenv("USER")
	}
	port := h.Port
	if port == "" {
		port = "22"
	}

	uInput := textinput.New()
	uInput.Placeholder = "username"
	uInput.CharLimit = 64
	uInput.Width = 30
	uInput.SetValue(user)
	uInput.Focus()

	pInput := textinput.New()
	pInput.Placeholder = "22"
	pInput.CharLimit = 5
	pInput.Width = 10
	pInput.SetValue(port)

	m.fields = [numFields]textinput.Model{uInput, pInput}
	m.activeField = fieldUser
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
		available := msg.Height - 10
		if available < 3 {
			available = 3
		}
		m.maxVisible = available
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case screenList:
			return m.updateList(msg)
		case screenDetail:
			return m.updateDetail(msg)
		}
	}

	return m, nil
}

// -----------------------------------------------------------------------
// List screen update
// -----------------------------------------------------------------------

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEnter:
		if len(m.filtered) > 0 {
			m.pending = m.filtered[m.cursor]
			m.buildDetailFields(m.pending)
			m.screen = screenDetail
			return m, textinput.Blink
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
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// -----------------------------------------------------------------------
// Detail screen update
// -----------------------------------------------------------------------

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEsc:
		// Go back to the list screen.
		m.screen = screenList
		m.searchInput.Focus()
		return m, textinput.Blink

	case tea.KeyEnter:
		// Commit: pull values from inputs into the pending host, then quit.
		m.pending.User = strings.TrimSpace(m.fields[fieldUser].Value())
		port := strings.TrimSpace(m.fields[fieldPort].Value())
		if port == "22" {
			port = ""
		}
		m.pending.Port = port
		m.chosen = &m.pending
		m.quitting = true
		return m, tea.Quit

	case tea.KeyTab, tea.KeyShiftTab:
		// Cycle between User and Port fields.
		m.fields[m.activeField].Blur()
		if msg.Type == tea.KeyTab {
			m.activeField = (m.activeField + 1) % numFields
		} else {
			m.activeField = (m.activeField + numFields - 1) % numFields
		}
		m.fields[m.activeField].Focus()
		return m, textinput.Blink
	}

	// Forward all other keys to the active field.
	var cmd tea.Cmd
	m.fields[m.activeField], cmd = m.fields[m.activeField].Update(msg)
	return m, cmd
}

// -----------------------------------------------------------------------
// Filter
// -----------------------------------------------------------------------

func (m *Model) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
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

// -----------------------------------------------------------------------
// View
// -----------------------------------------------------------------------

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	switch m.screen {
	case screenDetail:
		return m.viewDetail()
	default:
		return m.viewList()
	}
}

// ---- List screen -------------------------------------------------------

func (m Model) viewList() string {
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("  SSH Selector"))
	sb.WriteString("\n")
	sb.WriteString(m.searchInput.View())
	sb.WriteString("\n\n")

	if len(m.filtered) == 0 {
		sb.WriteString(styleDimmed.Render("  No hosts match your query.\n"))
	} else {
		end := m.scrollOffset + m.maxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := m.scrollOffset; i < end; i++ {
			sb.WriteString(m.renderRow(m.filtered[i], i == m.cursor))
			sb.WriteString("\n")
		}
		if len(m.filtered) > m.maxVisible {
			sb.WriteString(styleDimmed.Render(fmt.Sprintf(
				"  %d–%d of %d", m.scrollOffset+1, end, len(m.filtered),
			)))
			sb.WriteString("\n")
		}
	}

	sb.WriteString(styleHelp.Render("  ↑/↓ navigate   enter select   esc quit"))
	return sb.String()
}

func (m Model) renderRow(h model.Host, selected bool) string {
	var badge string
	switch h.Source {
	case model.SourceSSHConfig:
		badge = styleBadgeSSH.Render("[ssh]")
	case model.SourceTailscale:
		badge = styleBadgeTS.Render("[ts] ")
	default:
		badge = styleDimmed.Render("[?]  ")
	}

	meta := h.Addr
	if h.User != "" {
		meta = h.User + "@" + h.Addr
	}
	if h.Port != "" && h.Port != "22" {
		meta += ":" + h.Port
	}

	if selected {
		return styleSelected.Render(fmt.Sprintf("▶ %s  %s  %s", badge, h.Name, styleDimmed.Render(meta)))
	}
	return styleNormal.Render(fmt.Sprintf("  %s  %s  %s", badge, h.Name, styleDimmed.Render(meta)))
}

// ---- Detail screen -----------------------------------------------------

func (m Model) viewDetail() string {
	h := m.pending
	var sb strings.Builder

	sb.WriteString(styleTitle.Render("  Connect to host"))
	sb.WriteString("\n")

	// Host summary line
	var badge string
	switch h.Source {
	case model.SourceSSHConfig:
		badge = styleBadgeSSH.Render("[ssh]")
	case model.SourceTailscale:
		badge = styleBadgeTS.Render("[ts] ")
	default:
		badge = styleDimmed.Render("[?]  ")
	}
	sb.WriteString(fmt.Sprintf("  %s  %s  %s\n\n", badge, styleNormal.Render(h.Name), styleDimmed.Render(h.Addr)))

	// User field
	userLabel := styleLabel.Render("User")
	if m.activeField == fieldUser {
		userLabel = styleLabel.Foreground(lipgloss.Color("212")).Bold(true).Render("User")
	}
	sb.WriteString(fmt.Sprintf("  %s  %s\n", userLabel, m.fields[fieldUser].View()))

	// Port field
	portLabel := styleLabel.Render("Port")
	if m.activeField == fieldPort {
		portLabel = styleLabel.Foreground(lipgloss.Color("212")).Bold(true).Render("Port")
	}
	sb.WriteString(fmt.Sprintf("  %s  %s\n", portLabel, m.fields[fieldPort].View()))

	// Preview the SSH command that will run
	sb.WriteString("\n")
	sb.WriteString(styleDimmed.Render("  Command: "))
	sb.WriteString(styleConnectCmd.Render(m.previewCmd()))
	sb.WriteString("\n")

	sb.WriteString(styleHelp.Render("  tab cycle fields   enter connect   esc back"))
	return sb.String()
}

// previewCmd builds a preview of the ssh command from the current field values.
func (m Model) previewCmd() string {
	user := strings.TrimSpace(m.fields[fieldUser].Value())
	port := strings.TrimSpace(m.fields[fieldPort].Value())

	target := m.pending.Addr
	if user != "" {
		target = user + "@" + m.pending.Addr
	}

	cmd := "ssh"
	if port != "" && port != "22" {
		cmd += " -p " + port
	}
	cmd += " " + target
	return cmd
}

// -----------------------------------------------------------------------
// Public entry point
// -----------------------------------------------------------------------

// Run starts the TUI and returns the host chosen by the user (with User and
// Port fields updated from the detail screen), or nil if the user quit.
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

// keep styleError referenced so it is available for future use
var _ = styleError
