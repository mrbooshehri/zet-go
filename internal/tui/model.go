package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mrbooshehri/zet-go/internal/app"
	"github.com/mrbooshehri/zet-go/internal/config"
	"github.com/mrbooshehri/zet-go/internal/notes"
	"io"
	"sort"
	"strings"
)

type model struct {
	app                   app.App
	all                   []notes.Note
	selected              int
	width, height         int
	filter                string
	searching, help       bool
	screen                string
	err                   error
	created               *notes.CreatedNote
	sortOrder             int
	previewFocused        bool
	previewOffset         int
	typeFilter            notes.NoteType
	tagFilter             string
	searchInput           textinput.Model
	searchInputReady      bool
	readingFrom           string
	readingPreviewFocused bool
	readingViaRelation    bool
	linking               bool
	linkSource            string
	linkIndex             int
	linkQuery             string
	linkSearching         bool
	linkInput             textinput.Model
	linkErr               error
	notice                string
	pickMode              bool
	pickedID              string
	settingsFrom          string
	relationFrom          string
	relationIndex         int
	relationBack          []relationState
	relationOriginReading readingState
}
type readingState struct {
	from                        string
	previewFocused, viaRelation bool
}
type relationState struct {
	screen, relationFrom, filter          string
	selected, relationIndex               int
	typeFilter                            notes.NoteType
	tagFilter                             string
	readingFrom                           string
	readingPreviewFocused, previewFocused bool
	readingViaRelation                    bool
	relationOriginReading                 readingState
}
type editorDone struct {
	err     error
	created *notes.CreatedNote
	id      string
}
type settingsDone struct {
	draft *config.EditDraft
	err   error
}

func Run(a app.App) error {
	all, err := a.Notes()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(model{app: a, all: all, screen: "Notes"}, tea.WithAltScreen()).Run()
	return err
}

// Pick displays the selector on terminalOut and returns the selected note ID.
func Pick(a app.App, terminalOut io.Writer) (string, error) {
	all, err := a.Notes()
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", fmt.Errorf("no notes to select")
	}
	final, err := tea.NewProgram(model{app: a, all: all, screen: "Notes", pickMode: true}, tea.WithInputTTY(), tea.WithOutput(terminalOut), tea.WithAltScreen()).Run()
	if err != nil {
		return "", fmt.Errorf("open note picker: %w", err)
	}
	id := final.(model).pickedID
	if id == "" {
		return "", fmt.Errorf("note selection canceled")
	}
	return id, nil
}
func (m model) Init() tea.Cmd { return nil }
func (m model) visible() []notes.Note {
	list := append([]notes.Note(nil), m.all...)
	orphanContext := m.screen == "Orphans" || (m.screen == "Detail" && m.readingFrom == "Orphans") || (m.isRelations() && (m.relationFrom == "Orphans" || (m.relationFrom == "Detail" && m.readingFrom == "Orphans")))
	if orphanContext {
		g := m.app.Graph(m.all)
		set := map[string]bool{}
		for _, id := range g.Orphans {
			set[id] = true
		}
		var out []notes.Note
		for _, n := range list {
			if set[n.ID] {
				out = append(out, n)
			}
		}
		list = out
	}
	if m.typeFilter != "" || m.tagFilter != "" {
		filtered := make([]notes.Note, 0, len(list))
		for _, n := range list {
			if m.typeFilter != "" && n.Type != m.typeFilter {
				continue
			}
			if m.tagFilter != "" {
				found := false
				for _, tag := range n.Tags {
					if tag == m.tagFilter {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			filtered = append(filtered, n)
		}
		list = filtered
	}
	if m.filter != "" {
		list = notes.SearchWithOptions(list, m.filter, m.app.Config.Search.CaseSensitive, 0)
	}
	sort.SliceStable(list, func(i, j int) bool {
		switch m.sortOrder {
		case 1:
			return list[i].ID < list[j].ID
		case 2:
			a, b := strings.ToLower(list[i].Title), strings.ToLower(list[j].Title)
			if a != b {
				return a < b
			}
			return list[i].ID > list[j].ID
		case 3:
			if !list[i].Modified.Equal(list[j].Modified) {
				return list[i].Modified.After(list[j].Modified)
			}
			return list[i].ID > list[j].ID
		default:
			return list[i].ID > list[j].ID
		}
	})
	if m.filter != "" && m.app.Config.Search.MaxResults > 0 && len(list) > m.app.Config.Search.MaxResults {
		list = list[:m.app.Config.Search.MaxResults]
	}
	return list
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		if m.searchInputReady {
			m.searchInput.Width = m.searchInputWidth()
			m.searchInput.SetCursor(m.searchInput.Position())
		}
		m.clampPreview()
	case editorDone:
		if v.created != nil {
			m.err = m.app.FinishNew(*v.created, v.err)
		} else if v.err != nil {
			m.err = v.err
		} else {
			m.err = m.app.Commit(v.id, "Edit note "+v.id)
		}
		all, e := m.app.Notes()
		if e != nil {
			m.err = e
		} else {
			m.all = all
		}
		m.created = nil
		m.previewOffset = 0
	case settingsDone:
		updated, err := v.draft.Finish(v.err, m.app.DirectoryOverride, m.app.EditorOverride)
		if err != nil {
			m.err = err
		} else {
			m.app.Config = updated
			m.err = nil
			all, e := m.app.Notes()
			if e != nil {
				m.err = e
			} else {
				m.all = all
			}
			m.selected = 0
			m.notice = "Settings saved"
		}
		m.previewOffset = 0
	case tea.KeyMsg:
		key := v.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.linking {
			return m.updateLinkPicker(v)
		}
		if m.searching {
			if !m.searchInputReady {
				m.initSearchInput()
			}
			switch key {
			case "esc", "enter":
				m.searching = false
				m.searchInput.Blur()
			default:
				if v.Type == tea.KeySpace && len(v.Runes) == 0 {
					v.Runes = []rune{' '}
				}
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(v)
				if m.filter != m.searchInput.Value() {
					m.filter = m.searchInput.Value()
					m.selected = 0
					m.previewOffset = 0
					m.relationIndex = 0
				}
				return m, cmd
			}
			return m, nil
		}
		if m.help {
			if key == "?" || key == "esc" {
				m.help = false
			}
			return m, nil
		}
		if m.screen == "Settings" {
			switch key {
			case "esc", "S":
				m.screen = m.settingsFrom
				if m.screen == "" {
					m.screen = "Notes"
				}
				m.previewOffset = 0
			case "j", "down":
				m.previewOffset++
				m.clampPreview()
			case "k", "up":
				m.previewOffset = max(0, m.previewOffset-1)
			case "e":
				draft, err := config.BeginEdit(m.app.ConfigPath)
				if err != nil {
					m.err = err
					break
				}
				cmd, err := m.app.Editor(draft.Path)
				if err != nil {
					return m.Update(settingsDone{draft: draft, err: err})
				}
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return settingsDone{draft: draft, err: err} })
			case "?":
				m.help = true
			case "q":
				return m, tea.Quit
			}
			return m, nil
		}
		list := m.visible()
		if m.pickMode {
			switch key {
			case "enter":
				if len(list) > 0 && m.selected >= 0 && m.selected < len(list) {
					m.pickedID = list[m.selected].ID
					return m, tea.Quit
				}
				return m, nil
			case "esc", "q":
				return m, tea.Quit
			case "n", "e", "L", "g", "b", "l", "r", "o", "S":
				return m, nil
			}
		}
		switch key {
		case "q":
			return m, tea.Quit
		case "?":
			m.help = true
		case "S":
			m.settingsFrom = m.screen
			m.screen = "Settings"
			m.previewOffset = 0
			m.err = nil
			m.notice = ""
		case "/":
			m.searching = true
			m.initSearchInput()
			return m, m.searchInput.Focus()
		case "s":
			m.sortOrder = (m.sortOrder + 1) % 4
			m.selected = 0
			m.previewOffset = 0
			m.relationIndex = 0
		case "t":
			types := []notes.NoteType{"", notes.Fleeting, notes.Literature, notes.Permanent}
			for i, typ := range types {
				if typ == m.typeFilter {
					m.typeFilter = types[(i+1)%len(types)]
					break
				}
			}
			m.selected = 0
			m.previewOffset = 0
			m.relationIndex = 0
		case "#":
			seen := map[string]bool{}
			tags := []string{""}
			for _, n := range m.all {
				for _, tag := range n.Tags {
					if tag != "" && !seen[tag] {
						seen[tag] = true
						tags = append(tags, tag)
					}
				}
			}
			sort.Strings(tags[1:])
			index := -1
			for i, tag := range tags {
				if tag == m.tagFilter {
					index = i
					break
				}
			}
			m.tagFilter = tags[(index+1)%len(tags)]
			m.selected = 0
			m.previewOffset = 0
			m.relationIndex = 0
		case "esc":
			if m.screen == "Detail" {
				m.closeReading()
				break
			}
			if m.isRelations() {
				m.screen = m.relationFrom
				if m.screen == "" {
					m.screen = "Notes"
				}
				if m.screen == "Detail" {
					m.readingFrom = m.relationOriginReading.from
					m.readingPreviewFocused = m.relationOriginReading.previewFocused
					m.readingViaRelation = m.relationOriginReading.viaRelation
				}
				m.relationIndex = 0
				m.err = nil
				break
			}
			m.filter = ""
			m.typeFilter = ""
			m.tagFilter = ""
			m.screen = "Notes"
			m.selected = 0
			m.previewFocused = false
			m.previewOffset = 0
		case "tab":
			if m.screen == "Notes" || m.screen == "Review" || m.screen == "Orphans" || m.screen == "" {
				m.previewFocused = !m.previewFocused
			}
		case "j", "down":
			if m.previewFocused || m.screen == "Detail" {
				m.previewOffset++
				m.clampPreview()
			} else if m.isRelations() {
				if m.relationIndex < len(m.relationships())-1 {
					m.relationIndex++
				}
			} else if m.selected < len(list)-1 {
				m.selected++
				m.previewOffset = 0
			}
		case "k", "up":
			if m.previewFocused || m.screen == "Detail" {
				m.previewOffset = max(0, m.previewOffset-1)
			} else if m.isRelations() {
				m.relationIndex = max(0, m.relationIndex-1)
			} else if m.selected > 0 {
				m.selected--
				m.previewOffset = 0
			}
		case "g":
			m.openRelations("Graph")
		case "b":
			m.openRelations("Backlinks")
		case "l":
			m.openRelations("Links")
		case "r":
			m.screen = "Review"
		case "o":
			m.screen = "Orphans"
			m.selected = 0
			m.previewOffset = 0
		case "enter":
			if m.isRelations() {
				m.followRelationship()
			} else if len(list) > 0 && m.screen != "Detail" {
				m.openReading()
			}
		case "v":
			if m.screen == "Detail" {
				m.closeReading()
			} else if len(list) > 0 {
				m.openReading()
			}
		case "L":
			if len(list) > 0 && m.selected < len(list) {
				m.linking = true
				m.linkSource = list[m.selected].ID
				m.linkQuery = ""
				m.linkIndex = 0
				m.linkErr = nil
				m.linkSearching = false
				m.notice = ""
			}
		case "e":
			if m.selected < len(list) {
				n := list[m.selected]
				cmd, e := m.app.Editor(n.Path)
				if e != nil {
					m.err = e
					break
				}
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editorDone{err: err, id: n.ID} })
			}
		case "n":
			n, e := m.app.NewNote()
			if e != nil {
				m.err = e
				break
			}
			cmd, e := m.app.Editor(n.Path)
			if e != nil {
				return m.Update(editorDone{err: e, created: &n})
			}
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editorDone{err: err, created: &n} })
		}
	}
	if m.linking && m.linkSearching {
		var cmd tea.Cmd
		m.linkInput, cmd = m.linkInput.Update(msg)
		if m.linkQuery != m.linkInput.Value() {
			m.linkQuery = m.linkInput.Value()
			m.linkIndex = 0
		}
		return m, cmd
	}
	if m.searching && m.searchInputReady {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if m.filter != m.searchInput.Value() {
			m.filter = m.searchInput.Value()
			m.selected = 0
			m.previewOffset = 0
			m.relationIndex = 0
		}
		return m, cmd
	}
	return m, nil
}

func (m *model) openReading() {
	m.readingFrom = m.screen
	if m.readingFrom == "" {
		m.readingFrom = "Notes"
	}
	m.readingPreviewFocused = m.previewFocused
	m.readingViaRelation = false
	m.screen = "Detail"
	m.previewFocused = false
	m.previewOffset = 0
}
func (m *model) closeReading() {
	if m.readingViaRelation && len(m.relationBack) > 0 {
		i := len(m.relationBack) - 1
		s := m.relationBack[i]
		m.relationBack = m.relationBack[:i]
		m.screen = s.screen
		m.relationFrom = s.relationFrom
		m.filter = s.filter
		m.selected = s.selected
		m.relationIndex = s.relationIndex
		m.typeFilter = s.typeFilter
		m.tagFilter = s.tagFilter
		m.readingFrom = s.readingFrom
		m.readingPreviewFocused = s.readingPreviewFocused
		m.readingViaRelation = s.readingViaRelation
		m.relationOriginReading = s.relationOriginReading
		m.previewFocused = s.previewFocused
		m.previewOffset = 0
		return
	}
	m.screen = m.readingFrom
	if m.screen == "" {
		m.screen = "Notes"
	}
	m.previewFocused = m.readingPreviewFocused
	m.previewOffset = 0
}

func (m model) isRelations() bool {
	return m.screen == "Links" || m.screen == "Backlinks" || m.screen == "Graph"
}
func (m *model) openRelations(screen string) {
	if !m.isRelations() {
		m.relationFrom = m.screen
		if m.screen == "Detail" {
			m.relationOriginReading = readingState{from: m.readingFrom, previewFocused: m.readingPreviewFocused, viaRelation: m.readingViaRelation}
		}
	}
	m.screen = screen
	m.previewFocused = false
	m.relationIndex = 0
	m.err = nil
}
func (m model) relationships() []notes.Link {
	list := m.visible()
	if m.selected < 0 || m.selected >= len(list) {
		return nil
	}
	g := m.app.Graph(m.all)
	if m.screen == "Backlinks" {
		return g.Incoming[list[m.selected].ID]
	}
	return g.Outgoing[list[m.selected].ID]
}
func (m *model) followRelationship() {
	links := m.relationships()
	if m.relationIndex < 0 || m.relationIndex >= len(links) {
		return
	}
	id := links[m.relationIndex].To
	if m.screen == "Backlinks" {
		id = links[m.relationIndex].From
	}
	found := false
	for _, n := range m.all {
		if n.ID == id {
			found = true
			break
		}
	}
	if !found {
		m.err = fmt.Errorf("related note %s not found", id)
		return
	}
	m.relationBack = append(m.relationBack, relationState{screen: m.screen, relationFrom: m.relationFrom, filter: m.filter, selected: m.selected, relationIndex: m.relationIndex, typeFilter: m.typeFilter, tagFilter: m.tagFilter, readingFrom: m.readingFrom, readingPreviewFocused: m.readingPreviewFocused, readingViaRelation: m.readingViaRelation, relationOriginReading: m.relationOriginReading, previewFocused: m.previewFocused})
	m.screen = "Notes"
	m.filter = ""
	m.typeFilter = ""
	m.tagFilter = ""
	for i, n := range m.visible() {
		if n.ID == id {
			m.selected = i
			break
		}
	}
	m.err = nil
	m.openReading()
	m.readingViaRelation = true
}

func (m model) linkTargets() []notes.Note {
	targets := make([]notes.Note, 0, len(m.all))
	for _, n := range m.all {
		if n.ID != m.linkSource {
			targets = append(targets, n)
		}
	}
	if m.linkQuery != "" {
		targets = notes.SearchWithOptions(targets, m.linkQuery, m.app.Config.Search.CaseSensitive, 0)
	}
	sort.SliceStable(targets, func(i, j int) bool { return targets[i].ID > targets[j].ID })
	return targets
}

func (m model) updateLinkPicker(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := key.String()
	if m.linkSearching {
		switch s {
		case "esc", "enter":
			m.linkSearching = false
			m.linkInput.Blur()
			return m, nil
		default:
			if key.Type == tea.KeySpace && len(key.Runes) == 0 {
				key.Runes = []rune{' '}
			}
			var cmd tea.Cmd
			m.linkInput, cmd = m.linkInput.Update(key)
			if m.linkQuery != m.linkInput.Value() {
				m.linkQuery = m.linkInput.Value()
				m.linkIndex = 0
			}
			return m, cmd
		}
	}
	switch s {
	case "esc", "q":
		m.linking = false
		m.linkErr = nil
	case "/":
		m.linkErr = nil
		m.linkSearching = true
		m.linkInput = textinput.New()
		m.linkInput.Prompt = ""
		m.linkInput.Width = max(1, min(40, m.width-12))
		m.linkInput.SetValue(m.linkQuery)
		m.linkInput.CursorEnd()
		return m, m.linkInput.Focus()
	case "j", "down":
		if m.linkIndex < len(m.linkTargets())-1 {
			m.linkIndex++
			m.linkErr = nil
		}
	case "k", "up":
		if m.linkIndex > 0 {
			m.linkIndex--
			m.linkErr = nil
		}
	case "enter":
		targets := m.linkTargets()
		if m.linkIndex >= len(targets) {
			break
		}
		target := targets[m.linkIndex]
		if err := m.app.LinkNotes(m.linkSource, target.ID); err != nil {
			m.linkErr = err
			break
		}
		m.err = nil
		all, err := m.app.Notes()
		if err != nil {
			m.err = err
		} else {
			m.all = all
		}
		m.notice = "Linked " + m.linkSource + " → " + target.ID
		m.linking = false
		m.linkErr = nil
	}
	return m, nil
}

func (m *model) initSearchInput() {
	m.searchInput = textinput.New()
	m.searchInput.Prompt = ""
	m.searchInput.Placeholder = "Search notes by title, body or ID"
	m.searchInput.Width = m.searchInputWidth()
	m.searchInput.SetValue(m.filter)
	m.searchInput.CursorEnd()
	m.searchInput.Focus()
	m.searchInputReady = true
}
