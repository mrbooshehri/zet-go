package app

import (
	"errors"
	"github.com/mrbooshehri/zet-go/internal/config"
	"github.com/mrbooshehri/zet-go/internal/notes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewNoteTemplateHasAutomaticCreationTime(t *testing.T) {
	var c config.Config
	c.Notes.Directory = t.TempDir()
	a := App{Config: c}
	before := time.Now().UTC().Truncate(time.Second)
	n, err := a.NewNote()
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC().Add(time.Second)
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	lines := strings.Split(text, "\n")
	if len(lines) < 7 || lines[0] != "---" || !strings.HasPrefix(lines[1], "created: ") || lines[2] != "tags: []" || lines[3] != "sources: []" || lines[4] != "---" || lines[5] != "# " || !strings.Contains(text, "\n## Tags\n") || !strings.Contains(text, "\n## Related\n") || !strings.Contains(text, "\n## Sources\n") {
		t.Fatalf("unexpected template: %q", text)
	}
	created, err := time.Parse(time.RFC3339, strings.Trim(strings.TrimPrefix(lines[1], "created: "), "\""))
	if err != nil || created.Before(before) || created.After(after) {
		t.Fatalf("invalid generated timestamp %q: %v", lines[1], err)
	}
	if err := a.FinishNew(n, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(n.Path); !os.IsNotExist(err) {
		t.Fatalf("untouched template not removed: %v", err)
	}
}

func TestNewNoteTemplatePreservesChangesAndFailures(t *testing.T) {
	var c config.Config
	c.Notes.Directory = t.TempDir()
	a := App{Config: c}
	n, err := a.NewNote()
	if err != nil {
		t.Fatal(err)
	}
	initial, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	updated := append(initial, []byte("An idea\n")...)
	if err := os.WriteFile(n.Path, updated, 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.FinishNew(n, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(n.Path)
	if err != nil || string(got) != string(updated) {
		t.Fatalf("modified note lost: %q %v", got, err)
	}
	failed, err := a.NewNote()
	if err != nil {
		t.Fatal(err)
	}
	editorErr := errors.New("editor failed")
	if err := a.FinishNew(failed, editorErr); !errors.Is(err, editorErr) {
		t.Fatalf("editor error lost: %v", err)
	}
	if _, err := os.Stat(failed.Path); err != nil {
		t.Fatalf("failed note removed: %v", err)
	}
}

func TestTemplateMetadataAndRelatedLinksAreIndexed(t *testing.T) {
	var c config.Config
	c.Notes.Directory = t.TempDir()
	a := App{Config: c}
	n, err := a.NewNote()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(string(data), "tags: []", "tags: [go]", 1)
	body = strings.Replace(body, "sources: []", "sources: [Book]", 1)
	body = strings.Replace(body, "## Related\n", "## Related\n\n[Other](../20261113103512/README.md)\n", 1)
	if err := os.WriteFile(n.Path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	all, err := a.Notes()
	if err != nil || len(all) != 1 {
		t.Fatalf("read note: %v %#v", err, all)
	}
	if len(all[0].Tags) != 1 || all[0].Tags[0] != "go" || len(all[0].Sources) != 1 || all[0].Sources[0] != "Book" {
		t.Fatalf("metadata: %#v", all[0])
	}
	graph := notes.BuildGraph(all)
	if len(graph.Outgoing[n.ID]) != 1 || graph.Outgoing[n.ID][0].To != "20261113103512" {
		t.Fatalf("related link: %#v", graph.Outgoing)
	}
}

func TestMissingNotesDirectoryStartsEmpty(t *testing.T) {
	var c config.Config
	c.Notes.Directory = filepath.Join(t.TempDir(), "new")
	all, err := (App{Config: c}).Notes()
	if err != nil || len(all) != 0 {
		t.Fatalf("%v %v", all, err)
	}
}

func TestAutoPushFailureReportsCommittedNoteAndPreservesIndex(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	root := t.TempDir()
	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "unrelated"), []byte("staged"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "unrelated")
	id := "20261113102030"
	if err := os.Mkdir(filepath.Join(root, id), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "README.md"), []byte("# Saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var c config.Config
	c.Notes.Directory = root
	c.Git.Enabled = true
	c.Git.AutoCommit = true
	c.Git.AutoPush = true
	err := (App{Config: c}).Commit(id, "add note")
	if err == nil || !strings.Contains(err.Error(), "local commit succeeded") {
		t.Fatalf("push error must explain local commit: %v", err)
	}
	if got := git(t, root, "show", "HEAD:"+id+"/README.md"); got != "# Saved\n" {
		t.Fatalf("local commit missing: %q", got)
	}
	if got := git(t, root, "diff", "--cached", "--name-only"); got != "unrelated\n" {
		t.Fatalf("staged changes lost: %q", got)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v %s", args, e, out)
	}
	return string(out)
}

func TestCommitPreservesOtherStagedChanges(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "unrelated"), []byte("staged"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "unrelated")
	id := "20261113102030"
	if err := os.Mkdir(filepath.Join(root, id), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "README.md"), []byte("# Note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var c config.Config
	c.Notes.Directory = root
	c.Git.Enabled = true
	c.Git.AutoCommit = true
	if err := (App{Config: c}).Commit(id, "note"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, root, "diff", "--cached", "--name-only"); got != "unrelated\n" {
		t.Fatalf("staged changes lost: %q", got)
	}
}

func TestCommitRefusesAlreadyStagedNote(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init")
	git(t, root, "config", "user.email", "test@example.com")
	git(t, root, "config", "user.name", "Test")
	id := "20261113102030"
	if err := os.Mkdir(filepath.Join(root, id), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, id, "README.md")
	if err := os.WriteFile(path, []byte("staged"), 0600); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", id)
	if err := os.WriteFile(path, []byte("working"), 0600); err != nil {
		t.Fatal(err)
	}
	var c config.Config
	c.Notes.Directory = root
	c.Git.Enabled = true
	c.Git.AutoCommit = true
	if err := (App{Config: c}).Commit(id, "note"); err == nil {
		t.Fatal("expected refusal to overwrite staged note")
	}
	if got := git(t, root, "show", ":"+id+"/README.md"); got != "staged" {
		t.Fatalf("staged content changed: %q", got)
	}
}
