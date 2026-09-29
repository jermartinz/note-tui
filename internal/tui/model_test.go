package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"note-tui/internal/notes"
)

func testModel(t *testing.T) Model {
	t.Helper()
	store := notes.NewStore(t.TempDir())
	for _, note := range []notes.Note{
		{Title: "Compras", Body: "pan y café"},
		{Title: "Trabajo", Body: "seguimiento del proyecto"},
	} {
		if _, err := store.Save(note); err != nil {
			t.Fatal(err)
		}
	}
	m := New(store, "")
	m.width, m.height = 80, 24
	m.resizeInputs()
	m.reload()
	return m
}

func sendKey(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	updated, _ := m.Update(msg)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update devolvió %T", updated)
	}
	return result
}

func TestSearchFiltersTitleAndBody(t *testing.T) {
	m := testModel(t)
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.screen != searchScreen {
		t.Fatalf("screen = %v", m.screen)
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("proyecto")})
	if len(m.visible) != 1 || m.visible[0].Title != "Trabajo" {
		t.Fatalf("resultados inesperados: %#v", m.visible)
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != listScreen {
		t.Fatalf("screen después de Enter = %v", m.screen)
	}
}

func TestEditorProtectsUnsavedChangesAndSaves(t *testing.T) {
	m := testModel(t)
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.screen != editScreen {
		t.Fatalf("screen = %v", m.screen)
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Nueva")})
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != unsavedScreen {
		t.Fatalf("no se mostró confirmación: %v", m.screen)
	}
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.screen != listScreen || len(m.all) != 3 {
		t.Fatalf("guardado inesperado: screen=%v notas=%d err=%v", m.screen, len(m.all), m.err)
	}
}

func TestDeleteMovesSelectedNoteToTrash(t *testing.T) {
	m := testModel(t)
	slug := m.visible[0].Slug
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = sendKey(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.screen != listScreen || len(m.all) != 1 {
		t.Fatalf("borrado inesperado: screen=%v notas=%d err=%v", m.screen, len(m.all), m.err)
	}
	if _, err := os.Stat(filepath.Join(m.store.Dir, ".trash", slug+".md")); err != nil {
		t.Fatal(err)
	}
}
