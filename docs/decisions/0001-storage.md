# ADR 0001: Markdown filesystem as source of truth

## Context
Existing notes are ordinary Markdown files inside timestamp directories.

## Decision
Read these files directly and derive search and links in memory without modifying notes.

## Alternatives considered
A mandatory database would complicate migration and data ownership. A persistent index requires cache invalidation.

## Consequences
Existing files remain portable and an index can always be rebuilt. Large collections may eventually need a disposable index.
