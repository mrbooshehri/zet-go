# Development

Install Go 1.26 and Git. Run `make build`, `make test`, `make test-race`, `make vet`, and `make fmt`. `make lint` requires golangci-lint; `make snapshot` requires GoReleaser. `go run ./cmd/zet --help` runs the CLI without installation. Tests use temporary directories and temporary Git repositories; do not point tests at personal notes.

Use a terminal to exercise the TUI and resize it while checking the list, preview, search focus and editor suspend/resume. CI runs formatting, vet, tests and build. Release automation is configured for tagged builds, but no release has been published or verified.
