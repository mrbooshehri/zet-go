# Bash compatibility baseline

Inspected `/home/mhmd/Repos/github.com/mrbooshehri/dot/scripts/zet` on 2026-11-13. The script requires `GITUSER` for every command; chooses existing `$KN` over `$ZETDIR` despite its comment; generates UTC `YYYYMMDDHHMMSS` IDs; discovers directories beginning with `20`; reads `README.md`; and treats only a first-line `# ` heading as a title. It uses Pandoc to render titles, bodies and full text. `query` emits a GitHub search link, rather than searching local files. `last` emits a hard-coded GitHub link. `before` includes the given ID; `after` seeks a greater one. `add` invokes Vim, deletes a newly created empty directory, and automatically stages, commits, and pushes. `edit` invokes Vim without committing. `urlencode` accepts stdin when no arguments are supplied.

Known pitfalls include potentially colliding IDs, unconditional network operations, shell/path handling, use of Pandoc without checking it in some commands, and surprising `KN` precedence. Compatibility decisions must be explicit and tested; avoiding silent data loss takes precedence over reproducing destructive behavior.

## Current Go behavior

`ZETDIR` takes precedence over `KN`, and neither `GITUSER` nor Pandoc is required. `query`/`q` perform local search; `text` and `body` return Markdown rather than Pandoc-rendered plain text. `last` prints the newest ID and title instead of a hard-coded GitHub URL. `before` selects the strictly preceding note. Adding a note does not commit or push unless explicitly configured. Editor selection follows the documented configuration precedence. `urlencode` accepts arguments or reads one line at a time from standard input; Unicode is encoded as UTF-8 bytes.
