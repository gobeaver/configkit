# Credits

## Source
- Package: github.com/joho/godotenv
- Version: v1.5.1
- License: MIT (see LICENCE)
- Upstream: https://github.com/joho/godotenv

## Update History
| Date       | Version | Reviewer | Notes          |
|------------|---------|----------|----------------|
| 2026-04-06 | v1.5.1  | @basheer | Initial vendor |

## Vendoring Notes
- Vendored via `go mod download github.com/joho/godotenv@v1.5.1`, then root-level `*.go` files copied (excluding `*_test.go`, `cmd/`, `autoload/`, `fixtures/`).
- Files included: `godotenv.go`, `parser.go`.
- Package name renamed from `godotenv` to `dotenv` to match the local directory and beaver/configkit naming convention. This was the only modification to the upstream source.
- No external dependencies (pure stdlib).

## Upstream Changes Reviewed
- v1.5.1: initial vendor, full source reviewed.
