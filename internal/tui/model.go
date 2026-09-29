package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"note-tui/internal/notes"
	mdrender "note-tui/internal/render"
)

type screen int

const (
	listScreen screen = iota
	searchScreen
	readScreen
	editScreen
	deleteScreen
	unsavedScreen
)

type Model struct {
	store       *notes.Store
	all         []notes.Note
	visible     []notes.Note
	cursor      int
	width       int
	height      int
	screen      screen
	previous    screen
	title       textinput.Model
	body        textarea.Model
	search      textinput.Model
	focus       int
	editing     notes.Note
	dirty       bool
	reader      string
	viewport    viewport.Model
	status      string
	err         error
	startTitle  string
	startEditor bool
}

var (
	accent     = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	muted      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selected   = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	box        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
)

func New(store *notes.Store, newTitle string) Model {
	title := textinput.New()
	title.Placeholder = "Title"
	title.CharLimit = 120
	body := textarea.New()
	body.Placeholder = "Write your note…"
	body.ShowLineNumbers = false
	search := textinput.New()
	search.Prompt = "/ "
	search.Placeholder = "search titles and content"
	return Model{
		store: store, title: title, body: body, search: search, viewport: viewport.New(80, 20),
		startTitle: newTitle, startEditor: newTitle != "",
	}
}

func NewEditor(store *notes.Store, title string) Model {
	m := New(store, title)
	m.startEditor = true
	return m
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeInputs()
		if m.screen == readScreen {
			m.renderSelected()
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	if m.all == nil {
		m.reload()
		if m.startEditor {
			m.beginEdit(notes.Note{Title: m.startTitle})
			m.startEditor = false
		}
	}

	switch m.screen {
	case listScreen:
		return m.updateList(message)
	case searchScreen:
		return m.updateSearch(message)
	case readScreen:
		return m.updateRead(message)
	case editScreen:
		return m.updateEditor(message)
	case deleteScreen:
		return m.updateDelete(message)
	case unsavedScreen:
		return m.updateUnsaved(message)
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…\n"
	}
	var content string
	switch m.screen {
	case listScreen, searchScreen:
		content = m.listView()
	case readScreen:
		content = m.readView()
	case editScreen:
		content = m.editorView()
	case deleteScreen:
		content = m.listView() + "\n" + box.Render("Move this note to .trash?  y confirm · n cancel")
	case unsavedScreen:
		content = m.editorView() + "\n" + box.Render("There are unsaved changes  s save · d discard · esc cancel")
	}
	if m.err != nil {
		content += "\n" + errorStyle.Render("Error: "+m.err.Error())
	} else if m.status != "" {
		content += "\n" + muted.Render(m.status)
	}
	return content
}

func (m Model) updateList(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor+1 < len(m.visible) {
			m.cursor++
		}
	case "n":
		m.beginEdit(notes.Note{})
	case "e":
		if note, ok := m.selected(); ok {
			m.beginEdit(note)
		}
	case "enter":
		if _, ok := m.selected(); ok {
			m.screen = readScreen
			m.renderSelected()
		}
	case "d":
		if _, ok := m.selected(); ok {
			m.screen = deleteScreen
		}
	case "/":
		m.screen = searchScreen
		m.search.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) updateSearch(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.search.SetValue("")
			m.visible = notes.Search(m.all, "")
			m.cursor = 0
			m.search.Blur()
			m.screen = listScreen
			return m, nil
		case "enter":
			m.search.Blur()
			m.screen = listScreen
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(message)
	m.visible = notes.Search(m.all, m.search.Value())
	m.cursor = 0
	return m, cmd
}

func (m Model) updateRead(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "esc":
		m.screen = listScreen
		return m, nil
	case "e":
		if note, ok := m.selected(); ok {
			m.beginEdit(note)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(message)
	return m, cmd
}

func (m Model) updateEditor(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+s":
			m.saveEditor()
			return m, nil
		case "esc":
			if m.isDirty() {
				m.previous = editScreen
				m.screen = unsavedScreen
			} else {
				m.screen = listScreen
			}
			return m, nil
		case "tab", "shift+tab":
			if m.focus == 0 {
				m.focus = 1
				m.title.Blur()
				m.body.Focus()
				return m, textarea.Blink
			}
			m.focus = 0
			m.body.Blur()
			m.title.Focus()
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	if m.focus == 0 {
		m.title, cmd = m.title.Update(message)
	} else {
		m.body, cmd = m.body.Update(message)
	}
	m.dirty = m.isDirty()
	return m, cmd
}

func (m Model) updateDelete(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(key.String()) {
	case "y", "s":
		if note, ok := m.selected(); ok {
			if err := m.store.Trash(note.Slug); err != nil {
				m.err = err
			} else {
				m.status = "Note moved to .trash"
			}
		}
		m.reload()
		m.screen = listScreen
	case "n", "esc":
		m.screen = listScreen
	}
	return m, nil
}

func (m Model) updateUnsaved(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(key.String()) {
	case "s":
		m.saveEditor()
	case "d":
		m.screen = listScreen
	case "esc", "n":
		m.screen = editScreen
	}
	return m, nil
}

func (m *Model) beginEdit(note notes.Note) {
	m.editing = note
	m.title.SetValue(note.Title)
	m.body.SetValue(note.Body)
	m.focus = 0
	m.title.Focus()
	m.body.Blur()
	m.dirty = false
	m.err = nil
	m.status = ""
	m.screen = editScreen
	m.resizeInputs()
}

func (m *Model) saveEditor() {
	m.editing.Title = m.title.Value()
	m.editing.Body = m.body.Value()
	saved, err := m.store.Save(m.editing)
	if err != nil {
		m.err = err
		return
	}
	m.editing = saved
	m.dirty = false
	m.status = "Note saved: " + saved.Slug
	m.err = nil
	m.reload()
	for i := range m.visible {
		if m.visible[i].Slug == saved.Slug {
			m.cursor = i
			break
		}
	}
	m.screen = listScreen
}

func (m *Model) reload() {
	all, err := m.store.List()
	if err != nil {
		m.err = err
		m.all = []notes.Note{}
	} else {
		m.err = nil
		m.all = all
	}
	m.visible = notes.Search(m.all, m.search.Value())
	if len(m.visible) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
}

func (m *Model) renderSelected() {
	note, ok := m.selected()
	if !ok {
		m.reader = ""
		return
	}
	rendered, err := mdrender.Note(note, m.width-4)
	if err != nil {
		m.err = err
		return
	}
	m.reader = rendered
	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
}

func (m *Model) resizeInputs() {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	m.title.Width = w
	m.body.SetWidth(w)
	h := m.height - 8
	if h < 3 {
		h = 3
	}
	m.body.SetHeight(h)
	m.viewport.Width = w
	m.viewport.Height = m.height - 4
	if m.viewport.Height < 1 {
		m.viewport.Height = 1
	}
}

func (m Model) selected() (notes.Note, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return notes.Note{}, false
	}
	return m.visible[m.cursor], true
}

func (m Model) isDirty() bool {
	return m.title.Value() != m.editing.Title || m.body.Value() != m.editing.Body
}

func (m Model) listView() string {
	var b strings.Builder
	b.WriteString(accent.Render("note"))
	b.WriteString("  ")
	if m.screen == searchScreen {
		b.WriteString(m.search.View())
	} else {
		b.WriteString(muted.Render("n new · enter open · e edit · d delete · / search · q quit"))
	}
	b.WriteString("\n\n")
	if len(m.visible) == 0 {
		if m.search.Value() != "" {
			b.WriteString("No results.\n")
		} else {
			b.WriteString("No notes yet. Press n to create one.\n")
		}
		return b.String()
	}
	maxRows := m.height - 5
	if maxRows < 1 {
		maxRows = 1
	}
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := start; i < end; i++ {
		n := m.visible[i]
		line := fmt.Sprintf("%-28s  %s", truncate(n.Title, 28), n.UpdatedAt.Local().Format("2006-01-02 15:04"))
		if i == m.cursor {
			b.WriteString(selected.Render("› " + line))
		} else {
			b.WriteString("  " + line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func (m Model) readView() string {
	return accent.Render("note") + "  " + muted.Render("↑/↓ scroll · e edit · esc back") + "\n\n" + m.viewport.View()
}

func (m Model) editorView() string {
	return accent.Render("Edit note") + "  " + muted.Render("tab switch field · ctrl+s save · esc back") +
		"\n\n" + m.title.View() + "\n\n" + m.body.View()
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width < 2 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}
