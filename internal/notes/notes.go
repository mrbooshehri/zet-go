// Package notes reads legacy ZET notes without modifying them.
package notes

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type Note struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	Path            string    `json:"path"`
	Modified        time.Time `json:"modified"`
	Type            NoteType  `json:"type,omitempty"`
	Tags            []string  `json:"tags,omitempty"`
	Sources         []string  `json:"sources,omitempty"`
	MetadataWarning string    `json:"metadata_warning,omitempty"`
}

var idPattern = regexp.MustCompile(`^20[0-9]{12}$`)

// ValidID confines operations to legacy timestamp-shaped directory names.
func ValidID(id string) bool { return idPattern.MatchString(id) }

func Load(root string) ([]Note, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read notes directory: %w", err)
	}
	notes := make([]Note, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !ValidID(entry.Name()) {
			continue
		}
		path := filepath.Join(root, entry.Name(), "README.md")
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		body := string(data)
		title := "<untitled>"
		metadata, content, warning := readMetadata(body)
		first, _, _ := strings.Cut(content, "\n")
		if strings.HasPrefix(first, "# ") {
			if heading := strings.TrimSpace(strings.TrimPrefix(first, "# ")); heading != "" {
				title = heading
			}
		}
		notes = append(notes, Note{ID: entry.Name(), Title: title, Body: body, Path: path, Modified: info.ModTime(), Type: metadata.Type, Tags: metadata.Tags, Sources: metadata.Sources, MetadataWarning: warning})
	}
	return notes, nil
}

type Link struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description"`
}
type Graph struct {
	Outgoing map[string][]Link `json:"outgoing"`
	Incoming map[string][]Link `json:"incoming"`
	Broken   []Link            `json:"broken"`
	Orphans  []string          `json:"orphans"`
}

var wikiPattern = regexp.MustCompile(`\[\[(20[0-9]{12})\]\]`)
var targetPattern = regexp.MustCompile(`^\.\./(20[0-9]{12})/README\.md$`)

func extractLinks(n Note, wiki, markdown bool) []Link {
	_, content, _ := readMetadata(n.Body)
	source := []byte(content)
	doc := goldmark.DefaultParser().Parse(text.NewReader(source))
	links := []Link{}
	var pending strings.Builder
	flush := func() {
		if wiki {
			for _, m := range wikiPattern.FindAllStringSubmatch(pending.String(), -1) {
				links = append(links, Link{From: n.ID, To: m[1], Description: m[1]})
			}
		}
		pending.Reset()
	}
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			if node.Type() == ast.TypeBlock {
				flush()
			}
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case ast.KindCodeBlock, ast.KindFencedCodeBlock, ast.KindCodeSpan, ast.KindHTMLBlock, ast.KindRawHTML:
			flush()
			return ast.WalkSkipChildren, nil
		}
		if link, ok := node.(*ast.Link); ok {
			flush()
			u, err := url.Parse(string(link.Destination))
			if markdown && err == nil && u.Scheme == "" && u.Host == "" {
				m := targetPattern.FindStringSubmatch(u.Path)
				if len(m) > 1 {
					links = append(links, Link{From: n.ID, To: m[1], Description: string(link.Text(source))})
				}
			}
			return ast.WalkSkipChildren, nil
		}
		if t, ok := node.(*ast.Text); ok {
			pending.Write(t.Segment.Value(source))
		}
		return ast.WalkContinue, nil
	})
	flush()
	return links
}

func BuildGraph(notes []Note) Graph {
	return BuildGraphWithOptions(notes, true, true)
}

func BuildGraphWithOptions(notes []Note, wiki, markdown bool) Graph {
	g := Graph{Outgoing: map[string][]Link{}, Incoming: map[string][]Link{}, Broken: []Link{}, Orphans: []string{}}
	existing := map[string]bool{}
	for _, n := range notes {
		existing[n.ID] = true
	}
	for _, n := range notes {
		seen := map[string]bool{}
		for _, link := range extractLinks(n, wiki, markdown) {
			target := link.To
			if seen[target] {
				continue
			}
			seen[target] = true
			g.Outgoing[n.ID] = append(g.Outgoing[n.ID], link)
			if existing[target] {
				g.Incoming[target] = append(g.Incoming[target], link)
			} else {
				g.Broken = append(g.Broken, link)
			}
		}
	}
	for _, n := range notes {
		if len(g.Outgoing[n.ID]) == 0 && len(g.Incoming[n.ID]) == 0 {
			g.Orphans = append(g.Orphans, n.ID)
		}
	}
	return g
}

func Search(notes []Note, query string) []Note {
	return SearchWithOptions(notes, query, false, 0)
}

func SearchWithOptions(notes []Note, query string, caseSensitive bool, maxResults int) []Note {
	normalize := func(s string) string {
		if caseSensitive {
			return s
		}
		return strings.ToLower(s)
	}
	q := normalize(query)
	if q == "" {
		return []Note{}
	}
	result := make([]Note, 0)
	score := func(n Note) int {
		if strings.Contains(normalize(n.ID), q) {
			return 3
		}
		if strings.Contains(normalize(n.Title), q) {
			return 2
		}
		if strings.Contains(normalize(n.Body), q) {
			return 1
		}
		return 0
	}
	for _, n := range notes {
		if score(n) > 0 {
			result = append(result, n)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return score(result[i]) > score(result[j]) })
	if maxResults > 0 && len(result) > maxResults {
		return result[:maxResults]
	}
	return result
}
