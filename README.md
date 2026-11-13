# ZET

ZET is a local-first, terminal-based Zettelkasten for capturing ideas and connecting notes. Your knowledge stays in ordinary Markdown files: there is no database, account, cloud service, or required Git repository.

This Go rewrite is **under development**. It can open existing ZET notes directly and provides a CLI and keyboard-driven TUI over the same note collection. See [current limitations](#current-limitations) before relying on it for a workflow.

## Features

- Create and edit timestamped Markdown notes in your preferred editor.
- Browse, sort, search, filter, and read notes in a responsive terminal interface.
- Connect notes with Markdown or wiki links; inspect backlinks, broken links, orphans, and a textual graph.
- Add optional note types, tags, and source references in YAML frontmatter.
- Select a note interactively for use in shell commands with `zet pick`.
- Configure optional Git commits and pushes, with no Git requirement for normal use.
- Keep files portable and rebuild search and relationship data from Markdown.

## Install

Requirements: Go 1.26 or later and an editor. Git is needed only for Git features.

From this repository:

```sh
go install ./cmd/zet
```

Ensure `$(go env GOPATH)/bin` is on your `PATH`. Alternatively, `make build` creates `bin/zet`; run that path directly if you prefer not to install.

## Quick start

```sh
# Optional: the default is ~/zet. Existing ZET note directories work directly.
export ZETDIR="$HOME/zet"
export EDITOR=vim

zet add                      # create a note and open the editor
zet                         # open the TUI in a terminal
zet search "an idea"         # search from the command line
zet titles                  # list note IDs and titles
```

New notes get a unique UTC timestamp ID and a template with an automatically generated creation time. Write a title after `# `, then add your thoughts. Notes with no type or frontmatter remain valid. The first creation makes the notes directory if necessary.

```text
~/zet/
├── 20261113102030/
│   └── README.md
└── 20261113103512/
    └── README.md
```

An untouched generated template is removed after a successful editor session only if the original new file remains in place and its directory contains no other files. Edited or replaced files, and notes from failed editor sessions, are preserved.

## Note format and metadata

The generated template includes empty Tags, Related, and Sources sections. The timestamp below is illustrative; ZET supplies the actual time automatically.

```markdown
---
created: "2026-11-13T10:20:30Z"
type: permanent
tags: [zettelkasten, writing]
sources:
  - "A book, chapter 3"
  - "https://example.com/article"
---
# One idea per note

An atomic note is easier to connect to other ideas.

## Tags

## Related

[A related idea](../20261113103512/README.md)

## Sources
```

`type` is optional and may be `fleeting`, `literature`, or `permanent`. Tags and sources are optional lists of strings. **Put tags and sources in the frontmatter** for `zet tags`, `zet sources`, and tag filtering; the Markdown sections are available for your own notes. Tags are case-sensitive. Edit metadata in your editor. `zet check` reports unknown types and invalid metadata without rewriting the note.

The note's Markdown remains authoritative. Existing notes are not renamed or migrated, and indexing does not change them. See the [note format](docs/note-format.md) and [migration guide](docs/migration.md).

## Terminal interface

Run `zet` or `zet tui` in an interactive terminal. On a wide terminal the note list and Glamour-rendered Markdown preview appear side by side. On a narrow terminal, Tab switches between full-width list and preview. Focus is indicated by a thicker border; the current note is highlighted.

```text
 ZET  Notes · Newest first
╭─ Search ─────────────────────────────────────────────────────╮
╰──────────────────────────────────────────────────────────────╯
┏━ Notes ━━━━━━━━━━━━━━━━━┓╭─ Preview ─────────────────────────╮
┃ › 20261113103512  Idea  ┃│ A rendered Markdown note         │
┃   20261113102030  Note  ┃│                                  │
┗━━━━━━━━━━━━━━━━━━━━━━━━━┛╰──────────────────────────────────╯
 Tab Focus  v Read  L Link  / Search  t Type  # Tag  ? Help
```

This is a schematic, not a screenshot. Exact layout depends on terminal size and settings.

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Move in a list or scroll the focused preview |
| `Tab` | Switch focus between the list and preview |
| `Enter` / `v` | Open full-terminal reading mode; `v` or Escape returns |
| `/` | Focus search; Enter or Escape leaves search input |
| `t` / `#` | Cycle note-type / tag filters outside search |
| `s` | Cycle newest, oldest, title, and last-modified sorting |
| `n` / `e` | Create / edit a note |
| `L` | Choose a target and add a link from the selected note |
| `l` / `b` / `g` | View outgoing links / backlinks / graph |
| `o` / `r` | Show orphans / the basic review list |
| `S` | Show settings; press `e` there to edit them |
| `?` / `Esc` | Open / close help, or return from a view |
| `q` / `Ctrl+C` | Quit |

In relationship views, select a link and press Enter to open the related note; Escape returns to the previous view. In search input, keys are text or editing shortcuts rather than global actions: spaces, Ctrl+W, Ctrl+A/E, Ctrl+U/K, and cursor arrows work. See the [TUI guide](docs/tui-guide.md).

## Search and select

Search matches IDs, titles, and Markdown bodies. It is case-insensitive by default and ranks ID and title matches above body matches. The TUI filters incrementally as you type.

```sh
zet search "atomic notes"
zet q "atomic notes" --snippets
zet search "writing" --type permanent --tag zettelkasten
zet --json search "writing" --tag zettelkasten
```

`--snippets` adds an excerpt for body matches to plain-text results. Type and exact, case-sensitive tag filters can be combined and are applied before the configured result limit. `query` and `q` are aliases for `search`. In the TUI, `t` and `#` cycle those filters; Escape outside search clears them.

Choose a note interactively and pass its ID to another command:

```sh
zet pick
zet show "$(zet pick)"
```

`pick` renders its interface on the controlling terminal via standard error and writes only the selected ID to standard output. It needs a controlling terminal, even when output is piped. Enter selects; Escape or `q` cancels. Cancellation and an empty collection return an error without printing an ID.

## Connect notes

Write a relative Markdown link or wiki reference in a note:

```markdown
[A related idea](../20261113103512/README.md)
[[20261113103512]]
```

Alternatively, select the source note in the TUI, press uppercase `L`, and choose a target. Press `/` in the picker to search, Enter to finish searching, and Enter again to create the link. You can also use:

```sh
zet link 20261113102030 20261113103512
zet links 20261113102030
zet backlinks 20261113103512
zet related 20261113102030
zet graph
zet broken
zet orphans
```

Links are directed: the source gains an outgoing link and the target gains a backlink. The link action rejects self-links and existing relationships. It appends Markdown without rewriting existing text; if another section follows Related, it may append another Related heading. It requires Markdown links enabled and does not automatically commit or push. The current graph recognizes the note-link formats shown above and ignores links inside code. See [linking](docs/linking.md).

## CLI reference

| Command | Purpose |
| --- | --- |
| `zet`, `zet tui` | Open the TUI; plain `zet` lists notes when stdout is not a terminal |
| `zet add` / `a`; `zet edit <id>` / `e` | Create or edit with an external editor |
| `zet pick` | Interactively select and print a note ID |
| `zet show <id>`; `text <id>`; `title <id>`; `body <id>` | Read note content or its title |
| `zet ids`; `titles`; `last`; `before <id>`; `after <id>`; `recent` | List or navigate notes |
| `zet search <query>` / `query` / `q` | Search, optionally with `--type`, `--tag`, and `--snippets` |
| `zet link <from-id> <to-id>`; `links <id>`; `backlinks <id>`; `related <id>` | Create or inspect relationships |
| `zet graph`; `broken`; `orphans`; `check`; `stats`; `review` | Inspect the graph, metadata, and basic review list |
| `zet tags`; `sources <id>` | List indexed metadata |
| `zet config`; `doctor`; `dir` / `d` | Inspect configuration or the notes directory |
| `zet git-status`; `sync` | Show Git status or explicitly push the current branch |
| `zet id`; `urlencode [text...]`; `version`; `usage` | Generate a timestamp ID, URL-encode text, or show version / usage |

`urlencode` reads lines from standard input when no argument is supplied. Run `zet --help` or `zet <command> --help` for command-specific help. Supported read-only commands offer `--json`; for example:

```sh
zet --json show 20261113102030
zet --json tags
zet --json check
printf 'hello world\n' | zet urlencode
```

`review` currently lists notes; it is not a spaced-repetition scheduler. `text` outputs Markdown, and `body` removes a leading first-line heading when present. See the [CLI reference](docs/cli-reference.md) and [Bash compatibility notes](docs/compatibility.md).

## Configuration and settings

The default config file is `zet/config.yaml` under the platform user configuration directory, such as `$XDG_CONFIG_HOME/zet/config.yaml` on Unix. Configuration is optional:

```yaml
notes:
  directory: ~/zet
  filename: README.md
  id_format: timestamp
editor:
  command: vim
tui:
  theme: auto
  preview: true
  layout: adaptive
git:
  enabled: false
  auto_commit: false
  auto_push: false
search:
  case_sensitive: false
  max_results: 100
links:
  wiki_links: true
  markdown_links: true
```

Directory precedence: `--dir` → `ZETDIR` → `KN` → YAML → `~/zet`. Editor precedence: `--editor` → YAML → `VISUAL` → `EDITOR` → `vi`. `--config-file` selects a different YAML file. Editor commands are split into arguments without shell interpolation.

In the TUI, press `S` to see effective settings, then `e` to edit the YAML in your configured editor. ZET validates a temporary draft before saving; a failed edit or validation leaves the draft available and does not replace the original. Environment variables and flags can continue to override saved values. `zet config` prints effective settings. See [configuration](docs/configuration.md).

## Optional Git workflow

Git is disabled by default. To enable automatic commits after creation or editing, set `git.enabled` and `git.auto_commit` to `true` in the configuration. Set `auto_push: true` only if you also want automatic pushes; it requires both other Git options enabled. Configure your repository's Git identity and upstream using Git itself.

```sh
zet git-status    # inspect the notes repository
zet sync          # explicitly push the current branch to its configured upstream
```

`sync` pushes; it does not pull. Automatic commits target the note being changed and refuse to replace an already staged version of that note. Unrelated staged files are not included. If a push fails after a successful commit, the note is already committed locally; correct the remote or upstream and retry with `zet sync`. The `zet link` action leaves its change for you to commit separately. See [troubleshooting](docs/troubleshooting.md).

## Development and documentation

```sh
make build
make test
make test-race
make vet
make fmt
make lint       # requires golangci-lint
make snapshot   # requires GoReleaser
```

See [development](DEVELOPMENT.md), [contributing](CONTRIBUTING.md), [architecture](ARCHITECTURE.md), [design](DESIGN.md), [security](SECURITY.md), and the [documentation directory](docs/). Tests use temporary directories and Git repositories, not your notes.

## Current limitations

- The planned dashboard and three-panel navigation are not yet implemented; relationship and review workflows remain basic.
- Type selection during creation and customizable templates are not yet available.
- Search highlighting and TUI snippets are not yet available.
- Markdown relationship parsing is limited to the formats documented above. Creating a link may append a second Related heading.
- Git, editor, terminal, and release workflows need broader cross-platform runtime testing and release-tool validation.
- The original script's `query`, `last`, `before`, `text`, and `body` behaviors differ in this rewrite; see [compatibility](docs/compatibility.md).
- The owner has confirmed ownership of the original script. A license for this rewrite has not yet been selected.
