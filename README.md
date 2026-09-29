# note

`note` is a small terminal application for creating and browsing Markdown
notes. It is written in Go using Bubble Tea, Bubbles, and Lip Gloss.

Notes are stored as readable files in `~/Documents/notes`. Deleting a note from
the application does not remove it permanently: the file is moved to
`~/Documents/notes/.trash`.

## Installation

Go 1.24 or later is required.

```sh
make test
make install
```

`make install` installs `note` in `~/.local/bin`, which must be included in
`PATH`. To compile without installing, run `make build`; the executable will be
created at `bin/note`.

## Commands

```text
note                 Open the interactive note list
note new [title]     Create a note in the editor
note list            List each note's slug, title, and modification time
note show <slug>     Render a note in the terminal
note help            Show command help
```

## TUI key bindings

In the note list:

- `↑`/`↓` or `k`/`j`: move through the list.
- `Enter`: read the selected note.
- `n`: create a note.
- `e`: edit the selected note.
- `d`: move the selected note to trash after confirmation.
- `/`: search titles and content.
- `q`: quit.

In the editor, `Tab` switches between the title and body, `Ctrl+S` saves, and
`Esc` returns to the list. If there are pending changes, the application asks
whether they should be saved or discarded.

In the reader, use `↑`/`↓`, `j`/`k`, `PgUp`/`PgDown`, or the mouse wheel to
scroll through long notes.

## Note format

Each file uses a stable slug as its name and YAML front matter:

```markdown
---
title: Example
created_at: 2026-09-28T12:00:00Z
updated_at: 2026-09-28T12:00:00Z
---

Note content.
```

Markdown files copied manually into the notes directory also appear in the
list. When a file has no valid front matter, its filename is used as the title.
To recover a deleted note, move its file from `.trash` back into the `notes`
directory.

## Development

Run the checks with:

```sh
go test ./...
go vet ./...
```
