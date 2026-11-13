package app

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mrbooshehri/zet-go/internal/notes"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// LinkNotes appends a human-requested Markdown relationship to the source.
func (a App) LinkNotes(from, to string) error {
	if !notes.ValidID(from) || !notes.ValidID(to) {
		return fmt.Errorf("invalid note ID")
	}
	if from == to {
		return fmt.Errorf("a note cannot link to itself using this action")
	}
	if !a.Config.Links.MarkdownLinks {
		return fmt.Errorf("enable links.markdown_links to create relationships")
	}
	source, err := a.Find(from)
	if err != nil {
		return err
	}
	if _, err = a.Find(to); err != nil {
		return err
	}
	info, err := os.Lstat(source.Path)
	if err != nil {
		return fmt.Errorf("inspect source note: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source note is not a regular file")
	}
	f, err := os.OpenFile(source.Path, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("open source note: %w", err)
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect open note: %w", err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return fmt.Errorf("source note changed while opening; try again")
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read source note: %w", err)
	}
	for _, link := range notes.BuildGraph([]notes.Note{{ID: from, Body: string(data)}}).Outgoing[from] {
		if link.To == to {
			return fmt.Errorf("note %s already links to %s", from, to)
		}
	}
	after, err := f.Stat()
	if err != nil {
		return fmt.Errorf("inspect source note: %w", err)
	}
	if after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return fmt.Errorf("source note changed while reading; try again")
	}
	newline := "\n"
	if strings.Contains(string(data), "\r\n") {
		newline = "\r\n"
	}
	lastHeading := ""
	doc := goldmark.DefaultParser().Parse(text.NewReader(data))
	_ = ast.Walk(doc, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if enter {
			if h, ok := n.(*ast.Heading); ok {
				if h.Level == 2 {
					lastHeading = strings.TrimSpace(string(h.Text(data)))
				} else {
					lastHeading = ""
				}
			}
		}
		return ast.WalkContinue, nil
	})
	var addition strings.Builder
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		addition.WriteString(newline)
	}
	addition.WriteString(newline)
	if !strings.EqualFold(lastHeading, "Related") {
		addition.WriteString("## Related" + newline + newline)
	}
	fmt.Fprintf(&addition, "- [%s](../%s/README.md)%s", to, to, newline)
	valid := false
	candidate := string(data) + addition.String()
	for _, link := range notes.BuildGraph([]notes.Note{{ID: from, Body: candidate}}).Outgoing[from] {
		if link.To == to {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("cannot add a readable relationship at the end of this note; check for an unclosed Markdown block")
	}
	if _, err = f.WriteString(addition.String()); err != nil {
		return fmt.Errorf("append relationship (the note may be partially written): %w", err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("close source note after adding relationship: %w", err)
	}
	return nil
}
