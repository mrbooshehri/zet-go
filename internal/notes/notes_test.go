package notes

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTagsReadWithoutChangingNote(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "20261113102030")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: permanent\ntags: [go, ' knowledge ', go, Go]\n---\n# Note\n"
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	all, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	n := all[0]
	if n.Type != Permanent || !reflect.DeepEqual(n.Tags, []string{"go", "knowledge", "Go"}) || n.MetadataWarning != "" {
		t.Fatalf("unexpected metadata: %#v", n)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != body {
		t.Fatal("note was modified")
	}
}

func TestSourcesReadWithoutChangingNote(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "20261113102030")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: literature\ntags: [reading]\nsources:\n  - ' https://example.com/paper '\n  - 'Book, chapter 3'\n---\n# Note\n"
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	all, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	n := all[0]
	if n.Type != Literature || !reflect.DeepEqual(n.Tags, []string{"reading"}) || !reflect.DeepEqual(n.Sources, []string{"https://example.com/paper", "Book, chapter 3"}) || n.MetadataWarning != "" {
		t.Fatalf("unexpected metadata: %#v", n)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != body {
		t.Fatal("note was modified")
	}
}

func TestInvalidSourcesWarnWithoutLosingOtherMetadata(t *testing.T) {
	for _, value := range []string{"sources: not-a-list", "sources: [valid, '', 42]"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "20261113102030")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\ntype: literature\ntags: [reading]\n"+value+"\n---\n# Note\n"), 0600); err != nil {
				t.Fatal(err)
			}
			all, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			n := all[0]
			if n.Type != Literature || !reflect.DeepEqual(n.Tags, []string{"reading"}) || !strings.Contains(n.MetadataWarning, "sources") {
				t.Fatalf("invalid sources: %#v", n)
			}
		})
	}
}

func TestInvalidTagsWarnWithoutLosingType(t *testing.T) {
	for _, tags := range []string{"tags: not-a-list", "tags: [valid, '', 42]"} {
		t.Run(tags, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "20261113102030")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("---\ntype: fleeting\n"+tags+"\n---\n# Note\n"), 0600); err != nil {
				t.Fatal(err)
			}
			all, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			n := all[0]
			if n.Type != Fleeting || n.Title != "Note" || !strings.Contains(n.MetadataWarning, "tags") {
				t.Fatalf("invalid tags: %#v", n)
			}
		})
	}
}

func TestReadLegacyNotesAndGraph(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"20261113102030": "# First\n\nSee [second](../20261113103512/README.md) and [[20261113111111]].\n",
		"20261113103512": "# Second\n\nBody text\n",
		"20261113104845": "Untitled body\n",
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
	all, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].Title != "First" || all[2].Title != "<untitled>" {
		t.Fatalf("unexpected notes: %#v", all)
	}
	graph := BuildGraph(all)
	if len(graph.Outgoing[all[0].ID]) != 2 || len(graph.Incoming[all[1].ID]) != 1 || len(graph.Broken) != 1 {
		t.Fatalf("unexpected graph: %#v", graph)
	}
	if len(graph.Orphans) != 1 || graph.Orphans[0] != all[2].ID {
		t.Fatalf("unexpected orphans: %#v", graph.Orphans)
	}
	hits := Search(all, "BODY")
	if len(hits) != 2 {
		t.Fatalf("search results: %#v", hits)
	}
	content, err := os.ReadFile(filepath.Join(root, all[0].ID, "README.md"))
	if err != nil || string(content) != files[all[0].ID] {
		t.Fatal("read changed file")
	}
}

func TestRejectInvalidID(t *testing.T) {
	for _, id := range []string{"../outside", "20261113102030/../x", "abc"} {
		if ValidID(id) {
			t.Fatalf("accepted %q", id)
		}
	}
}

func TestIgnoresSymlinkedNotesAndNonNotes(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "README.md"), []byte("# private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "20261113102030")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "20261113103512"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "README.md"), filepath.Join(root, "20261113103512", "README.md")); err != nil {
		t.Fatal(err)
	}
	all, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("loaded symlinked notes: %#v", all)
	}
}

func TestSearchRanksIDThenTitleThenBody(t *testing.T) {
	all := []Note{{ID: "20261113102030", Title: "other", Body: "needle"}, {ID: "20261113103512", Title: "Needle title"}, {ID: "needle", Title: "other"}}
	hits := Search(all, "needle")
	if len(hits) != 3 || hits[0].ID != "needle" || hits[1].ID != "20261113103512" {
		t.Fatalf("unexpected ranking: %#v", hits)
	}
}

func TestCreateAllocatesUniqueDirectories(t *testing.T) {
	root := t.TempDir()
	when := time.Date(2026, 11, 13, 10, 20, 30, 0, time.UTC)
	a, err := Create(root, when)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Create(root, when)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "20261113102030" || b.ID != "20261113102031" {
		t.Fatalf("unexpected IDs: %s %s", a.ID, b.ID)
	}
	if _, err := os.Stat(a.Path); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveEmptyNewNotePreservesContentAndOtherFiles(t *testing.T) {
	root := t.TempDir()
	n, err := Create(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(n.Path, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveEmptyNewNote(root, n); err != nil || removed {
		t.Fatalf("removed content: %v %v", removed, err)
	}
	if err := os.WriteFile(n.Path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, n.ID, "attachment"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if removed, err := RemoveEmptyNewNote(root, n); err != nil || removed {
		t.Fatalf("removed attachment: %v %v", removed, err)
	}
	if _, err := os.Stat(filepath.Join(root, n.ID, "attachment")); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveEmptyNewNoteRemovesOnlyCreatedEmptyNote(t *testing.T) {
	root := t.TempDir()
	n, err := Create(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	removed, err := RemoveEmptyNewNote(root, n)
	if err != nil || !removed {
		t.Fatalf("cleanup: %v %v", removed, err)
	}
	if _, err := os.Stat(filepath.Dir(n.Path)); !os.IsNotExist(err) {
		t.Fatalf("directory still present: %v", err)
	}
}

func TestRemoveEmptyNewNoteRejectsReplacement(t *testing.T) {
	root := t.TempDir()
	n, err := Create(root, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(n.Path, n.Path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(n.Path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	removed, err := RemoveEmptyNewNote(root, n)
	if err != nil || removed {
		t.Fatalf("removed replacement: %v %v", removed, err)
	}
}

func TestTemplateCleanupPreservesSameLengthEdits(t *testing.T) {
	root := t.TempDir()
	n, err := CreateWithTemplate(root, time.Date(2026, 11, 13, 10, 20, 30, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(string(data), "# ")
	if i < 0 {
		t.Fatal("heading missing")
	}
	data[i+1] = 'X'
	if err := os.WriteFile(n.Path, data, 0600); err != nil {
		t.Fatal(err)
	}
	removed, err := RemoveEmptyNewNote(root, n)
	if err != nil || removed {
		t.Fatalf("modified file removed: %v %v", removed, err)
	}
	got, err := os.ReadFile(n.Path)
	if err != nil || string(got) != string(data) {
		t.Fatalf("modified content lost: %q %v", got, err)
	}
}

func TestGraphIgnoresCodeAndFindsReferenceLinks(t *testing.T) {
	all := []Note{{ID: "20261113102030", Body: "# test\n\n`[[20261113111111]]`\n\n```\n[fake](../20261113111111/README.md)\n```\n\n[real][target]\n\n[target]: ../20261113103512/README.md\n"}, {ID: "20261113103512"}}
	g := BuildGraph(all)
	if len(g.Outgoing[all[0].ID]) != 1 || g.Outgoing[all[0].ID][0].To != all[1].ID || len(g.Broken) != 0 {
		t.Fatalf("incorrect graph: %#v", g)
	}
}

func TestSearchOptions(t *testing.T) {
	all := []Note{{ID: "20261113102030", Title: "Needle"}, {ID: "20261113102031", Title: "needle"}}
	if got := SearchWithOptions(all, "needle", true, 1); len(got) != 1 || got[0].ID != all[1].ID {
		t.Fatalf("search options: %#v", got)
	}
}

func TestSearchSnippet(t *testing.T) {
	for _, tc := range []struct {
		name, body, query, want string
		caseSensitive           bool
		limit                   int
	}{
		{"unicode and case", "متن سلام دنیا", "سلام", "متن سلام دنیا", false, 40},
		{"context", strings.Repeat("a", 20) + " needle " + strings.Repeat("b", 20), "needle", "…aaaa needle bbbb…", false, 16},
		{"whitespace and controls", "Before\nneedle\t\x1b[31m after", "NEEDLE", "Before needle [31m after", false, 80},
		{"case sensitive miss", "Text Needle", "needle", "", true, 40},
		{"case sensitive hit", "Text Needle", "Needle", "Text Needle", true, 40},
		{"match at end", "abcdefghijNEEDLE", "needle", "…ijNEEDLE", false, 8},
		{"zero limit", "needle", "needle", "", false, 0},
		{"metadata and heading excluded", "---\ntype: permanent\n---\n# Needle\n\nOther text", "needle", "", false, 40},
		{"body after metadata", "---\ntype: permanent\n---\n# Title\n\nNeedle", "needle", "Needle", false, 40},
		{"no query", "Anything", "", "", false, 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SearchSnippet(tc.body, tc.query, tc.caseSensitive, tc.limit); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestOptionalNoteTypesPreserveMarkdown(t *testing.T) {
	tests := []struct {
		name, body, wantType, wantTitle string
		warning                         bool
	}{
		{"legacy", "# Legacy\n\nText\n", "", "Legacy", false},
		{"empty title", "# \n\nText\n", "", "<untitled>", false},
		{"fleeting", "---\ntype: fleeting\n---\n# Idea\n\nText\n", "fleeting", "Idea", false},
		{"literature CRLF", "---\r\ntype: literature\r\nother: value\r\n---\r\n\r\n# Book\r\n", "literature", "Book", false},
		{"permanent", "---\ntype: permanent\n---\n# Insight\n", "permanent", "Insight", false},
		{"unknown type", "---\ntype: invented\n---\n# Unknown\n", "", "Unknown", true},
		{"malformed yaml", "---\ntype: [broken\n---\n# Still readable\n", "", "<untitled>", true},
		{"missing closing delimiter", "---\ntype: fleeting\n# Plain content\n", "", "<untitled>", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "20261113102030")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "README.md")
			if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			all, err := Load(root)
			if err != nil || len(all) != 1 {
				t.Fatalf("load: %v %#v", err, all)
			}
			n := all[0]
			if string(n.Type) != tc.wantType || n.Title != tc.wantTitle || (n.MetadataWarning != "") != tc.warning {
				t.Fatalf("unexpected metadata: %#v", n)
			}
			if n.Body != tc.body {
				t.Fatalf("body changed: %q", n.Body)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != tc.body {
				t.Fatalf("file changed: %q %v", data, err)
			}
		})
	}
}

func TestFrontmatterDoesNotCreateGraphLinks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "20261113102030")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: fleeting\nsource: '[[20261113111111]]'\n---\n# Note\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	all, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if g := BuildGraph(all); len(g.Broken) != 0 {
		t.Fatalf("frontmatter created links: %#v", g.Broken)
	}
}
