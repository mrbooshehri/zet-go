# CLI reference

Run `zet --help` and `zet <command> --help` for the current command list. `--dir` selects a notes directory, `--config-file` selects YAML, and `--editor` selects an editor. Supported structured read-only commands accept `--json`. `zet query` and `zet q` search locally, unlike the original script's GitHub search URL. `zet git-status` reports Git state and `zet sync` explicitly pushes to the configured upstream when Git integration is enabled.

`zet urlencode 'text to encode'` encodes its arguments; without arguments it reads and encodes each line from standard input, for example `printf 'hello world\n' | zet urlencode`.

`zet tags` lists unique tags from note frontmatter. Add `--json` for a JSON array.

`zet sources <id>` lists a note's source references in their original order. Add `--json` for a JSON array.

`zet link <source-id> <target-id>` appends a directed Markdown relationship to the source note. It rejects self-links and duplicates. See the linking guide for behavior and Git considerations.

`zet pick` opens an interactive selector on the controlling terminal and prints only the selected ID to standard output. Use `/` to search, `j`/`k` to move, Enter to select, and Escape or `q` to cancel. For example, `zet show "$(zet pick)"`. The selector is rendered on standard error so command substitution captures only the ID. It requires a controlling terminal; cancellation or an empty note collection returns an error and prints no ID.

Search can be restricted with `zet search idea --type permanent --tag notes`. Both filters are optional and can be combined. Types must be `fleeting`, `literature`, or `permanent`; tags match exactly and are case-sensitive. The filters apply before the configured search result limit. `query` and `q` accept the same flags. The TUI also offers type and tag filters; see the TUI guide for shortcuts.

Add `--snippets` to show a short excerpt around the first body match under each result, for example `zet search idea --snippets`. Whitespace is condensed, and ellipses indicate truncated context. Results matching only the ID or title have no excerpt. Default plain-text output remains one line per note; JSON output remains the note data and does not include snippets.
