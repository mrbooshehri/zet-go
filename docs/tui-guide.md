# TUI guide

Run `zet` in an interactive terminal. The search bar remains visible; press `/` to focus it, type to filter, and press Escape or Enter to leave it. Keys typed while searching do not execute shortcuts. Use `j` and `k` or arrow keys to select a note, and Enter to open its rendered Markdown. Escape returns to the list and clears the search.

Search accepts spaces and shell-style editing shortcuts, including Ctrl+W to delete the previous word, Ctrl+A/E to move to the start/end, and Ctrl+U/K to delete text before/after the cursor. Left and right arrows also move the cursor while search is focused.

Press Tab to switch focus between the list and preview. The focused panel has a thicker, highlighted border. In the preview, `j`/`k` and down/up scroll the rendered note; when the list is focused, they change the selected note. On narrow terminals or when the preview setting is disabled, Tab switches between full-width list and preview. The full note detail also scrolls with those keys.

Press `v` or Enter to open the selected note in full-width reading mode. Use `j`/`k` or the arrow keys to scroll, and `v` or Escape to return without losing the current search or filters. Press uppercase `L` on a selected note to choose a note to link; use `/` in the picker to search targets, Enter to finish searching and Enter again to create the link, or Escape to cancel.

In the backlinks, links, or graph view, use `j`/`k` or the arrow keys to select a relationship and Enter to open that note. Escape returns to the relationship and restores the previous search and selection; Escape again returns to the source view. Missing targets show an error without leaving the list.

Press uppercase `S` to view effective settings, then `e` to edit the configuration using your configured external editor. ZET edits a temporary draft, validates it, and saves it only if valid and the original has not changed. On failure, the draft is kept and its path is included in the error. Escape returns to the previous screen. The screen displays effective values, including environment and command-line overrides; the editor opens the configuration file itself.

The list defaults to newest note ID first. Press `s` to cycle through newest, oldest, title, and last-modified order; the current order appears in the header. `n` creates and `e` edits through an external editor. `b`, `l`, `g`, `o`, and `r` open backlinks, links, graph, orphan and review views. `?` opens a floating help panel; Escape or `?` closes it. `q` quits. On narrow terminals the preview appears separately when focused.

Outside the search field, press `t` to cycle through all types, fleeting, literature, and permanent. Press `#` to cycle through all tags and the available tags in sorted order. Active filters appear in the search bar and combine with the search text; tags match exactly, including case. Press Escape outside search to clear the search and both filters. While typing in search, `t` and `#` enter text instead.
