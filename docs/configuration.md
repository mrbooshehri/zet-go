# Configuration

The default file is `zet/config.yaml` under the platform user configuration directory. Use `--config-file` to choose another file; an explicitly named missing file is an error. Unknown YAML keys are rejected. Note directory precedence is `--dir`, `ZETDIR`, `KN`, YAML, then `~/zet`. Editor precedence is `--editor`, YAML, `VISUAL`, `EDITOR`, then `vi`. Commands are parsed into arguments without shell expansion.

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

Only the shown filename, ID format and layout are supported. Theme values `auto`, `dark`, and `light` control the Markdown renderer; the surrounding TUI uses adaptive terminal colors. `max_results: 0` means unlimited. Auto-push requires Git and auto-commit enabled. `zet config` prints effective configuration.

In the TUI, press `S` to inspect effective values and `e` to edit the YAML in the configured external editor. ZET validates a temporary draft before saving, refuses to overwrite a concurrently changed or symlinked configuration, and preserves the draft on failure. The settings view includes environment and command-line overrides; editing changes the YAML file, so overrides may continue to take precedence. An unchanged draft does not create or rewrite a configuration file.
