package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShowJSONIncludesNoteType(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\ntype: permanent\n---\n# Insight\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--json", "show", id})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "permanent" || got["title"] != "Insight" {
		t.Fatalf("unexpected JSON: %s", out.String())
	}
}

func TestTagsCommandListsUniqueSortedTags(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"20261113102030": "---\ntags: [go, knowledge, go]\n---\n# First\n", "20261113103512": "---\ntags: [zettel, go]\n---\n# Second\n"}
	for id, body := range files {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ZETDIR", root)
	for _, tc := range []struct {
		args []string
		want string
	}{{[]string{"tags"}, "go\nknowledge\nzettel\n"}, {[]string{"--json", "tags"}, "[\n  \"go\",\n  \"knowledge\",\n  \"zettel\"\n]\n"}} {
		var out bytes.Buffer
		cmd := New(&out, &out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != tc.want {
			t.Fatalf("%v: got %q", tc.args, out.String())
		}
	}
}

func TestSearchSnippetsSkipTitleOnlyMatches(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Title\n\nBody\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"search", "Title", "--snippets"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != id+" Title\n" {
		t.Fatalf("unexpected snippet: %q", out.String())
	}
}

func TestSearchTypeAndTagFiltersBeforeResultLimit(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"20261113102030": "---\ntype: fleeting\ntags: [go]\n---\n# Needle one\n",
		"20261113103512": "---\ntype: permanent\ntags: [go, reading]\n---\n# Needle two\n",
		"20261113104845": "---\ntype: permanent\ntags: [Go]\n---\n# Needle three\n",
	}
	for id, body := range files {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("search:\n  max_results: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	for _, tc := range []struct {
		args             []string
		contains, absent string
	}{
		{[]string{"search", "needle", "--type", "permanent", "--tag", "go"}, "20261113103512", "20261113102030"},
		{[]string{"q", "needle", "--tag", "Go"}, "20261113104845", "20261113103512"},
		{[]string{"--json", "search", "needle", "--type", "permanent", "--tag", "reading"}, "20261113103512", "20261113104845"},
	} {
		var out bytes.Buffer
		cmd := New(&out, &out)
		cmd.SetArgs(append([]string{"--config-file", configPath}, tc.args...))
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.contains) || strings.Contains(out.String(), tc.absent) {
			t.Fatalf("%v: %s", tc.args, out.String())
		}
	}
}

func TestSearchRejectsUnknownNoteType(t *testing.T) {
	t.Setenv("ZETDIR", t.TempDir())
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"search", "word", "--type", "unknown"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown note type") {
		t.Fatalf("expected invalid type, got %v", err)
	}
}

func TestSearchSnippetsOptInAndJSONUnchanged(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	body := "# A title\n\nSome searchable content.\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	for _, tc := range []struct {
		args []string
		want string
	}{{[]string{"search", "searchable", "--snippets"}, "  Some searchable content.\n"}, {[]string{"search", "searchable"}, id + " A title\n"}} {
		var out bytes.Buffer
		cmd := New(&out, &out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%v: %q", tc.args, out.String())
		}
	}
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--json", "search", "searchable", "--snippets"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got []struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || len(got) != 1 || got[0].Body != body {
		t.Fatalf("JSON changed: %s %v", out.String(), err)
	}
}

func TestSearchFilterWithNoMatchesReturnsEmptyJSON(t *testing.T) {
	t.Setenv("ZETDIR", t.TempDir())
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--json", "search", "word", "--tag", "missing"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "[]\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestSourcesCommand(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\nsources: ['Book', 'https://example.com']\n---\n# Note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	for _, tc := range []struct {
		args []string
		want string
	}{{[]string{"sources", id}, "Book\nhttps://example.com\n"}, {[]string{"--json", "sources", id}, "[\n  \"Book\",\n  \"https://example.com\"\n]\n"}} {
		var out bytes.Buffer
		cmd := New(&out, &out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != tc.want {
			t.Fatalf("%v: got %q", tc.args, out.String())
		}
	}
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"sources", "../outside"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid ID error")
	}
}

func TestLinkCommandCreatesRelationship(t *testing.T) {
	root := t.TempDir()
	from, to := "20261113102030", "20261113103512"
	for id, body := range map[string]string{from: "# First\n", to: "# Second\n"} {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"link", from, to})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, from, "README.md"))
	if err != nil || !strings.Contains(string(content), "["+to+"](../"+to+"/README.md)") {
		t.Fatalf("missing link: %s %v", content, err)
	}
}

func TestSourcesEmptyAndInvalidMetadata(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{"20261113102030": "# Untyped\n", "20261113103512": "---\nsources: [good, 42]\n---\n# Invalid\n"}
	for id, body := range files {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--json", "sources", "20261113102030"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "[]\n" {
		t.Fatalf("empty sources: %q", out.String())
	}
	out.Reset()
	cmd = New(&out, &out)
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "20261113103512") || !strings.Contains(out.String(), "sources must be") {
		t.Fatalf("warning missing: %q", out.String())
	}
	out.Reset()
	cmd = New(&out, &out)
	cmd.SetArgs([]string{"sources", "20261113999999"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing note error")
	}
}

func TestTagsAppearInNoteJSONAndInvalidTagsInCheck(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\ntype: permanent\ntags: [go, 42]\n---\n# Note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--json", "show", id})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Tags []string `json:"tags"`
		Type string   `json:"type"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "go" || got.Type != "permanent" {
		t.Fatalf("unexpected JSON: %s", out.String())
	}
	out.Reset()
	cmd = New(&out, &out)
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), id) || !strings.Contains(out.String(), "tags must be") {
		t.Fatalf("warning missing: %s", out.String())
	}
}

func TestCheckReportsInvalidNoteType(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	dir := filepath.Join(root, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\ntype: invalid\n---\n# Note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"check"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), id) || !strings.Contains(out.String(), "unknown note type") {
		t.Fatalf("missing warning: %q", out.String())
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("input failure") }

func TestURLEncodeArgumentsAndInput(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		input, want string
	}{
		{"arguments", []string{"urlencode", "hello", "world+"}, "ignored", "hello%20world%2B\n"},
		{"stdin", []string{"urlencode"}, "hello world\nسلام\n\nlast line", "hello%20world\n%D8%B3%D9%84%D8%A7%D9%85\n\nlast%20line\n"},
		{"empty stdin", []string{"urlencode"}, "", ""},
		{"long line", []string{"urlencode"}, strings.Repeat("a", 70000), strings.Repeat("a", 70000) + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := New(&out, &out)
			cmd.SetIn(strings.NewReader(tc.input))
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("unexpected output: got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestURLEncodeReportsInputFailure(t *testing.T) {
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetIn(failingReader{})
	cmd.SetArgs([]string{"urlencode"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "input failure") {
		t.Fatalf("expected input error, got %v", err)
	}
}

func TestReadCommandsUseLegacyNotes(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	if err := os.Mkdir(filepath.Join(root, id), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "README.md"), []byte("# Hello\n\nA needle.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	for _, tc := range []struct {
		args []string
		want string
	}{{[]string{"title", id}, "Hello"}, {[]string{"search", "needle"}, id}, {[]string{"ids"}, id}, {[]string{"show", id}, "A needle."}} {
		var out bytes.Buffer
		cmd := New(&out, &out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("%v: %q", tc.args, out.String())
		}
	}
}

func TestInvalidNoteIDRejected(t *testing.T) {
	t.Setenv("ZETDIR", t.TempDir())
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"show", "../secret"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid ID error")
	}
}

func TestPickEmptyNotesDoesNotWriteOutput(t *testing.T) {
	t.Setenv("ZETDIR", t.TempDir())
	var out, errout bytes.Buffer
	cmd := New(&out, &errout)
	cmd.SetArgs([]string{"pick"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no notes") {
		t.Fatalf("expected empty picker error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", out.String())
	}
}

func TestAddUsesEditorAndPersistsNote(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("shell fixture")
	}
	root := filepath.Join(t.TempDir(), "notes")
	editor := filepath.Join(t.TempDir(), "editor")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf '# Created\\n\\nBody\\n' > \"$1\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--editor", editor, "add"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("%v %v", entries, err)
	}
	content, err := os.ReadFile(filepath.Join(root, entries[0].Name(), "README.md"))
	if err != nil || !strings.Contains(string(content), "Created") {
		t.Fatalf("%s %v", content, err)
	}
}

func TestAddProvidesTemplateToEditor(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("shell fixture")
	}
	root := t.TempDir()
	editor := filepath.Join(t.TempDir(), "editor")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf 'An idea\\n' >> \"$1\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--editor", editor, "add"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("note missing: %v %v", entries, err)
	}
	body, err := os.ReadFile(filepath.Join(root, entries[0].Name(), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "created: ") || !strings.Contains(string(body), "# \n\n## Tags\n") || !strings.Contains(string(body), "## Related\n") || !strings.Contains(string(body), "## Sources\n\nAn idea\n") {
		t.Fatalf("template not provided: %q", body)
	}
}

func TestAddEditorFailureReportsPreservedPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--editor", filepath.Join(t.TempDir(), "missing-editor"), "add"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected editor failure")
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 1 {
		t.Fatalf("preserved note: %v %v", entries, readErr)
	}
	path := filepath.Join(root, entries[0].Name(), "README.md")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatal(statErr)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error does not identify preserved note %q: %v", path, err)
	}
}

func TestAddEditorFailurePreservesWrittenContent(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("shell fixture")
	}
	root := t.TempDir()
	editor := filepath.Join(t.TempDir(), "editor")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\nprintf 'saved content' > \"$1\"\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{"--editor", editor, "add"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected editor failure")
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil || len(entries) != 1 {
		t.Fatalf("notes: %v %v", entries, readErr)
	}
	path := filepath.Join(root, entries[0].Name(), "README.md")
	content, readErr := os.ReadFile(path)
	if readErr != nil || string(content) != "saved content" {
		t.Fatalf("content %q: %v", content, readErr)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error does not identify preserved note: %v", err)
	}
}

func TestDefaultCommandToPipeListsNotes(t *testing.T) {
	root := t.TempDir()
	id := "20261113102030"
	if err := os.Mkdir(filepath.Join(root, id), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, id, "README.md"), []byte("# Pipe\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZETDIR", root)
	var out bytes.Buffer
	cmd := New(&out, &out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), id) {
		t.Fatal(out.String())
	}
}
