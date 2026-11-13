package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mrbooshehri/zet-go/internal/app"
	"github.com/mrbooshehri/zet-go/internal/config"
	"github.com/mrbooshehri/zet-go/internal/notes"
	"github.com/mrbooshehri/zet-go/internal/tui"
	"github.com/spf13/cobra"
	"io"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
)

var Version = "dev"

func New(out, errout io.Writer) *cobra.Command {
	var cfgPath, dir, editor string
	var jsonOutput bool
	root := &cobra.Command{Use: "zet", Short: "Local-first Markdown notes", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errout)
	root.PersistentFlags().StringVar(&cfgPath, "config-file", "", "configuration file")
	root.PersistentFlags().StringVar(&dir, "dir", "", "notes directory")
	root.PersistentFlags().StringVar(&editor, "editor", "", "editor command")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print structured JSON")
	get := func() (app.App, error) {
		c, e := config.Load(cfgPath, dir, editor)
		return app.App{Config: c, ConfigPath: cfgPath, DirectoryOverride: dir, EditorOverride: editor}, e
	}
	emit := func(v any) error { e := json.NewEncoder(out); e.SetIndent("", "  "); return e.Encode(v) }
	runTUI := func(_ *cobra.Command, _ []string) error {
		a, e := get()
		if e != nil {
			return e
		}
		return tui.Run(a)
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		f, ok := out.(*os.File)
		if ok && isatty.IsTerminal(f.Fd()) {
			return runTUI(cmd, args)
		}
		a, err := get()
		if err != nil {
			return err
		}
		all, err := a.Notes()
		if err != nil {
			return err
		}
		if jsonOutput {
			return emit(all)
		}
		for _, n := range all {
			fmt.Fprintf(out, "%s %s\n", n.ID, n.Title)
		}
		return nil
	}
	root.AddCommand(&cobra.Command{Use: "tui", Short: "Open the interactive explorer", Args: cobra.NoArgs, RunE: runTUI})
	root.AddCommand(&cobra.Command{Use: "pick", Short: "Interactively choose a note and print its ID", Args: cobra.NoArgs, RunE: func(_ *cobra.Command, _ []string) error {
		a, err := get()
		if err != nil {
			return err
		}
		id, err := tui.Pick(a, errout)
		if err != nil {
			return err
		}
		if jsonOutput {
			return emit(id)
		}
		_, err = fmt.Fprintln(out, id)
		return err
	}})
	add := func(use, short string, aliases []string, args cobra.PositionalArgs, f func(app.App, []string) error) {
		root.AddCommand(&cobra.Command{Use: use, Short: short, Aliases: aliases, Args: args, RunE: func(_ *cobra.Command, x []string) error {
			a, e := get()
			if e != nil {
				return e
			}
			return f(a, x)
		}})
	}
	add("add", "Create a note", []string{"a"}, cobra.NoArgs, func(a app.App, _ []string) error {
		n, e := a.NewNote()
		if e != nil {
			return e
		}
		e = a.RunEditor(n.Path)
		return a.FinishNew(n, e)
	})
	add("edit <id>", "Edit a note", []string{"e"}, cobra.ExactArgs(1), func(a app.App, x []string) error {
		n, e := a.Find(x[0])
		if e != nil {
			return e
		}
		if e = a.RunEditor(n.Path); e != nil {
			return e
		}
		return a.Commit(n.ID, "Edit note "+n.ID)
	})
	add("dir", "Print notes directory", []string{"d"}, cobra.NoArgs, func(a app.App, _ []string) error { fmt.Fprintln(out, a.Config.Notes.Directory); return nil })
	add("config", "Show effective configuration", nil, cobra.NoArgs, func(a app.App, _ []string) error { return emit(a.Config) })
	add("link <from-id> <to-id>", "Add a directed link between two notes", nil, cobra.ExactArgs(2), func(a app.App, x []string) error {
		if err := a.LinkNotes(x[0], x[1]); err != nil {
			return err
		}
		_, err := fmt.Fprintf(out, "Linked %s -> %s\n", x[0], x[1])
		return err
	})
	add("sources <id>", "List a note's source references", nil, cobra.ExactArgs(1), func(a app.App, x []string) error {
		n, err := a.Find(x[0])
		if err != nil {
			return err
		}
		if jsonOutput {
			return emit(append([]string{}, n.Sources...))
		}
		for _, source := range n.Sources {
			if _, err := fmt.Fprintln(out, source); err != nil {
				return err
			}
		}
		return nil
	})
	add("tags", "List unique note tags", nil, cobra.NoArgs, func(a app.App, _ []string) error {
		all, err := a.Notes()
		if err != nil {
			return err
		}
		unique := map[string]bool{}
		for _, n := range all {
			for _, tag := range n.Tags {
				unique[tag] = true
			}
		}
		tags := make([]string, 0, len(unique))
		for tag := range unique {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		if jsonOutput {
			return emit(tags)
		}
		for _, tag := range tags {
			if _, err := fmt.Fprintln(out, tag); err != nil {
				return err
			}
		}
		return nil
	})
	add("doctor", "Check notes directory", nil, cobra.NoArgs, func(a app.App, _ []string) error {
		_, e := os.Stat(a.Config.Notes.Directory)
		if e != nil {
			return fmt.Errorf("notes directory: %w", e)
		}
		fmt.Fprintln(out, "Notes directory accessible:", a.Config.Notes.Directory)
		return nil
	})
	add("git-status", "Show Git status", nil, cobra.NoArgs, func(a app.App, _ []string) error {
		cmd := exec.Command("git", "-C", a.Config.Notes.Directory, "status", "--short", "--branch")
		cmd.Stdout = out
		cmd.Stderr = errout
		return cmd.Run()
	})
	add("sync", "Push the current branch using its configured upstream", nil, cobra.NoArgs, func(a app.App, _ []string) error {
		if !a.Config.Git.Enabled {
			return fmt.Errorf("git integration is disabled")
		}
		cmd := exec.Command("git", "-C", a.Config.Notes.Directory, "push")
		cmd.Stdout = out
		cmd.Stderr = errout
		return cmd.Run()
	})
	for _, spec := range []struct {
		use, short string
		aliases    []string
	}{{"ids", "List note IDs", nil}, {"titles", "List titles", nil}, {"recent", "List recently modified notes", nil}, {"orphans", "List unconnected notes", nil}, {"broken", "List broken note links", nil}, {"graph", "Print relationships", nil}, {"stats", "Show note counts", nil}, {"check", "Check relationships and note metadata", nil}, {"review", "List notes for review", nil}} {
		s := spec
		add(s.use, s.short, s.aliases, cobra.NoArgs, func(a app.App, _ []string) error {
			all, e := a.Notes()
			if e != nil {
				return e
			}
			g := a.Graph(all)
			switch s.use {
			case "ids":
				if jsonOutput {
					return emit(allIDs(all))
				}
				for _, n := range all {
					fmt.Fprintln(out, n.ID)
				}
			case "titles", "review":
				if jsonOutput {
					return emit(all)
				}
				for _, n := range all {
					fmt.Fprintf(out, "%s %s\n", n.ID, n.Title)
				}
			case "recent":
				sort.SliceStable(all, func(i, j int) bool { return all[i].Modified.After(all[j].Modified) })
				if jsonOutput {
					return emit(all)
				}
				for _, n := range all {
					fmt.Fprintf(out, "%s %s\n", n.ID, n.Title)
				}
			case "orphans":
				if jsonOutput {
					return emit(g.Orphans)
				}
				for _, id := range g.Orphans {
					fmt.Fprintln(out, id)
				}
			case "broken":
				if jsonOutput {
					return emit(g.Broken)
				}
				for _, l := range g.Broken {
					fmt.Fprintf(out, "%s -> %s (%s)\n", l.From, l.To, l.Description)
				}
			case "graph":
				if jsonOutput {
					return emit(g)
				}
				for _, n := range all {
					for _, l := range g.Outgoing[n.ID] {
						fmt.Fprintf(out, "%s -> %s (%s)\n", l.From, l.To, l.Description)
					}
				}
			case "stats", "check":
				v := map[string]int{"notes": len(all), "orphans": len(g.Orphans), "broken": len(g.Broken)}
				if s.use == "check" {
					v["metadata_warnings"] = 0
					for _, n := range all {
						if n.MetadataWarning != "" {
							v["metadata_warnings"]++
						}
					}
				}
				if jsonOutput {
					return emit(v)
				}
				fmt.Fprintf(out, "Notes: %d\nOrphans: %d\nBroken links: %d\n", len(all), len(g.Orphans), len(g.Broken))
				if s.use == "check" {
					for _, n := range all {
						if n.MetadataWarning != "" {
							fmt.Fprintf(out, "%s: %s\n", n.ID, n.MetadataWarning)
						}
					}
				}
			}
			return nil
		})
	}
	for _, spec := range []struct {
		use, short string
		aliases    []string
	}{{"show <id>", "Print Markdown", nil}, {"title <id>", "Print title", nil}, {"body <id>", "Print body", nil}, {"text <id>", "Print note text", nil}, {"links <id>", "List outgoing links", nil}, {"backlinks <id>", "List incoming links", nil}, {"related <id>", "List connected notes", nil}, {"before <id>", "Find preceding note", nil}, {"after <id>", "Find following note", nil}} {
		s := spec
		add(s.use, s.short, s.aliases, cobra.ExactArgs(1), func(a app.App, x []string) error {
			n, e := a.Find(x[0])
			if e != nil {
				return e
			}
			all, e := a.Notes()
			if e != nil {
				return e
			}
			g := a.Graph(all)
			name := strings.Fields(s.use)[0]
			switch name {
			case "show", "text":
				if jsonOutput {
					return emit(n)
				}
				fmt.Fprint(out, n.Body)
			case "title":
				fmt.Fprintln(out, n.Title)
			case "body":
				body := n.Body
				if strings.HasPrefix(body, "# ") {
					_, body, _ = strings.Cut(body, "\n")
					body = strings.TrimPrefix(body, "\n")
				}
				fmt.Fprint(out, body)
			case "before", "after":
				for i, v := range all {
					if v.ID == n.ID {
						if name == "before" && i > 0 {
							fmt.Fprintln(out, all[i-1].ID)
						}
						if name == "after" && i+1 < len(all) {
							fmt.Fprintln(out, all[i+1].ID)
						}
						break
					}
				}
			default:
				ls := g.Outgoing[n.ID]
				if name == "backlinks" {
					ls = g.Incoming[n.ID]
				}
				if name == "related" {
					ls = append(append([]notes.Link{}, ls...), g.Incoming[n.ID]...)
				}
				if jsonOutput {
					return emit(ls)
				}
				for _, l := range ls {
					fmt.Fprintf(out, "%s -> %s  %s\n", l.From, l.To, l.Description)
				}
			}
			return nil
		})
	}
	var searchType, searchTag string
	var searchSnippets bool
	searchCmd := &cobra.Command{Use: "search <query>", Short: "Search notes", Aliases: []string{"query", "q"}, Args: cobra.MinimumNArgs(1), RunE: func(_ *cobra.Command, x []string) error {
		if searchType != "" && searchType != string(notes.Fleeting) && searchType != string(notes.Literature) && searchType != string(notes.Permanent) {
			return fmt.Errorf("unknown note type %q (choose fleeting, literature, or permanent)", searchType)
		}
		a, e := get()
		if e != nil {
			return e
		}
		all, e := a.Notes()
		if e != nil {
			return e
		}
		filtered := make([]notes.Note, 0, len(all))
		for _, n := range all {
			if searchType != "" && string(n.Type) != searchType {
				continue
			}
			if searchTag != "" {
				found := false
				for _, tag := range n.Tags {
					if tag == searchTag {
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
		query := strings.Join(x, " ")
		hits := notes.SearchWithOptions(filtered, query, a.Config.Search.CaseSensitive, a.Config.Search.MaxResults)
		if jsonOutput {
			return emit(hits)
		}
		for _, n := range hits {
			if _, err := fmt.Fprintf(out, "%s %s\n", n.ID, n.Title); err != nil {
				return err
			}
			if searchSnippets {
				if snippet := notes.SearchSnippet(n.Body, query, a.Config.Search.CaseSensitive, 120); snippet != "" {
					if _, err := fmt.Fprintf(out, "  %s\n", snippet); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}}
	searchCmd.Flags().StringVar(&searchType, "type", "", "filter by note type (fleeting, literature, permanent)")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "filter by exact, case-sensitive tag")
	searchCmd.Flags().BoolVar(&searchSnippets, "snippets", false, "show an excerpt for body matches")
	root.AddCommand(searchCmd)
	add("last", "Show newest note by ID", nil, cobra.NoArgs, func(a app.App, _ []string) error {
		all, e := a.Notes()
		if e != nil {
			return e
		}
		if len(all) > 0 {
			n := all[len(all)-1]
			fmt.Fprintf(out, "%s %s\n", n.ID, n.Title)
		}
		return nil
	})
	root.AddCommand(&cobra.Command{Use: "id", Short: "Print current UTC timestamp ID", Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { fmt.Fprintln(out, time.Now().UTC().Format("20060102150405")) }})
	root.AddCommand(&cobra.Command{Use: "urlencode [text...]", Short: "URL-encode arguments or lines from standard input", RunE: func(cmd *cobra.Command, x []string) error {
		write := func(s string) error {
			_, err := fmt.Fprintln(out, strings.ReplaceAll(url.QueryEscape(s), "+", "%20"))
			return err
		}
		if len(x) > 0 {
			return write(strings.Join(x, " "))
		}
		reader := bufio.NewReader(cmd.InOrStdin())
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				if writeErr := write(strings.TrimSuffix(line, "\n")); writeErr != nil {
					return fmt.Errorf("write encoded text: %w", writeErr)
				}
			}
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("read text to encode: %w", err)
			}
		}
	}})
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print version", Args: cobra.NoArgs, Run: func(*cobra.Command, []string) { fmt.Fprintln(out, Version) }})
	root.AddCommand(&cobra.Command{Use: "usage", Short: "Show usage", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error { return root.Help() }})
	return root
}
func allIDs(all []notes.Note) []string {
	ids := make([]string, 0, len(all))
	for _, n := range all {
		ids = append(ids, n.ID)
	}
	return ids
}
