package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"note-tui/internal/notes"
)

func Note(note notes.Note, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", fmt.Errorf("create renderer: %w", err)
	}
	markdown := "# " + note.Title + "\n\n" + note.Body
	result, err := renderer.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("render note: %w", err)
	}
	return strings.TrimRight(result, "\n") + "\n", nil
}
