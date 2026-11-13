package app

import (
	"errors"
	"fmt"
	"github.com/mattn/go-shellwords"
	"github.com/mrbooshehri/zet-go/internal/config"
	"github.com/mrbooshehri/zet-go/internal/notes"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type App struct {
	Config            config.Config
	ConfigPath        string
	DirectoryOverride string
	EditorOverride    string
}

func (a App) Notes() ([]notes.Note, error) {
	all, err := notes.Load(a.Config.Notes.Directory)
	if errors.Is(err, os.ErrNotExist) {
		return []notes.Note{}, nil
	}
	return all, err
}
func (a App) Graph(all []notes.Note) notes.Graph {
	return notes.BuildGraphWithOptions(all, a.Config.Links.WikiLinks, a.Config.Links.MarkdownLinks)
}
func (a App) Find(id string) (notes.Note, error) {
	if !notes.ValidID(id) {
		return notes.Note{}, fmt.Errorf("invalid note ID %q", id)
	}
	all, err := a.Notes()
	if err != nil {
		return notes.Note{}, err
	}
	for _, n := range all {
		if n.ID == id {
			return n, nil
		}
	}
	return notes.Note{}, fmt.Errorf("note %s not found", id)
}
func (a App) Editor(path string) (*exec.Cmd, error) {
	args, err := shellwords.Parse(a.Config.Editor.Command)
	if err != nil || len(args) == 0 {
		return nil, fmt.Errorf("invalid editor command %q", a.Config.Editor.Command)
	}
	args = append(args, path)
	return exec.Command(args[0], args[1:]...), nil
}
func (a App) RunEditor(path string) error {
	cmd, err := a.Editor(path)
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("editor: %w", err)
	}
	return nil
}
func (a App) NewNote() (notes.CreatedNote, error) {
	if err := os.MkdirAll(a.Config.Notes.Directory, 0700); err != nil {
		return notes.CreatedNote{}, err
	}
	return notes.CreateWithTemplate(a.Config.Notes.Directory, time.Now())
}
func (a App) FinishNew(n notes.CreatedNote, editorErr error) error {
	if editorErr != nil {
		return fmt.Errorf("editor failed; new note preserved at %s: %w", n.Path, editorErr)
	}
	removed, err := notes.RemoveEmptyNewNote(a.Config.Notes.Directory, n)
	if err != nil {
		return err
	}
	if removed {
		return nil
	}
	return a.Commit(n.ID, "Add note "+n.ID)
}
func (a App) Commit(id, message string) error {
	if !a.Config.Git.Enabled || !a.Config.Git.AutoCommit {
		return nil
	}
	if !notes.ValidID(id) {
		return fmt.Errorf("invalid ID")
	}
	rel := filepath.Join(id, "README.md")
	cmd := exec.Command("git", "-C", a.Config.Notes.Directory, "diff", "--cached", "--quiet", "--", rel)
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
			return fmt.Errorf("note %s already has staged changes; commit it manually", id)
		}
		return fmt.Errorf("inspect staged changes: %w", err)
	}
	cmd = exec.Command("git", "-C", a.Config.Notes.Directory, "add", "--", rel)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	cmd = exec.Command("git", "-C", a.Config.Notes.Directory, "commit", "--only", "-m", message, "--", rel)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	if a.Config.Git.AutoPush {
		cmd = exec.Command("git", "-C", a.Config.Notes.Directory, "push")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("local commit succeeded, but git push failed: %w: %s", err, out)
		}
	}
	return nil
}
