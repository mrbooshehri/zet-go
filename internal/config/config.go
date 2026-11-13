package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Notes struct {
		Directory string `yaml:"directory"`
		Filename  string `yaml:"filename"`
		IDFormat  string `yaml:"id_format"`
	} `yaml:"notes"`
	Editor struct {
		Command string `yaml:"command"`
	} `yaml:"editor"`
	Git struct {
		Enabled    bool `yaml:"enabled"`
		AutoCommit bool `yaml:"auto_commit"`
		AutoPush   bool `yaml:"auto_push"`
	} `yaml:"git"`
	TUI struct {
		Theme   string `yaml:"theme"`
		Preview bool   `yaml:"preview"`
		Layout  string `yaml:"layout"`
	} `yaml:"tui"`
	Search struct {
		CaseSensitive bool `yaml:"case_sensitive"`
		MaxResults    int  `yaml:"max_results"`
	} `yaml:"search"`
	Links struct {
		WikiLinks     bool `yaml:"wiki_links"`
		MarkdownLinks bool `yaml:"markdown_links"`
	} `yaml:"links"`
}

func Load(path, directory, editor string) (Config, error) {
	var c Config
	home, err := os.UserHomeDir()
	if err != nil {
		return c, err
	}
	c.Notes.Directory = filepath.Join(home, "zet")
	c.Notes.Filename = "README.md"
	c.Notes.IDFormat = "timestamp"
	c.TUI.Theme = "auto"
	c.TUI.Preview = true
	c.TUI.Layout = "adaptive"
	c.Search.MaxResults = 100
	c.Links.WikiLinks = true
	c.Links.MarkdownLinks = true
	explicit := path != ""
	if !explicit {
		dir, err := os.UserConfigDir()
		if err != nil {
			return c, err
		}
		path = filepath.Join(dir, "zet", "config.yaml")
	}
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		dec := yaml.NewDecoder(f)
		dec.KnownFields(true)
		if err := dec.Decode(&c); err != nil {
			return c, fmt.Errorf("configuration %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) || explicit {
		return c, fmt.Errorf("configuration %s: %w", path, err)
	}
	if v := os.Getenv("KN"); v != "" {
		c.Notes.Directory = v
	}
	if v := os.Getenv("ZETDIR"); v != "" {
		c.Notes.Directory = v
	}
	if directory != "" {
		c.Notes.Directory = directory
	}
	if editor != "" {
		c.Editor.Command = editor
	}
	if c.Editor.Command == "" {
		c.Editor.Command = os.Getenv("VISUAL")
	}
	if c.Editor.Command == "" {
		c.Editor.Command = os.Getenv("EDITOR")
	}
	if c.Editor.Command == "" {
		c.Editor.Command = "vi"
	}
	if strings.HasPrefix(c.Notes.Directory, "~/") {
		c.Notes.Directory = filepath.Join(home, c.Notes.Directory[2:])
	}
	if c.Notes.Directory == "" {
		return c, fmt.Errorf("notes directory cannot be empty")
	}
	c.Notes.Directory, err = filepath.Abs(c.Notes.Directory)
	if err != nil {
		return c, err
	}
	if c.Git.AutoPush && (!c.Git.Enabled || !c.Git.AutoCommit) {
		return c, fmt.Errorf("git auto_push requires enabled and auto_commit")
	}
	if c.Notes.Filename != "README.md" || c.Notes.IDFormat != "timestamp" {
		return c, fmt.Errorf("only README.md and timestamp IDs are supported")
	}
	if c.Search.MaxResults < 0 {
		return c, fmt.Errorf("search max_results must be nonnegative")
	}
	if c.TUI.Theme != "auto" && c.TUI.Theme != "dark" && c.TUI.Theme != "light" {
		return c, fmt.Errorf("invalid tui theme %q", c.TUI.Theme)
	}
	if c.TUI.Layout != "adaptive" {
		return c, fmt.Errorf("unsupported tui layout %q", c.TUI.Layout)
	}
	return c, nil
}
