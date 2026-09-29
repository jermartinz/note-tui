package notes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("note not found")

type Note struct {
	Slug      string
	Title     string
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Store struct {
	Dir string
	now func() time.Time
}

type metadata struct {
	Title     string    `yaml:"title"`
	CreatedAt time.Time `yaml:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at"`
}

func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, "Documents", "notes"), nil
}

func NewStore(dir string) *Store {
	return &Store{Dir: dir, now: time.Now}
}

func (s *Store) Ensure() error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create notes directory: %w", err)
	}
	return nil
}

func (s *Store) List() ([]Note, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}
	result := make([]Note, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		note, err := s.loadPath(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		result = append(result, note)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (s *Store) Load(slug string) (Note, error) {
	if !validSlug(slug) {
		return Note{}, ErrNotFound
	}
	return s.loadPath(filepath.Join(s.Dir, slug+".md"))
}

func (s *Store) loadPath(path string) (Note, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Note{}, ErrNotFound
	}
	if err != nil {
		return Note{}, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Note{}, fmt.Errorf("inspect %s: %w", filepath.Base(path), err)
	}
	slug := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	note, ok := decode(data)
	if !ok {
		note = Note{
			Title:     strings.ReplaceAll(slug, "-", " "),
			Body:      string(data),
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
		}
	}
	note.Slug = slug
	return note, nil
}

func (s *Store) Save(note Note) (Note, error) {
	note.Title = strings.TrimSpace(note.Title)
	if note.Title == "" {
		return Note{}, errors.New("title cannot be empty")
	}
	if err := s.Ensure(); err != nil {
		return Note{}, err
	}
	now := s.now().UTC().Truncate(time.Second)
	if note.Slug == "" {
		note.Slug = s.availableSlug(Slugify(note.Title))
		note.CreatedAt = now
	} else if !validSlug(note.Slug) {
		return Note{}, errors.New("invalid slug")
	}
	if note.CreatedAt.IsZero() {
		note.CreatedAt = now
	}
	note.UpdatedAt = now
	data, err := encode(note)
	if err != nil {
		return Note{}, err
	}
	tmp, err := os.CreateTemp(s.Dir, ".note-*.tmp")
	if err != nil {
		return Note{}, fmt.Errorf("create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		cleanup()
		return Note{}, err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return Note{}, fmt.Errorf("write note: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return Note{}, fmt.Errorf("sync note: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return Note{}, fmt.Errorf("close note: %w", err)
	}
	target := filepath.Join(s.Dir, note.Slug+".md")
	if err := os.Rename(tmpName, target); err != nil {
		cleanup()
		return Note{}, fmt.Errorf("save note: %w", err)
	}
	return note, nil
}

func (s *Store) Trash(slug string) error {
	if !validSlug(slug) {
		return ErrNotFound
	}
	source := filepath.Join(s.Dir, slug+".md")
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	trashDir := filepath.Join(s.Dir, ".trash")
	if err := os.MkdirAll(trashDir, 0o700); err != nil {
		return fmt.Errorf("create trash directory: %w", err)
	}
	destination := filepath.Join(trashDir, slug+".md")
	if _, err := os.Stat(destination); err == nil {
		stamp := s.now().UTC().Format("20060102-150405")
		base := slug + "-" + stamp
		destination = filepath.Join(trashDir, base+".md")
		for n := 2; ; n++ {
			if _, err := os.Stat(destination); errors.Is(err, os.ErrNotExist) {
				break
			}
			destination = filepath.Join(trashDir, fmt.Sprintf("%s-%d.md", base, n))
		}
	}
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("move note to trash: %w", err)
	}
	return nil
}

func Slugify(title string) string {
	var b strings.Builder
	separator := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if separator && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			separator = false
		} else {
			separator = true
		}
	}
	if b.Len() == 0 {
		return "note"
	}
	return b.String()
}

func Search(all []Note, query string) []Note {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]Note(nil), all...)
	}
	result := make([]Note, 0)
	for _, note := range all {
		if strings.Contains(strings.ToLower(note.Title), query) || strings.Contains(strings.ToLower(note.Body), query) {
			result = append(result, note)
		}
	}
	return result
}

func (s *Store) availableSlug(base string) string {
	candidate := base
	for n := 2; ; n++ {
		_, err := os.Stat(filepath.Join(s.Dir, candidate+".md"))
		if errors.Is(err, os.ErrNotExist) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
}

func validSlug(slug string) bool {
	return slug != "" && slug == filepath.Base(slug) && !strings.ContainsAny(slug, `/\\`)
}

func encode(note Note) ([]byte, error) {
	front, err := yaml.Marshal(metadata{note.Title, note.CreatedAt, note.UpdatedAt})
	if err != nil {
		return nil, fmt.Errorf("encode metadata: %w", err)
	}
	return []byte("---\n" + string(front) + "---\n\n" + note.Body), nil
}

func decode(data []byte) (Note, bool) {
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return Note{}, false
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return Note{}, false
	}
	end += 4
	var meta metadata
	if err := yaml.Unmarshal([]byte(text[4:end]), &meta); err != nil || strings.TrimSpace(meta.Title) == "" {
		return Note{}, false
	}
	body := strings.TrimPrefix(text[end+5:], "\n")
	return Note{Title: meta.Title, Body: body, CreatedAt: meta.CreatedAt, UpdatedAt: meta.UpdatedAt}, true
}
