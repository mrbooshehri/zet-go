package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mrbooshehri/zet-go/internal/config"
	"github.com/mrbooshehri/zet-go/internal/notes"
	"github.com/muesli/termenv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSearchInputDoesNotTriggerShortcuts(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "needle"}}, width: 90, height: 24}
	m.filter = ""
	m.searching = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	got := next.(model)
	if got.filter != "n" || !got.searching {
		t.Fatalf("input not captured: %#v", got)
	}
}

func TestPickModeSelectsAndCancels(t *testing.T) {
	m := model{pickMode: true, screen: "Notes", all: []notes.Note{{ID: "20261113102030", Title: "Older"}, {ID: "20261113103512", Title: "Newer"}}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)
	if cmd == nil || got.pickedID != "20261113103512" {
		t.Fatalf("selection: %#v", got)
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got = next.(model)
	if cmd == nil || got.pickedID != "" {
		t.Fatalf("cancellation: %#v", got)
	}
}

func TestPickModeEnterInSearchOnlyLeavesSearch(t *testing.T) {
	m := model{pickMode: true, searching: true, screen: "Notes", all: []notes.Note{{ID: "20261113102030", Title: "Note"}}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(model)
	if got.searching || got.pickedID != "" || cmd != nil {
		t.Fatal("search enter should not select a note")
	}
}

func TestSettingsScreenAndValidatedSave(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(path, []byte("tui:\n  preview: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path, "", "")
	if err != nil {
		t.Fatal(err)
	}
	m := model{screen: "Notes", width: 100, height: 20}
	m.app.Config = c
	m.app.ConfigPath = path
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = next.(model)
	if m.screen != "Settings" || !strings.Contains(ansi.Strip(m.View()), "preview: true") {
		t.Fatalf("settings screen: %s", m.View())
	}
	draft, err := config.BeginEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draft.Path, []byte("tui:\n  preview: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	next, _ = m.Update(settingsDone{draft: draft})
	m = next.(model)
	if m.err != nil || m.app.Config.TUI.Preview {
		t.Fatalf("settings save: %v", m.err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "preview: false") {
		t.Fatalf("config not saved: %s %v", data, err)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(model).screen != "Notes" {
		t.Fatal("settings did not return to notes")
	}
}

func TestSettingsInvalidDraftKeepsOriginalAndReportsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := "tui:\n  preview: true\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	draft, err := config.BeginEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draft.Path, []byte("tui: [bad"), 0600); err != nil {
		t.Fatal(err)
	}
	m := model{screen: "Settings"}
	m.app.ConfigPath = path
	next, _ := m.Update(settingsDone{draft: draft})
	m = next.(model)
	if m.err == nil || !strings.Contains(m.err.Error(), draft.Path) {
		t.Fatalf("draft path missing: %v", m.err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != original {
		t.Fatalf("original changed: %q %v", data, err)
	}
	if _, err := os.Stat(draft.Path); err != nil {
		t.Fatal("draft lost")
	}
}

func TestSearchAcceptsSpacesAndShellEditingKeys(t *testing.T) {
	m := model{screen: "Notes", width: 80, height: 20}
	press := func(msg tea.KeyMsg) { next, _ := m.Update(msg); m = next.(model) }
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("how")})
	press(tea.KeyMsg{Type: tea.KeySpace})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("to")})
	if m.filter != "how to" {
		t.Fatalf("space not inserted: %q", m.filter)
	}
	press(tea.KeyMsg{Type: tea.KeyCtrlW})
	if m.filter != "how " {
		t.Fatalf("ctrl+w: %q", m.filter)
	}
	press(tea.KeyMsg{Type: tea.KeyCtrlA})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("why ")})
	if m.filter != "why how " {
		t.Fatalf("ctrl+a: %q", m.filter)
	}
	press(tea.KeyMsg{Type: tea.KeyCtrlE})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("now")})
	if m.filter != "why how now" {
		t.Fatalf("ctrl+e: %q", m.filter)
	}
	press(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.filter != "" {
		t.Fatalf("ctrl+u: %q", m.filter)
	}
}

func TestSearchEditsUnicodeAndDeletesAfterCursor(t *testing.T) {
	m := model{screen: "Notes", width: 80, height: 20}
	press := func(msg tea.KeyMsg) { next, _ := m.Update(msg); m = next.(model) }
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("سلام")})
	press(tea.KeyMsg{Type: tea.KeySpace})
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("دنیا")})
	press(tea.KeyMsg{Type: tea.KeyCtrlW})
	if m.filter != "سلام " {
		t.Fatalf("unicode word deletion: %q", m.filter)
	}
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("دنیا")})
	press(tea.KeyMsg{Type: tea.KeyCtrlA})
	press(tea.KeyMsg{Type: tea.KeyCtrlK})
	if m.filter != "" {
		t.Fatalf("delete after cursor: %q", m.filter)
	}
}

func TestSelectedRowHasColorAndMarker(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(profile)
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Older"}, {ID: "20261113103512", Title: "Newer"}}, width: 70, height: 20, screen: "Notes"}
	view := m.View()
	if !strings.Contains(view, "\x1b[") || !strings.Contains(ansi.Strip(view), "› 20261113103512") {
		t.Fatalf("selected row lacks color or marker: %q", view)
	}
}

func TestFocusedPanelUsesDistinctBorderColor(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(profile)
	unfocused := panel("Notes", "content", 30, 6, false)
	focused := panel("Notes", "content", 30, 6, true)
	if !strings.Contains(ansi.Strip(focused), "┃") || !strings.Contains(ansi.Strip(unfocused), "│") {
		t.Fatal("focused panel must use a heavier border")
	}
	if lipgloss.Width(focused) != lipgloss.Width(unfocused) || lipgloss.Height(focused) != lipgloss.Height(unfocused) {
		t.Fatal("focus changed panel dimensions")
	}
	m := model{width: 100, height: 20, screen: "Notes", all: []notes.Note{{ID: "20261113102030", Title: "Example"}}}
	m.app.Config.TUI.Preview = true
	before := m.View()
	if strings.Contains(ansi.Strip(before), "Notes •") {
		t.Fatal("focus dot remains")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	after := m.View()
	if strings.Contains(ansi.Strip(after), "Preview •") {
		t.Fatal("focus dot remains on preview")
	}
	beforeBorder, afterBorder := strings.Split(before, "\n")[4], strings.Split(after, "\n")[4]
	if ansi.Cut(beforeBorder, 0, 42) == ansi.Cut(afterBorder, 0, 42) || ansi.Cut(beforeBorder, 42, 100) == ansi.Cut(afterBorder, 42, 100) {
		t.Fatal("tab did not change both panel borders")
	}
}

func TestMarkdownStripsTerminalControls(t *testing.T) {
	m := model{}
	out := m.markdown("# Safe\n\n\x1b]0;malicious\x07body", 50)
	if strings.Contains(out, "malicious") || strings.Contains(out, "\x1b]0;") {
		t.Fatalf("terminal control leaked: %q", out)
	}
}

func TestNewestFirstAndSortCycling(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Zulu", Modified: time.Unix(1, 0)}, {ID: "20261113103512", Title: "Alpha", Modified: time.Unix(2, 0)}}}
	if got := m.visible(); got[0].ID != "20261113103512" {
		t.Fatalf("default order: %#v", got)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = next.(model)
	if got := m.visible(); got[0].ID != "20261113102030" {
		t.Fatalf("oldest order: %#v", got)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = next.(model)
	if got := m.visible(); got[0].Title != "Alpha" {
		t.Fatalf("title order: %#v", got)
	}
}

func TestSearchBarAndFloatingHelp(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Visible note"}}, width: 100, height: 30, screen: "Notes"}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Search") || !strings.Contains(view, "Visible note") || !strings.Contains(view, "╭") {
		t.Fatalf("missing search, note or panel: %s", view)
	}
	m.help = true
	view = ansi.Strip(m.View())
	if !strings.Contains(view, "Keyboard shortcuts") || !strings.Contains(view, "Search") || !strings.Contains(view, "Visible note") {
		t.Fatalf("help should overlay page: %s", view)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(model).help {
		t.Fatal("escape did not close help")
	}
}

func TestMarkdownRenderedInDetail(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Title", Body: "# Title\n\n**bold text**"}}, width: 80, height: 20, screen: "Detail"}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "bold text") || strings.Contains(view, "**bold text**") {
		t.Fatalf("markdown not rendered: %s", view)
	}
}

func TestReadingModeUsesFullScreenAndRestoresFilters(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Needle", Body: "# Needle\n\nContent"}}, width: 100, height: 20, screen: "Notes", filter: "needle"}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = next.(model)
	view := ansi.Strip(m.View())
	if m.screen != "Detail" || strings.Contains(view, "Search notes by") || !strings.Contains(view, "Content") {
		t.Fatalf("reading view: %s", view)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.screen != "Notes" || m.filter != "needle" {
		t.Fatalf("reading exit lost state: %#v", m)
	}
}

func TestReadingModeKeepsOrphanSelection(t *testing.T) {
	m := model{screen: "Orphans", width: 90, height: 20, all: []notes.Note{{ID: "20261113102030", Title: "Source", Body: "# Source\n\n[Target](../20261113103512/README.md)"}, {ID: "20261113103512", Title: "Target", Body: "# Target"}, {ID: "20261113104845", Title: "Orphan", Body: "# Orphan"}}}
	m.app.Config.Links.MarkdownLinks = true
	before := m.visible()
	if len(before) != 1 || before[0].Title != "Orphan" {
		t.Fatalf("orphans: %#v", before)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = next.(model)
	if !strings.Contains(ansi.Strip(m.View()), "Orphan") {
		t.Fatal("reading switched to a different note")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = next.(model)
	if m.screen != "Orphans" {
		t.Fatalf("lost previous screen: %q", m.screen)
	}
}

func TestRelationshipViewKeepsOrphanSource(t *testing.T) {
	m := model{screen: "Orphans", width: 100, height: 20, all: []notes.Note{{ID: "20261113102030", Title: "Orphan", Body: "# Orphan"}, {ID: "20261113103512", Title: "Source", Body: "[Target](../20261113104845/README.md)"}, {ID: "20261113104845", Title: "Target", Body: "# Target"}}}
	m.app.Config.Links.MarkdownLinks = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = next.(model)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Orphan") || !strings.Contains(view, "No relationships") {
		t.Fatalf("wrong source: %s", view)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.screen != "Orphans" || len(m.visible()) != 1 || m.visible()[0].Title != "Orphan" {
		t.Fatal("orphan view not restored")
	}
}

func TestLinkPickerCanSearchAndCancel(t *testing.T) {
	m := model{screen: "Notes", width: 100, height: 20, all: []notes.Note{{ID: "20261113102030", Title: "Alpha"}, {ID: "20261113103512", Title: "Beta"}, {ID: "20261113104845", Title: "Gamma"}}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = next.(model)
	if !m.linking || !strings.Contains(ansi.Strip(m.View()), "Link to note") {
		t.Fatal("link picker did not open")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Beta")})
	m = next.(model)
	if got := m.linkTargets(); len(got) != 1 || got[0].Title != "Beta" {
		t.Fatalf("filter: %#v", got)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.linking {
		t.Fatal("picker did not close")
	}
}

func TestRelationshipNavigationOpensTargetAndReturns(t *testing.T) {
	from, to := "20261113102030", "20261113103512"
	m := model{screen: "Notes", width: 100, height: 20, filter: "source", all: []notes.Note{{ID: from, Title: "Source", Body: "# Source\n\n[Target](../" + to + "/README.md)"}, {ID: to, Title: "Target", Body: "# Target\n\nTarget body"}}}
	m.app.Config.Links.MarkdownLinks = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = next.(model)
	if m.screen != "Links" || !strings.Contains(ansi.Strip(m.View()), "› ") {
		t.Fatalf("missing relationship selector: %s", m.View())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.screen != "Detail" || !strings.Contains(ansi.Strip(m.View()), "Target body") {
		t.Fatalf("target not opened: %s", m.View())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.screen != "Links" || m.filter != "source" {
		t.Fatalf("previous relationship view lost: %#v", m)
	}
}

func TestBrokenRelationshipCannotNavigate(t *testing.T) {
	m := model{screen: "Notes", all: []notes.Note{{ID: "20261113102030", Title: "Source", Body: "[Missing](../20261113103512/README.md)"}}}
	m.app.Config.Links.MarkdownLinks = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.screen != "Links" || m.err == nil {
		t.Fatalf("broken target should show error: %#v", m)
	}
}

func TestBacklinksSelectionAndBackNavigation(t *testing.T) {
	a, b, c := "20261113102030", "20261113103512", "20261113104845"
	m := model{screen: "Notes", all: []notes.Note{{ID: a, Title: "A", Body: "[C](../" + c + "/README.md)"}, {ID: b, Title: "B", Body: "[C](../" + c + "/README.md)"}, {ID: c, Title: "C", Body: "# C"}}}
	m.app.Config.Links.MarkdownLinks = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.relationIndex != 1 || m.selected != 0 {
		t.Fatal("navigation changed selected source")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.screen != "Detail" || m.visible()[m.selected].ID != b {
		t.Fatalf("backlink target: %#v", m.visible())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.screen != "Backlinks" || m.relationIndex != 1 {
		t.Fatal("relationship cursor not restored")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.screen != "Notes" || m.selected != 0 {
		t.Fatal("source note not restored")
	}
}

func TestNestedReadingAndRelationshipBackNavigation(t *testing.T) {
	a, b := "20261113102030", "20261113103512"
	m := model{screen: "Notes", filter: "source", all: []notes.Note{{ID: a, Title: "Source", Body: "[Target](../" + b + "/README.md)"}, {ID: b, Title: "Target", Body: "# Target"}}}
	m.app.Config.Links.MarkdownLinks = true
	press := func(msg tea.KeyMsg) { next, _ := m.Update(msg); m = next.(model) }
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	press(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != "Detail" {
		t.Fatal("target not open")
	}
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if m.screen != "Backlinks" {
		t.Fatal("backlinks not open")
	}
	press(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	press(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != "Backlinks" {
		t.Fatalf("reading should return to backlinks, got %q", m.screen)
	}
	press(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != "Detail" {
		t.Fatalf("backlinks should return to target, got %q", m.screen)
	}
	press(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != "Links" || m.filter != "source" {
		t.Fatal("original relationship context lost")
	}
}

func TestLinkPickerCreatesRelationship(t *testing.T) {
	root := t.TempDir()
	older, newer := "20261113102030", "20261113103512"
	for id, body := range map[string]string{older: "# Older\n", newer: "# Newer\n"} {
		dir := filepath.Join(root, id)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := model{screen: "Notes", width: 100, height: 20}
	m.app.Config.Notes.Directory = root
	m.app.Config.Links.MarkdownLinks = true
	var err error
	m.all, err = m.app.Notes()
	if err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.linking || m.linkErr != nil {
		t.Fatalf("picker error: %v", m.linkErr)
	}
	content, err := os.ReadFile(filepath.Join(root, newer, "README.md"))
	if err != nil || !strings.Contains(string(content), "../"+older+"/README.md") {
		t.Fatalf("link missing: %s %v", content, err)
	}
	if !strings.Contains(m.notice, "Linked") {
		t.Fatal("success notice missing")
	}
}

func TestPreviewDisplaysNoteType(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Typed", Body: "# Typed", Type: notes.Permanent}}, width: 100, height: 20, screen: "Notes"}
	m.app.Config.TUI.Preview = true
	if !strings.Contains(ansi.Strip(m.View()), "Preview · permanent") {
		t.Fatal("preview does not show note type")
	}
}

func TestSortAndSearchKeepInputAndSelectionSeparate(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Same"}, {ID: "20261113103512", Title: "Same"}}, width: 80, height: 20}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = next.(model)
	if m.sortOrder != 0 || m.filter != "s" {
		t.Fatalf("shortcut intercepted search: %#v", m)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if len(m.visible()) != 2 || m.visible()[0].ID != "20261113103512" {
		t.Fatalf("search order: %#v", m.visible())
	}
}

func TestTypeAndTagFiltersCombineWithSearch(t *testing.T) {
	m := model{screen: "Notes", width: 100, height: 20, filter: "needle", all: []notes.Note{
		{ID: "20261113102030", Title: "Needle alpha", Type: notes.Fleeting, Tags: []string{"go"}},
		{ID: "20261113103512", Title: "Needle beta", Type: notes.Permanent, Tags: []string{"go", "reading"}},
		{ID: "20261113104845", Title: "Needle gamma", Type: notes.Permanent, Tags: []string{"Go"}},
		{ID: "20261113105959", Title: "Other", Type: notes.Permanent, Tags: []string{"go"}},
	}}
	m.app.Config.Search.MaxResults = 1
	press := func(key string) {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = next.(model)
	}
	press("t")
	press("t")
	press("t")
	if m.typeFilter != notes.Permanent {
		t.Fatalf("type filter: %q", m.typeFilter)
	}
	press("#")
	press("#")
	if m.tagFilter != "go" {
		t.Fatalf("tag filter: %q", m.tagFilter)
	}
	list := m.visible()
	if len(list) != 1 || list[0].ID != "20261113103512" {
		t.Fatalf("filtered list: %#v", list)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "permanent") || !strings.Contains(view, "go") {
		t.Fatalf("filters not visible: %s", view)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.typeFilter != "" || m.tagFilter != "" || m.filter != "" {
		t.Fatal("escape did not clear filters")
	}
}

func TestFilterShortcutsRespectSearchAndEmptyTags(t *testing.T) {
	m := model{screen: "Notes", searching: true, all: []notes.Note{{ID: "20261113102030", Title: "note"}}}
	for _, key := range []string{"t", "#"} {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = next.(model)
	}
	if m.filter != "t#" || m.typeFilter != "" || m.tagFilter != "" {
		t.Fatalf("search captured incorrectly: %#v", m)
	}
	m.searching = false
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
	m = next.(model)
	if m.tagFilter != "" {
		t.Fatal("no tags should leave tag filter empty")
	}
	m.help = true
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if next.(model).typeFilter != "" {
		t.Fatal("help failed to capture type shortcut")
	}
}

func TestFilterCyclingAndSelectionReset(t *testing.T) {
	m := model{screen: "Notes", selected: 1, previewOffset: 4, all: []notes.Note{{ID: "20261113102030", Tags: []string{"one"}}, {ID: "20261113103512", Tags: []string{"two"}}}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
	m = next.(model)
	if m.tagFilter != "one" || m.selected != 0 || m.previewOffset != 0 {
		t.Fatalf("filter must reset selection: %#v", m)
	}
	for i := 0; i < 2; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
		m = next.(model)
	}
	if m.tagFilter != "" {
		t.Fatal("tag filter did not cycle to all")
	}
	for i := 0; i < 4; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		m = next.(model)
	}
	if m.typeFilter != "" {
		t.Fatal("type filter did not cycle to all")
	}
}

func TestLayoutFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{20, 8}, {40, 10}, {80, 20}, {120, 30}} {
		for _, help := range []bool{false, true} {
			m := model{all: []notes.Note{{ID: "20261113102030", Title: "A long title that should not overflow the frame", Body: "# Heading\n\nBody"}}, width: size[0], height: size[1], screen: "Notes", help: help}
			view := m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > size[1] {
				t.Fatalf("%dx%d: %d lines\n%s", size[0], size[1], len(lines), view)
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("%dx%d: overwide line %q (%d)", size[0], size[1], line, ansi.StringWidth(line))
				}
			}
		}
	}
}
func TestEmptyStateAndResize(t *testing.T) {
	m := model{}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	got := next.(model)
	if got.width != 40 || got.height != 10 || !strings.Contains(got.View(), "No notes") {
		t.Fatal(got.View())
	}
}

func TestNewNoteWithInvalidEditorRemainsVisible(t *testing.T) {
	root := t.TempDir()
	m := model{width: 100, height: 20, screen: "Notes"}
	m.app.Config.Notes.Directory = root
	m.app.Config.Editor.Command = "'unclosed"
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(model)
	if m.err == nil || len(m.all) != 1 {
		t.Fatalf("expected visible preserved note and error: %#v", m)
	}
	if !strings.Contains(m.err.Error(), m.all[0].Path) {
		t.Fatalf("error must identify preserved note: %v", m.err)
	}
}

func TestTabFocusAndPreviewScrolling(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Long", Body: strings.Repeat("A paragraph of content.\n\n", 40)}, {ID: "20261113103512", Title: "Latest", Body: strings.Repeat("Latest content.\n\n", 40)}}, width: 100, height: 15, screen: "Notes"}
	m.app.Config.TUI.Preview = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	if !m.previewFocused {
		t.Fatal("Tab did not focus preview")
	}
	before := ansi.Strip(m.View())
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.previewOffset != 1 || m.selected != 0 {
		t.Fatalf("down did not scroll preview: %#v", m)
	}
	if ansi.Strip(m.View()) == before {
		t.Fatal("scroll did not change rendered preview")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = next.(model)
	if m.previewOffset != 0 {
		t.Fatalf("k did not scroll up: %d", m.previewOffset)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = next.(model)
	if m.previewFocused || m.selected != 1 || m.previewOffset != 0 {
		t.Fatalf("list focus: %#v", m)
	}
}

func TestNarrowPreviewAndDetailScroll(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Long", Body: strings.Repeat("Paragraph with text.\n\n", 30)}}, width: 50, height: 12, screen: "Notes"}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	if !strings.Contains(ansi.Strip(m.View()), "Preview") {
		t.Fatal("narrow preview missing")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = next.(model)
	if m.previewOffset != 1 {
		t.Fatalf("preview did not scroll: %d", m.previewOffset)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.screen != "Detail" || m.previewOffset != 1 {
		t.Fatalf("detail did not scroll: %#v", m)
	}
}

func TestPreviewScrollClampsAndModalCapturesTab(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Short", Body: "# Short"}}, width: 100, height: 20, screen: "Notes"}
	m.app.Config.TUI.Preview = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.previewOffset != 0 {
		t.Fatal("short note scrolled past end")
	}
	m.help = true
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !next.(model).previewFocused {
		t.Fatal("help failed to capture tab")
	}
	m.help = false
	m.searching = true
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !next.(model).previewFocused {
		t.Fatal("search failed to capture tab")
	}
}

func TestPreviewScrollReachesEndAndClampsAfterResize(t *testing.T) {
	m := model{all: []notes.Note{{ID: "20261113102030", Title: "Long", Body: strings.Repeat("paragraph\n\n", 30) + "END OF NOTE"}}, width: 100, height: 12, screen: "Notes"}
	m.app.Config.TUI.Preview = true
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(model)
	for i := 0; i < 200; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = next.(model)
	}
	if !strings.Contains(ansi.Strip(m.View()), "END OF NOTE") {
		t.Fatal("end of note is not visible")
	}
	end := m.previewOffset
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.previewOffset != end {
		t.Fatal("scrolled past end")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(model)
	if m.previewOffset != end-1 {
		t.Fatal("up did not scroll")
	}
	next, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 200})
	m = next.(model)
	if m.previewOffset != 0 {
		t.Fatalf("resize did not clamp offset: %d", m.previewOffset)
	}
}
