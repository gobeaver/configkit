# configkit

A thin Go config loader for Beaver projects. Wraps vendored copies of
[`caarlos0/env`](https://github.com/caarlos0/env) and
[`joho/godotenv`](https://github.com/joho/godotenv) — **zero external runtime
dependencies**.

```go
import "github.com/gobeaver/beaver/configkit"
```

## Why

- **Vendored, audited deps.** No surprise upstream updates. Every version bump
  is reviewed line-by-line and recorded in `env/CREDITS.md` and
  `dotenv/CREDITS.md`.
- **Sensible defaults.** Loads `.env` automatically, applies a `BEAVER_` prefix.
- **All caarlos0/env features.** Slices, maps, nested structs, required fields,
  custom unmarshalers, env var expansion.
- **Multi-instance via prefix.** Run multiple configured instances of the same
  package side-by-side without YAML.

## Quick start

```go
package main

import (
    "fmt"
    "github.com/gobeaver/beaver/configkit"
)

type Config struct {
    Host string `env:"HOST" envDefault:"localhost"`
    Port int    `env:"PORT" envDefault:"8080"`
}

func main() {
    var cfg Config
    if err := configkit.Load(&cfg); err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", cfg)
}
```

```bash
# .env
BEAVER_HOST=example.com
BEAVER_PORT=3000
```

## API

| Function | Purpose |
|---|---|
| `configkit.Load(&cfg, opts...) error` | Parse env into a struct |
| `configkit.MustLoad(&cfg, opts...)` | Same, panics on error (panic value is an `error`) |
| `configkit.WithPrefix(p string)` | Override the default `BEAVER_` prefix (use `""` for none) |
| `configkit.WithEnvFiles(files ...string)` | Choose which `.env` files to load |
| `configkit.WithoutDotEnv()` | Skip `.env` loading entirely |
| `configkit.WithRequired()` | Treat all undefaulted fields as required |

### .env semantics

- A **missing** `.env` is silently ignored (the common case where projects don't ship one).
- A **malformed** `.env` returns an error from `Load` rather than being swallowed.
- With multiple files, **first-wins**: process env always takes precedence, then earlier files override later ones. To make `.env.local` win over `.env`, list it first:
  ```go
  configkit.Load(&cfg, configkit.WithEnvFiles(".env.local", ".env"))
  ```

## Multi-instance pattern

```bash
DEV_SLACK_WEBHOOK_URL=https://hooks.slack.com/dev
PROD_SLACK_WEBHOOK_URL=https://hooks.slack.com/prod

PRIMARY_DB_HOST=primary.db.example.com
REPLICA_DB_HOST=replica.db.example.com
```

```go
devSlack    := slack.WithPrefix("DEV_").New()
prodSlack   := slack.WithPrefix("PROD_").New()

primaryDB   := database.WithPrefix("PRIMARY_").Init()
replicaDB   := database.WithPrefix("REPLICA_").Init()
```

## Supported field types

Anything `caarlos0/env` supports:

```go
type Config struct {
    Str      string        `env:"STR"`
    Int      int           `env:"INT"`
    Bool     bool          `env:"BOOL"`
    Duration time.Duration `env:"DURATION"`

    Hosts    []string          `env:"HOSTS" envSeparator:","`
    Ports    []int             `env:"PORTS" envSeparator:","`
    Metadata map[string]string `env:"METADATA"`

    APIKey string `env:"API_KEY,required"`
    Port   int    `env:"PORT" envDefault:"8080"`

    Database DatabaseConfig `envPrefix:"DB_"`
}
```

Combined with the package's default `BEAVER_` prefix and the nested struct's
`envPrefix:"DB_"`, the database fields would read from `BEAVER_DB_*`.

## Layout

```
configkit/
├── config.go            # Public API: Load, MustLoad, options
├── doc.go               # Package godoc
├── config_test.go
├── env/                 # Vendored caarlos0/env (see env/CREDITS.md)
├── dotenv/              # Vendored joho/godotenv (see dotenv/CREDITS.md)
└── docs/
    └── proposal.md      # Original design proposal (historical)
```

## Vendor update process

1. **Wait.** Don't update unless there's a security fix or a feature you need.
   For features, lag 3–6 months behind upstream.
2. **Diff.** `git clone` the upstream repo and review every changed line
   between the current vendored tag and the target tag. Confirm no new
   external dependencies were introduced.
3. **Refresh.**
   ```bash
   go mod download github.com/caarlos0/env/v11@<new-version>
   # Copy *.go (excluding *_test.go and example_*.go) into env/
   ```
   Upstream tests are deliberately **excluded** from the vendored tree: they
   pull in test-only build tags, exercise reflection on internal types that
   are easy to break when files are reorganized for vendoring, and would
   nearly double the on-disk footprint without testing anything we're
   actually consuming. The trade-off is that we don't catch upstream
   regressions automatically — which is exactly why every update goes
   through the line-by-line diff review in step 2.
4. **Record.** Update `env/CREDITS.md` (or `dotenv/CREDITS.md`) with the new
   version, date, reviewer, and a short note on what changed upstream.
5. **Verify.** `go build ./... && go vet ./... && go test ./...` from the
   `configkit/` root.

### Update checklist

- [ ] Upstream version is 3+ months old (unless security fix)
- [ ] No open security advisories on the target version
- [ ] All changed lines reviewed
- [ ] No new external dependencies added
- [ ] All configkit tests still pass
- [ ] `CREDITS.md` updated with version, date, reviewer

## Tests

```bash
go test ./...
```

## License

The wrapper code (`config.go`, `doc.go`, `config_test.go`) is MIT-licensed,
copyright © 2026 Amedaz s.a.l. — see [`LICENSE`](LICENSE). The vendored
libraries retain their original MIT licenses — see `env/LICENSE.md` and
`dotenv/LICENCE`.
