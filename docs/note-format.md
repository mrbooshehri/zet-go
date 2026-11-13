# Note format

Notes live at `<notes-directory>/<14-digit UTC timestamp>/README.md`. A first line beginning with `# ` supplies the displayed title; otherwise the note is shown as untitled. The original Markdown is retained, and frontmatter is optional.

New notes start with a template containing an automatically generated UTC `created` timestamp (RFC 3339), empty `tags` and `sources` frontmatter lists, a blank Markdown title, and Tags, Related, and Sources sections. Enter the title after `# ` and add your content. Add indexed tags and source references to the frontmatter lists; the corresponding Markdown sections are available for your own notes. Put note links under Related to include them in the relationship graph. Existing notes are not changed. If the editor exits successfully without changing the original template, ZET removes that newly created note only when its original file and directory are still in place and contain no other files. Changed or replaced files and notes from failed editor sessions are preserved.

To classify a note, add YAML frontmatter at the start of its Markdown:

```markdown
---
type: permanent
---
# An insight

Write the note here.
```

Supported values are `fleeting`, `literature`, and `permanent` (lowercase). Other YAML fields are left untouched. A heading immediately after valid frontmatter, with optional blank lines, supplies the title. The type appears in `zet show --json` and the TUI preview. Use `zet check` to find unknown types or malformed YAML; such notes remain readable and are never rewritten. An opening `---` without a closing delimiter is treated as ordinary Markdown. Links in valid frontmatter are not counted as note links. Creating or changing types is currently done in the external editor; there is no type-selection command yet.

Tags may be included in the same frontmatter as a YAML list of non-empty strings, for example `tags: [go, notes]` or an equivalent block list. Tags are case-sensitive; leading and trailing whitespace is removed and duplicate tags on a note are ignored. `zet tags` lists unique tags in sorted order, and `zet tags --json` prints a JSON array. A note's tags are included in `zet show --json`. Invalid tag entries are omitted and reported by `zet check`. Editing tags is done in the external editor; use `zet search <query> --tag <tag>` to filter CLI search results.

Sources can be recorded as a YAML list of non-empty strings, such as `sources: ["Book, chapter 3", "https://example.com/paper"]`. These are plain references; ZET does not fetch or validate URLs. Leading and trailing whitespace is removed, and source order and duplicates are retained. `zet sources <id>` lists the references; add `--json` for a JSON array. They are also included in `zet show --json`. Invalid source entries are omitted and reported by `zet check`. Edit sources in the external editor.
