package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"gopkg.in/yaml.v3"
)

var (
	accent        = lipgloss.AdaptiveColor{Light: "#4F46A5", Dark: "#AFA4FF"}
	muted         = lipgloss.AdaptiveColor{Light: "#626879", Dark: "#9BA5B7"}
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#51459B"))
)

// cleanText prevents note content from injecting terminal controls.
func cleanText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}

func (m model) markdown(body string, width int) string {
	body = cleanText(body)
	if width < 8 {
		return body
	}
	style := m.app.Config.TUI.Theme
	if style != "light" && style != "dark" {
		if lipgloss.HasDarkBackground() {
			style = "dark"
		} else {
			style = "light"
		}
	}
	styleOption := glamour.WithStandardStyle(style)
	r, err := glamour.NewTermRenderer(styleOption, glamour.WithWordWrap(width))
	if err != nil {
		return body
	}
	result, err := r.Render(body)
	if err != nil {
		return body
	}
	return strings.Trim(result, "\n")
}

func panel(title, content string, width, height int, focused bool) string {
	if width < 4 || height < 3 {
		return ansi.Truncate(content, max(0, width), "")
	}
	innerWidth, innerHeight := width-4, height-2
	label := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(ansi.Truncate(title, innerWidth, ""))
	if innerHeight > 1 {
		content = "\n" + lipgloss.NewStyle().MaxWidth(innerWidth).MaxHeight(innerHeight-1).Render(content)
	} else {
		content = ""
	}
	borderColor := muted
	border := lipgloss.RoundedBorder()
	if focused {
		borderColor = accent
		border = lipgloss.ThickBorder()
	}
	return lipgloss.NewStyle().Border(border).BorderForeground(borderColor).Padding(0, 1).Width(width - 2).Height(innerHeight).MaxWidth(width).MaxHeight(height).Render(label + content)
}

func (m model) previewSize() (int, int) {
	width, height := m.width, m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}
	if m.screen == "Detail" || m.screen == "Settings" {
		return max(1, width-4), max(1, height-4)
	}
	if width >= 80 && m.app.Config.TUI.Preview && m.screen != "Detail" && m.screen != "Graph" && m.screen != "Backlinks" && m.screen != "Links" {
		width -= width * 42 / 100
	}
	return max(1, width-4), max(1, height-8)
}

func (m model) previewLines() []string {
	if m.screen == "Settings" {
		data, err := yaml.Marshal(m.app.Config)
		if err != nil {
			return []string{"Unable to display settings: " + err.Error()}
		}
		return strings.Split(strings.TrimSuffix(cleanText(string(data)), "\n"), "\n")
	}
	list := m.visible()
	if len(list) == 0 {
		return nil
	}
	idx := min(max(0, m.selected), len(list)-1)
	width, _ := m.previewSize()
	return strings.Split(m.markdown(list[idx].Body, width), "\n")
}

func (m *model) clampPreview() {
	_, height := m.previewSize()
	m.previewOffset = min(max(0, m.previewOffset), max(0, len(m.previewLines())-height))
}

func (m model) previewContent() string {
	lines := m.previewLines()
	if len(lines) == 0 {
		return "Select a note to preview it."
	}
	_, height := m.previewSize()
	start := min(max(0, m.previewOffset), max(0, len(lines)-height))
	return strings.Join(lines[start:], "\n")
}

func (m model) filterLabels() string {
	filters := ""
	if m.typeFilter != "" {
		filters += "[type:" + string(m.typeFilter) + "] "
	}
	if m.tagFilter != "" {
		tag := strings.NewReplacer("\n", " ", "\t", " ").Replace(cleanText(m.tagFilter))
		filters += "[tag:" + tag + "] "
	}
	return filters
}

func (m model) searchInputWidth() int {
	width := m.width
	if width < 1 {
		width = 80
	}
	return max(1, width-7-ansi.StringWidth(m.filterLabels()))
}

func (m model) View() string {
	width, height := m.width, m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}
	if width < 20 || height < 8 {
		return ansi.Truncate("ZET  / Search  ? Help  q Quit", width, "")
	}
	list := m.visible()
	idx := m.selected
	if idx >= len(list) {
		idx = len(list) - 1
	}
	if idx < 0 {
		idx = 0
	}
	if m.screen == "Settings" {
		footer := " e Edit settings  j/k Scroll  Esc Back"
		if m.err != nil {
			footer = " Error: " + strings.ReplaceAll(cleanText(m.err.Error()), "\n", " ")
		} else if m.notice != "" {
			footer = " " + m.notice
		}
		page := panel("Settings · effective values", m.previewContent(), width, height-1, !m.help) + "\n" + ansi.Truncate(footer, width, "")
		if m.help {
			page = overlayHelp(page, width, height, m.pickMode)
		}
		return page
	}
	if m.screen == "Detail" {
		title := "Reading"
		if len(list) > 0 {
			title += " · " + strings.ReplaceAll(cleanText(list[idx].Title), "\n", " ")
		}
		footer := " Esc/v Back  j/k Scroll  L Link  ? Help  q Quit"
		if m.err != nil {
			footer = " Error: " + strings.ReplaceAll(cleanText(m.err.Error()), "\n", " ")
		} else if m.notice != "" {
			footer = " " + m.notice
		}
		page := panel(title, m.previewContent(), width, height-1, !m.help && !m.linking) + "\n" + ansi.Truncate(footer, width, "")
		if m.help {
			page = overlayHelp(page, width, height, m.pickMode)
		}
		if m.linking {
			page = m.overlayLinkPicker(page, width, height)
		}
		return page
	}
	orders := []string{"Newest", "Oldest", "Title", "Modified"}
	order := m.sortOrder
	if order < 0 || order >= len(orders) {
		order = 0
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(" ZET") + "  " + cleanText(m.screen) + "  ·  " + orders[order] + " first"
	if m.pickMode {
		header = lipgloss.NewStyle().Bold(true).Foreground(accent).Render(" ZET") + "  Pick a note · " + orders[order] + " first"
	}
	searchValue := cleanText(m.filter)
	if searchValue == "" {
		searchValue = "Search notes by title, body or ID"
	}
	searchState := "/"
	if m.searching {
		searchState = "›"
		if m.searchInputReady {
			searchValue = m.searchInput.View()
		} else {
			searchValue += "▏"
		}
	}
	searchBorder := muted
	searchBorderStyle := lipgloss.RoundedBorder()
	if m.searching && !m.help {
		searchBorder = accent
		searchBorderStyle = lipgloss.ThickBorder()
	}
	filters := m.filterLabels()
	search := lipgloss.NewStyle().Border(searchBorderStyle).BorderForeground(searchBorder).Padding(0, 1).Width(width - 2).MaxWidth(width).Render(ansi.Truncate(searchState+"  "+filters+searchValue, width-4, ""))
	bodyHeight := height - 5
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	showPreview := width >= 80 && m.app.Config.TUI.Preview && m.screen != "Detail" && m.screen != "Graph" && m.screen != "Backlinks" && m.screen != "Links"
	listWidth := width
	if showPreview {
		listWidth = width * 42 / 100
	}
	var content string
	if len(list) == 0 {
		content = "No notes found. Press n to create a note or / to search."
	} else {
		n := list[idx]
		switch m.screen {
		case "Detail":
			content = m.previewContent()
		case "Graph", "Backlinks", "Links":
			links := m.relationships()
			var rows []string
			rows = append(rows, n.ID+"  "+cleanText(n.Title))
			capacity := max(1, bodyHeight-4)
			start := 0
			if m.relationIndex >= capacity {
				start = m.relationIndex - capacity + 1
			}
			for i := start; i < len(links) && i < start+capacity; i++ {
				l := links[i]
				id := l.To
				if m.screen == "Backlinks" {
					id = l.From
				}
				title := "(missing)"
				for _, candidate := range m.all {
					if candidate.ID == id {
						title = cleanText(candidate.Title)
						break
					}
				}
				prefix := "  "
				if i == m.relationIndex {
					prefix = "› "
				}
				description := strings.ReplaceAll(cleanText(l.Description), "\n", " ")
				line := ansi.Truncate(prefix+id+"  "+strings.ReplaceAll(title, "\n", " ")+"  "+description, width-4, "…")
				if i == m.relationIndex {
					line = selectedStyle.Width(max(1, width-4)).Render(line)
				}
				rows = append(rows, line)
			}
			if len(links) == 0 {
				rows = append(rows, "No relationships.")
			}
			content = strings.Join(rows, "\n")
		default:
			rows := []string{}
			capacity := max(1, bodyHeight-3)
			start := 0
			if idx >= capacity {
				start = idx - capacity + 1
			}
			for i := start; i < len(list) && len(rows) < capacity; i++ {
				prefix := "  "
				if i == idx {
					prefix = "› "
				}
				label := prefix + list[i].ID + "  " + strings.ReplaceAll(cleanText(list[i].Title), "\n", " ")
				label = ansi.Truncate(label, max(1, listWidth-4), "…")
				if i == idx {
					label = selectedStyle.Width(max(1, listWidth-4)).Render(label)
				}
				rows = append(rows, label)
			}
			content = strings.Join(rows, "\n")
		}
	}
	sectionTitle := m.screen
	if !showPreview && m.previewFocused && m.screen != "Detail" && m.screen != "Graph" && m.screen != "Backlinks" && m.screen != "Links" {
		sectionTitle = "Preview"
		content = m.previewContent()
	}
	if len(list) > 0 && !showPreview && (sectionTitle == "Preview" || m.screen == "Detail") && list[idx].Type != "" {
		sectionTitle += " · " + string(list[idx].Type)
	}
	sectionFocused := !m.searching && !m.help && (!showPreview || !m.previewFocused)
	section := panel(sectionTitle, content, listWidth, bodyHeight, sectionFocused)
	if showPreview {
		previewTitle := "Preview"
		if len(list) > 0 && list[idx].Type != "" {
			previewTitle += " · " + string(list[idx].Type)
		}
		section = lipgloss.JoinHorizontal(lipgloss.Top, section, panel(previewTitle, m.previewContent(), width-listWidth, bodyHeight, m.previewFocused && !m.searching && !m.help))
	}
	footer := " Tab Focus  v Read  L Link  / Search  t Type  # Tag  ? Help"
	if m.isRelations() {
		footer = " j/k Select  Enter Open related note  Esc Back  ? Help"
	}
	if m.pickMode {
		footer = " Enter Select  / Search  j/k Move  t Type  # Tag  Esc Cancel"
	}
	if m.err != nil {
		footer = " Error: " + strings.ReplaceAll(cleanText(m.err.Error()), "\n", " ")
	} else if m.notice != "" {
		footer = " " + m.notice
	}
	page := strings.Join([]string{ansi.Truncate(header, width, ""), search, section, ansi.Truncate(footer, width, "")}, "\n")
	if m.help {
		page = overlayHelp(page, width, height, m.pickMode)
	}
	if m.linking {
		page = m.overlayLinkPicker(page, width, height)
	}
	return page
}

// overlayHelp replaces only the cells occupied by the centered modal.
func overlayHelp(page string, width, height int, picking bool) string {
	help := "Tab        List / preview focus\nj/k, ↑/↓   Move / scroll\n/          Focus search\nt / #      Cycle type / tag filters\ns          Change sort order\nEnter / v  Reading mode\nL          Link selected note\nS          Settings\nn / e      New / edit note\nb / l / g  Backlinks / links / graph\no / r      Orphans / review\nEsc / ?    Close help\nq          Quit"
	if picking {
		help = "j/k, ↑/↓   Move selection\n/          Search\nt / #      Cycle type / tag filters\ns          Change sort order\nTab        List / preview focus\nv          Reading mode\nEnter      Select note\nEsc / q    Cancel"
	}
	w := min(width-2, 48)
	h := min(height-2, 16)
	if w < 8 || h < 4 {
		return page
	}
	modal := panel("Keyboard shortcuts", help, w, h, true)
	return placeOverlay(page, modal, width, height)
}

func (m model) overlayLinkPicker(page string, width, height int) string {
	w, h := min(width-2, 70), min(height-2, 16)
	if w < 12 || h < 6 {
		return page
	}
	query := cleanText(m.linkQuery)
	if m.linkSearching {
		query = m.linkInput.View() + "  (Enter: finish search)"
	} else if query == "" {
		query = "/ search targets"
	} else {
		query = "/ " + query
	}
	rows := []string{ansi.Truncate(query, w-4, ""), "Enter: link  ·  Esc: cancel"}
	targets := m.linkTargets()
	index := min(max(0, m.linkIndex), max(0, len(targets)-1))
	capacity := max(1, h-6)
	if m.linkErr != nil {
		capacity = max(0, capacity-1)
	}
	start := 0
	if index >= capacity && capacity > 0 {
		start = index - capacity + 1
	}
	if len(targets) == 0 {
		rows = append(rows, "No matching target notes.")
	} else {
		for i := start; i < len(targets) && i < start+capacity; i++ {
			prefix := "  "
			if i == index {
				prefix = "› "
			}
			label := ansi.Truncate(prefix+targets[i].ID+"  "+strings.ReplaceAll(cleanText(targets[i].Title), "\n", " "), w-4, "…")
			if i == index {
				label = selectedStyle.Width(w - 4).Render(label)
			}
			rows = append(rows, label)
		}
	}
	if m.linkErr != nil {
		rows = append(rows, ansi.Truncate("Error: "+strings.ReplaceAll(cleanText(m.linkErr.Error()), "\n", " "), w-4, "…"))
	}
	modal := panel("Link to note · "+m.linkSource, strings.Join(rows, "\n"), w, h, true)
	return placeOverlay(page, modal, width, height)
}

func placeOverlay(page, modal string, width, height int) string {
	lines := strings.Split(page, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	over := strings.Split(modal, "\n")
	w := lipgloss.Width(modal)
	x := (width - w) / 2
	y := (height - len(over)) / 2
	if y < 0 {
		y = 0
	}
	for i, line := range over {
		if y+i >= len(lines) {
			break
		}
		base := lines[y+i]
		left := ansi.Cut(base, 0, x)
		left += strings.Repeat(" ", max(0, x-ansi.StringWidth(left)))
		right := ansi.Cut(base, x+w, width)
		line += strings.Repeat(" ", max(0, w-ansi.StringWidth(line)))
		lines[y+i] = left + line + right
	}
	return strings.Join(lines, "\n")
}
