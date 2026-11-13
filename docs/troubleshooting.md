# Troubleshooting

If an editor cannot start, check `zet config`, `VISUAL`, `EDITOR`, and executable permissions. If Git commits fail, verify Git identity and repository status. An already staged version of the note is intentionally left alone; commit it manually. A failed editor retains the new file so saved content is not discarded; the error identifies its path, and the TUI refreshes its note list. `zet doctor` checks access to the configured directory.

If automatic push fails after a successful commit, the note is already committed locally. Configure the intended remote and upstream, then use `zet sync` to retry the push; there is no need to recreate the note or repeat the commit.

If a settings edit fails validation or the configuration changed while the editor was open, the original configuration is not replaced. The error identifies the preserved draft so you can recover your edits. `zet pick` needs a controlling terminal, including when its standard output is piped.
