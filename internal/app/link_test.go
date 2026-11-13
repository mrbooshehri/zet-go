package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrbooshehri/zet-go/internal/config"
	"github.com/mrbooshehri/zet-go/internal/notes"
)

func TestLinkNotesPreservesExistingContentAndAvoidsDuplicates(t *testing.T) {
	root := t.TempDir()
	source := "20261113102030"
	target := "20261113103512"
	original := "# Source\n\nBody\n\n## Related\n\n## Sources\n\nSource text\n"
	for id, body := range map[string]string{source: original, target: "# Target\n"} {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var c config.Config
	c.Notes.Directory = root
	c.Links.MarkdownLinks = true
	a := App{Config: c}
	if err := a.LinkNotes(source, target); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, source, "README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), original) || !strings.Contains(string(data), "["+target+"](../"+target+"/README.md)") {
		t.Fatalf("unexpected content: %q", data)
	}
	all, err := a.Notes()
	if err != nil {
		t.Fatal(err)
	}
	if g := notes.BuildGraph(all); len(g.Outgoing[source]) != 1 || len(g.Incoming[target]) != 1 {
		t.Fatalf("link not discoverable: %#v", g)
	}
	if err := a.LinkNotes(source, target); err == nil {
		t.Fatal("duplicate link should fail")
	}
	again, _ := os.ReadFile(path)
	if string(again) != string(data) {
		t.Fatal("duplicate changed note")
	}
}

func TestLinkNotesRejectsInvalidTargetsAndSymlinks(t *testing.T) {
	root := t.TempDir()
	source := "20261113102030"
	dir := filepath.Join(root, source)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "README.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	var c config.Config
	c.Notes.Directory = root
	c.Links.MarkdownLinks = true
	a := App{Config: c}
	for _, target := range []string{"../bad", source, "20261113103512"} {
		if err := a.LinkNotes(source, target); err == nil {
			t.Fatalf("accepted %q", target)
		}
	}
	data, err := os.ReadFile(outside)
	if err != nil || string(data) != "private" {
		t.Fatalf("outside changed: %s %v", data, err)
	}
}

func TestLinkNotesUsesFinalRelatedSectionAndCRLF(t *testing.T) {
	root := t.TempDir()
	from, to := "20261113102030", "20261113103512"
	original := "# Source\r\n\r\n## Related\r\n\r\n"
	for id, body := range map[string]string{from: original, to: "# Target\r\n"} {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var c config.Config
	c.Notes.Directory = root
	c.Links.MarkdownLinks = true
	a := App{Config: c}
	if err := a.LinkNotes(from, to); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, from, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Count(body, "## Related") != 1 || !strings.HasPrefix(body, original) || strings.Contains(strings.ReplaceAll(body, "\r\n", ""), "\n") {
		t.Fatalf("unexpected CRLF content: %q", body)
	}
}

func TestLinkNotesLeavesSourceUnchangedOnInvalidRequest(t *testing.T) {
	root := t.TempDir()
	from, to := "20261113102030", "20261113103512"
	original := "# Source\n\n[[20261113103512]]\n"
	for id, body := range map[string]string{from: original, to: "# Target\n"} {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var c config.Config
	c.Notes.Directory = root
	c.Links.MarkdownLinks = true
	a := App{Config: c}
	for _, target := range []string{to, from, "20261113111111", "../outside"} {
		if err := a.LinkNotes(from, target); err == nil {
			t.Fatalf("expected error for %q", target)
		}
		data, err := os.ReadFile(filepath.Join(root, from, "README.md"))
		if err != nil || string(data) != original {
			t.Fatalf("source changed for %q: %q %v", target, data, err)
		}
	}
}

func TestLinkNotesRejectsUnclosedCodeFence(t *testing.T) {
	root := t.TempDir()
	from, to := "20261113102030", "20261113103512"
	original := "# Source\n\n```\nunclosed\n"
	for id, body := range map[string]string{from: original, to: "# Target\n"} {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var c config.Config
	c.Notes.Directory = root
	c.Links.MarkdownLinks = true
	if err := (App{Config: c}).LinkNotes(from, to); err == nil {
		t.Fatal("expected error for unclosed code fence")
	}
	data, err := os.ReadFile(filepath.Join(root, from, "README.md"))
	if err != nil || string(data) != original {
		t.Fatalf("source changed: %q %v", data, err)
	}
}
