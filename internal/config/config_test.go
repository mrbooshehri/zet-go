package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsDraftSavesValidChangesAndPreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := "# personal comment\ntui:\n  preview: true\n"
	if err := os.WriteFile(path, []byte(original), 0640); err != nil {
		t.Fatal(err)
	}
	draft, err := BeginEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(original, "preview: true", "preview: false", 1)
	if err := os.WriteFile(draft.Path, []byte(updated), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := draft.Finish(nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.TUI.Preview {
		t.Fatal("preview unchanged")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != updated {
		t.Fatalf("saved config: %q %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("permissions: %v %v", info, err)
	}
}

func TestSettingsDraftRejectsInvalidOrConcurrentChanges(t *testing.T) {
	for _, tc := range []struct {
		name, draftText, external string
		editorErr                 error
	}{{"invalid", "tui: [bad", "", nil}, {"changed", "tui:\n  preview: false\n", "# external\n", nil}, {"editor failed", "tui:\n  preview: false\n", "", errors.New("editor error")}} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			original := "tui:\n  preview: true\n"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			draft, err := BeginEdit(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(draft.Path, []byte(tc.draftText), 0600); err != nil {
				t.Fatal(err)
			}
			expected := original
			if tc.external != "" {
				expected = tc.external
				if err := os.WriteFile(path, []byte(expected), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err = draft.Finish(tc.editorErr, "", "")
			if err == nil || !strings.Contains(err.Error(), draft.Path) {
				t.Fatalf("expected error with draft path: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != expected {
				t.Fatalf("original lost: %q %v", data, err)
			}
			if _, err := os.Stat(draft.Path); err != nil {
				t.Fatal("draft not preserved")
			}
		})
	}
}

func TestSettingsDraftRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("tui: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.Symlink(target, path); err != nil {
		t.Skip(err)
	}
	if _, err := BeginEdit(path); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestUnchangedNewSettingsDraftDoesNotCreateConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	draft, err := BeginEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = draft.Finish(nil, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected config: %v", err)
	}
	if _, err = os.Stat(draft.Path); !os.IsNotExist(err) {
		t.Fatalf("draft not removed: %v", err)
	}
}

func TestConfigurationPrecedence(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "config.yaml")
	if err := os.WriteFile(path, []byte("notes:\n  directory: /file\n  filename: README.md\n  id_format: timestamp\ntui:\n  theme: auto\n  preview: true\n  layout: adaptive\nsearch:\n  case_sensitive: false\n  max_results: 100\nlinks:\n  wiki_links: true\n  markdown_links: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KN", "/kn")
	t.Setenv("ZETDIR", "/zetdir")
	c, err := Load(path, "/flag", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Notes.Directory != "/flag" {
		t.Fatal(c.Notes.Directory)
	}
	c, err = Load(path, "", "")
	if err != nil || c.Notes.Directory != "/zetdir" {
		t.Fatalf("%+v %v", c, err)
	}
}
