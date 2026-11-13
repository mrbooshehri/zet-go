# Security

ZET processes local Markdown as untrusted input. Note IDs are validated and note symlinks are skipped; note reads must never modify source files. Editor and Git commands run as argument vectors, not shell strings. Git must not force-push or stage unrelated files. Terminal output should strip terminal control sequences from note content. The program is not a sandbox against a malicious process concurrently changing files in the notes directory. Review dependency updates and report vulnerabilities privately to the repository owner.
