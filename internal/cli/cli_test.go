package cli

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"note-tui/internal/notes"
)

func noProgram(tea.Model) error { return nil }

func TestHelpAndUsageErrors(t *testing.T) {
	store := notes.NewStore(t.TempDir())
	var out, errOut bytes.Buffer
	if code := RunWithStore([]string{"help"}, &out, &errOut, store, noProgram); code != 0 {
		t.Fatalf("help code = %d", code)
	}
	if !strings.Contains(out.String(), "note new") {
		t.Fatalf("ayuda inesperada: %s", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := RunWithStore([]string{"wat"}, &out, &errOut, store, noProgram); code != 2 {
		t.Fatalf("unknown code = %d", code)
	}
}

func TestListAndShow(t *testing.T) {
	store := notes.NewStore(t.TempDir())
	note, err := store.Save(notes.Note{Title: "Prueba CLI", Body: "contenido visible"})
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := RunWithStore([]string{"list"}, &out, &errOut, store, noProgram); code != 0 {
		t.Fatalf("list code = %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), note.Slug) || !strings.Contains(out.String(), note.Title) {
		t.Fatalf("list inesperado: %s", out.String())
	}
	out.Reset()
	if code := RunWithStore([]string{"show", note.Slug}, &out, &errOut, store, noProgram); code != 0 {
		t.Fatalf("show code = %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "contenido visible") {
		t.Fatalf("show inesperado: %s", out.String())
	}
}

func TestShowMissing(t *testing.T) {
	store := notes.NewStore(t.TempDir())
	var out, errOut bytes.Buffer
	if code := RunWithStore([]string{"show", "no-existe"}, &out, &errOut, store, noProgram); code != 1 {
		t.Fatalf("show code = %d", code)
	}
	if !strings.Contains(errOut.String(), "does not exist") {
		t.Fatalf("error inesperado: %s", errOut.String())
	}
}
