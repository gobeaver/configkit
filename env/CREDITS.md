# Credits

## Source
- Package: github.com/caarlos0/env/v11
- Version: v11.4.0
- License: MIT (see LICENSE.md)
- Upstream: https://github.com/caarlos0/env

## Update History
| Date       | Version  | Reviewer | Notes          |
|------------|----------|----------|----------------|
| 2026-04-06 | v11.4.0  | @basheer | Initial vendor |

## Vendoring Notes
- Vendored via `go mod download github.com/caarlos0/env/v11@v11.4.0`, then root-level `*.go` files copied (excluding `*_test.go` and `example_*.go`).
- Files included: `env.go`, `env_tomap.go`, `env_tomap_windows.go`, `error.go`.
- No external dependencies (pure stdlib).
- No internal subpackages — no import path rewriting required.

## Upstream Changes Reviewed
- v11.4.0: initial vendor, full source reviewed.
