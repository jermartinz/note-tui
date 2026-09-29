package notes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"My first note":     "my-first-note",
		"  Go & Terminal! ": "go-terminal",
		"Tree and coffee":   "tree-and-coffee",
		"---":               "note",
	}
	for input, want := range tests {
		if got := Slugify(input); got != want {
			t.Errorf("Slugify(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestSaveLoadListAndCollision(t *testing.T) {
	store := NewStore(t.TempDir())
	clock := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return clock }

	first, err := store.Save(Note{Title: "Una nota", Body: "hola"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Save(Note{Title: "Una nota", Body: "otra"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Slug != "una-nota" || second.Slug != "una-nota-2" {
		t.Fatalf("slugs inesperados: %q, %q", first.Slug, second.Slug)
	}

	loaded, err := store.Load(first.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != first.Title || loaded.Body != first.Body || !loaded.CreatedAt.Equal(clock) {
		t.Fatalf("nota cargada inesperada: %#v", loaded)
	}
	all, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("List() devolvió %d notas", len(all))
	}
}

func TestEditKeepsSlugAndCreationDate(t *testing.T) {
	store := NewStore(t.TempDir())
	created := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return created }
	note, err := store.Save(Note{Title: "Original"})
	if err != nil {
		t.Fatal(err)
	}

	updated := created.Add(time.Hour)
	store.now = func() time.Time { return updated }
	note.Title = "Título distinto"
	note, err = store.Save(note)
	if err != nil {
		t.Fatal(err)
	}
	if note.Slug != "original" {
		t.Fatalf("el slug cambió a %q", note.Slug)
	}
	if !note.CreatedAt.Equal(created) || !note.UpdatedAt.Equal(updated) {
		t.Fatalf("fechas incorrectas: %#v", note)
	}
}

func TestPlainMarkdownFallbackAndSearch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nota-externa.md"), []byte("Texto Importante"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	all, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Title != "nota externa" || all[0].Body != "Texto Importante" {
		t.Fatalf("fallback inesperado: %#v", all)
	}
	if got := Search(all, "importante"); len(got) != 1 {
		t.Fatalf("Search devolvió %d resultados", len(got))
	}
	if got := Search(all, "ausente"); len(got) != 0 {
		t.Fatalf("Search devolvió %d resultados", len(got))
	}
}

func TestTrash(t *testing.T) {
	store := NewStore(t.TempDir())
	note, err := store.Save(Note{Title: "Temporal"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Trash(note.Slug); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(note.Slug); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load después de Trash = %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, ".trash", note.Slug+".md")); err != nil {
		t.Fatal(err)
	}
}

func TestTrashNeverOverwritesAnExistingFile(t *testing.T) {
	store := NewStore(t.TempDir())
	clock := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return clock }
	for i := 0; i < 3; i++ {
		note, err := store.Save(Note{Title: "Repetida"})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Trash(note.Slug); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(store.Dir, ".trash"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("la papelera contiene %d archivos; want 3", len(entries))
	}
}

func TestInvalidAndEmptyTitle(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Save(Note{Title: "   "}); err == nil {
		t.Fatal("se aceptó un título vacío")
	}
	if _, err := store.Load("../secreto"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load inválido = %v", err)
	}
}

func TestAtomicSaveLeavesNoTemporaryFiles(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Save(Note{Title: "Atómica", Body: "contenido"}); err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(store.Dir, ".note-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("quedaron temporales: %v", matches)
	}
}
