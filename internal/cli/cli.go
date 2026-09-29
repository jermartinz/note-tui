package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"note-tui/internal/notes"
	mdrender "note-tui/internal/render"
	"note-tui/internal/tui"
)

const help = `note — Markdown notes from the terminal

Usage:
  note                 Open the interactive note list
  note new [title]     Create a note in the editor
  note list            List notes
  note show <slug>     Display a rendered note
  note help            Show this help

Notes are stored in ~/Documents/notes.
`

func Run(args []string, stdout, stderr io.Writer) int {
	dir, err := notes.DefaultDir()
	if err != nil {
		fmt.Fprintln(stderr, "note:", err)
		return 1
	}
	return RunWithStore(args, stdout, stderr, notes.NewStore(dir), runProgram)
}

type programRunner func(tea.Model) error

func runProgram(model tea.Model) error {
	_, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func RunWithStore(args []string, stdout, stderr io.Writer, store *notes.Store, runner programRunner) int {
	if len(args) == 0 {
		if err := runner(tui.New(store, "")); err != nil {
			fmt.Fprintln(stderr, "note:", err)
			return 1
		}
		return 0
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprint(stdout, help)
		return 0
	case "new":
		title := strings.Join(args[1:], " ")
		if err := runner(tui.NewEditor(store, title)); err != nil {
			fmt.Fprintln(stderr, "note:", err)
			return 1
		}
		return 0
	case "list":
		if len(args) != 1 {
			return usageError(stderr, "list does not accept arguments")
		}
		all, err := store.List()
		if err != nil {
			fmt.Fprintln(stderr, "note:", err)
			return 1
		}
		if len(all) == 0 {
			fmt.Fprintln(stdout, "No notes found.")
			return 0
		}
		for _, note := range all {
			fmt.Fprintf(stdout, "%-28s  %-36s  %s\n", note.Slug, note.Title, note.UpdatedAt.Local().Format("2006-01-02 15:04"))
		}
		return 0
	case "show":
		if len(args) != 2 {
			return usageError(stderr, "show requires exactly one slug")
		}
		note, err := store.Load(args[1])
		if err != nil {
			if errors.Is(err, notes.ErrNotFound) {
				fmt.Fprintf(stderr, "note: %q does not exist\n", args[1])
			} else {
				fmt.Fprintln(stderr, "note:", err)
			}
			return 1
		}
		output, err := mdrender.Note(note, 100)
		if err != nil {
			fmt.Fprintln(stderr, "note:", err)
			return 1
		}
		fmt.Fprint(stdout, output)
		return 0
	default:
		return usageError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
}

func usageError(stderr io.Writer, message string) int {
	fmt.Fprintln(stderr, "note:", message)
	fmt.Fprintln(stderr, "Run 'note help' to see the available commands.")
	return 2
}
