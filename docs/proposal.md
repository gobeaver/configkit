# Proposal: Beaver Config Refactor

**Author:** Engineering Team  
**Status:** Draft  
**Created:** December 2025

---

## Summary

Replace the custom beaver-kit/config implementation with a thin wrapper around vendored copies of `caarlos0/env` and `joho/godotenv`, following a security-first approach with audited, lagged updates.

---

## Security Philosophy

**Vendor all dependencies. Trust no one blindly.**

| Principle | Implementation |
|-----------|----------------|
| **Lag behind** | Wait 3-6 months after upstream release before updating |
| **Community audit** | Let others discover issues first |
| **Code review** | Diff every update, review changes line-by-line |
| **No transitive trust** | Vendored code = no surprise updates |
| **Audit trail** | CREDITS.md tracks versions and update dates |

---

## Problem Statement

The current beaver-kit/config package:

1. **Reinvents the wheel** — Reimplements features that caarlos0/env already provides
2. **Feature gap** — Missing slices, maps, nested structs, required fields, env expansion
3. **Maintenance burden** — We maintain code that a well-tested library already handles
4. **No real value-add** — Only unique feature is auto `.env` loading (one line of code)

### Current beaver-kit/config vs caarlos0/env

| Feature | beaver-kit/config | caarlos0/env |
|---------|-------------------|--------------|
| String, int, bool, duration | ✓ | ✓ |
| Prefix support | ✓ | ✓ |
| Default values | ✓ | ✓ |
| Slices | ✗ | ✓ |
| Maps | ✗ | ✓ |
| Required fields | ✗ | ✓ |
| Nested structs | ✗ | ✓ |
| Custom unmarshalers | ✗ | ✓ |
| Env var expansion | ✗ | ✓ |
| Notnil validation | ✗ | ✓ |
| File loading | ✗ | ✓ |

---

## Design Philosophy

### Why Prefix-Based Multi-Instance?

The prefix pattern enables multi-tenant and multi-instance configurations without YAML complexity:

```bash
# Two Slack instances
DEV_SLACK_WEBHOOK_URL=https://hooks.slack.com/dev
MARKETING_SLACK_WEBHOOK_URL=https://hooks.slack.com/marketing

# Two databases
PRIMARY_DB_HOST=primary.db.example.com
REPLICA_DB_HOST=replica.db.example.com

# Multi-tenant SaaS
TENANT1_DB_HOST=tenant1.db.example.com
TENANT2_DB_HOST=tenant2.db.example.com
```

```go
// Usage
devSlack := slack.WithPrefix("DEV_").New()
marketingSlack := slack.WithPrefix("MARKETING_").New()

primaryDB := database.WithPrefix("PRIMARY_").New()
replicaDB := database.WithPrefix("REPLICA_").New()
```

**Benefits:**
- No YAML files to manage
- 12-factor app compliant
- Works with any secret manager (Vault, AWS SSM, K8s secrets)
- Simple mental model
- Easy debugging (`env | grep PRIMARY_`)

---

## Proposed Solution

### New Package Structure

```
config/
├── config.go       # Thin wrapper (~60 lines)
├── config_test.go  # Tests
├── doc.go          # Package documentation
├── dotenv/         # Vendored: joho/godotenv
│   ├── CREDITS.md  # Version, source, update history
│   ├── godotenv.go
│   ├── parser.go
│   └── ...
└── env/            # Vendored: caarlos0/env
    ├── CREDITS.md  # Version, source, update history
    ├── env.go
    ├── parser.go
    └── ...
```

### CREDITS.md Format

```markdown
# Credits

## Source
- Package: github.com/caarlos0/env
- Version: v11.2.0
- License: MIT

## Update History
| Date | Version | Reviewer | Notes |
|------|---------|----------|-------|
| 2025-12-10 | v11.2.0 | @engineer | Initial vendor |
| 2026-06-15 | v11.3.0 | @engineer | Security patch for X |

## Upstream Changes Reviewed
- v11.2.0 → v11.3.0: +45/-12 lines, no new deps, reviewed OK
```

### Implementation

```go
// config/config.go
package config

import (
    "github.com/gobeaver/beaver-kit/config/dotenv"
    "github.com/gobeaver/beaver-kit/config/env"
)

// DefaultPrefix is the default environment variable prefix
const DefaultPrefix = "BEAVER_"

// Options configures the config loader
type Options struct {
    // Prefix for environment variable names (default: "BEAVER_")
    Prefix string
    
    // EnvFiles to load before parsing (default: ".env")
    EnvFiles []string
    
    // SkipDotEnv skips loading .env files
    SkipDotEnv bool
    
    // Required makes all fields required unless marked optional
    Required bool
}

// Load parses environment variables into a struct.
// Automatically loads .env file first (can be disabled via options).
//
// Example:
//
//     type Config struct {
//         Host string `env:"HOST" envDefault:"localhost"`
//         Port int    `env:"PORT" envDefault:"8080"`
//     }
//
//     var cfg Config
//     config.Load(&cfg)                           // Uses BEAVER_ prefix
//     config.Load(&cfg, config.WithPrefix(""))    // No prefix
//     config.Load(&cfg, config.WithPrefix("APP_")) // Custom prefix
//
func Load(cfg interface{}, opts ...Option) error {
    options := Options{
        Prefix:   DefaultPrefix,
        EnvFiles: []string{".env"},
    }
    
    for _, opt := range opts {
        opt(&options)
    }
    
    // Load .env files (vendored joho/godotenv)
    if !options.SkipDotEnv {
        for _, file := range options.EnvFiles {
            _ = dotenv.Load(file) // Ignore errors - file may not exist
        }
    }
    
    // Parse environment variables (vendored caarlos0/env)
    envOpts := env.Options{
        Prefix: options.Prefix,
    }
    
    if options.Required {
        envOpts.RequiredIfNoDef = true
    }
    
    return env.ParseWithOptions(cfg, envOpts)
}

// MustLoad is like Load but panics on error
func MustLoad(cfg interface{}, opts ...Option) {
    if err := Load(cfg, opts...); err != nil {
        panic("config: " + err.Error())
    }
}

// Option configures Load behavior
type Option func(*Options)

// WithPrefix sets a custom environment variable prefix
func WithPrefix(prefix string) Option {
    return func(o *Options) {
        o.Prefix = prefix
    }
}

// WithEnvFiles sets custom .env files to load
func WithEnvFiles(files ...string) Option {
    return func(o *Options) {
        o.EnvFiles = files
    }
}

// WithoutDotEnv disables automatic .env file loading
func WithoutDotEnv() Option {
    return func(o *Options) {
        o.SkipDotEnv = true
    }
}

// WithRequired makes all fields required unless they have defaults
func WithRequired() Option {
    return func(o *Options) {
        o.Required = true
    }
}
```

### Tests

```go
// config/config_test.go
package config

import (
    "os"
    "testing"
)

type TestConfig struct {
    Host     string   `env:"HOST" envDefault:"localhost"`
    Port     int      `env:"PORT" envDefault:"8080"`
    Debug    bool     `env:"DEBUG"`
    Tags     []string `env:"TAGS" envSeparator:","`
    Required string   `env:"REQUIRED"`
}

func TestLoad(t *testing.T) {
    // Clean up
    defer func() {
        os.Unsetenv("BEAVER_HOST")
        os.Unsetenv("BEAVER_PORT")
        os.Unsetenv("BEAVER_DEBUG")
        os.Unsetenv("BEAVER_TAGS")
    }()
    
    os.Setenv("BEAVER_HOST", "example.com")
    os.Setenv("BEAVER_PORT", "3000")
    os.Setenv("BEAVER_DEBUG", "true")
    os.Setenv("BEAVER_TAGS", "api,web,backend")
    
    var cfg TestConfig
    if err := Load(&cfg); err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    
    if cfg.Host != "example.com" {
        t.Errorf("Host = %q, want %q", cfg.Host, "example.com")
    }
    if cfg.Port != 3000 {
        t.Errorf("Port = %d, want %d", cfg.Port, 3000)
    }
    if !cfg.Debug {
        t.Error("Debug = false, want true")
    }
    if len(cfg.Tags) != 3 || cfg.Tags[0] != "api" {
        t.Errorf("Tags = %v, want [api web backend]", cfg.Tags)
    }
}

func TestLoadWithPrefix(t *testing.T) {
    defer os.Unsetenv("CUSTOM_HOST")
    
    os.Setenv("CUSTOM_HOST", "custom.example.com")
    
    var cfg TestConfig
    if err := Load(&cfg, WithPrefix("CUSTOM_")); err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    
    if cfg.Host != "custom.example.com" {
        t.Errorf("Host = %q, want %q", cfg.Host, "custom.example.com")
    }
}

func TestLoadWithNoPrefix(t *testing.T) {
    defer os.Unsetenv("HOST")
    
    os.Setenv("HOST", "noprefix.example.com")
    
    var cfg TestConfig
    if err := Load(&cfg, WithPrefix("")); err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    
    if cfg.Host != "noprefix.example.com" {
        t.Errorf("Host = %q, want %q", cfg.Host, "noprefix.example.com")
    }
}

func TestLoadDefaults(t *testing.T) {
    var cfg TestConfig
    if err := Load(&cfg, WithoutDotEnv()); err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    
    if cfg.Host != "localhost" {
        t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
    }
    if cfg.Port != 8080 {
        t.Errorf("Port = %d, want %d", cfg.Port, 8080)
    }
}

func TestMustLoadPanics(t *testing.T) {
    type RequiredConfig struct {
        Value string `env:"MUST_EXIST,required"`
    }
    
    defer func() {
        if r := recover(); r == nil {
            t.Error("MustLoad should panic on missing required field")
        }
    }()
    
    var cfg RequiredConfig
    MustLoad(&cfg, WithoutDotEnv())
}
```

### Documentation

```go
// config/doc.go

// Package config provides environment variable configuration loading with
// automatic .env file support and configurable prefixes.
//
// This package wraps github.com/caarlos0/env with sensible defaults for
// Beaver Kit applications.
//
// # Basic Usage
//
//     type Config struct {
//         Host string `env:"HOST" envDefault:"localhost"`
//         Port int    `env:"PORT" envDefault:"8080"`
//     }
//
//     var cfg Config
//     config.Load(&cfg) // Uses BEAVER_ prefix, loads .env
//
// # Environment Variables
//
// By default, all environment variables are prefixed with "BEAVER_":
//
//     BEAVER_HOST=example.com
//     BEAVER_PORT=3000
//
// # Custom Prefixes
//
// Use WithPrefix for multi-instance configurations:
//
//     // Primary database: PRIMARY_DB_HOST, PRIMARY_DB_PORT
//     config.Load(&dbCfg, config.WithPrefix("PRIMARY_DB_"))
//
//     // Replica database: REPLICA_DB_HOST, REPLICA_DB_PORT
//     config.Load(&dbCfg, config.WithPrefix("REPLICA_DB_"))
//
//     // No prefix: DB_HOST, DB_PORT
//     config.Load(&dbCfg, config.WithPrefix(""))
//
// # Supported Types
//
// All types supported by caarlos0/env are available:
//
//     type Config struct {
//         // Basic types
//         String   string        `env:"STRING"`
//         Int      int           `env:"INT"`
//         Bool     bool          `env:"BOOL"`
//         Duration time.Duration `env:"DURATION"`
//
//         // Slices
//         Hosts []string `env:"HOSTS" envSeparator:","`
//
//         // Required fields
//         APIKey string `env:"API_KEY,required"`
//
//         // Defaults
//         Port int `env:"PORT" envDefault:"8080"`
//
//         // Nested structs
//         Database DatabaseConfig `envPrefix:"DB_"`
//     }
//
// # .env File Support
//
// The package automatically loads .env files before parsing.
// Use WithEnvFiles to specify custom files:
//
//     config.Load(&cfg, config.WithEnvFiles(".env", ".env.local"))
//
// Use WithoutDotEnv to disable automatic loading:
//
//     config.Load(&cfg, config.WithoutDotEnv())
//
// # Multi-Instance Pattern
//
// The prefix pattern enables running multiple instances with different configs:
//
//     // Environment:
//     // DEV_SLACK_WEBHOOK_URL=https://hooks.slack.com/dev
//     // PROD_SLACK_WEBHOOK_URL=https://hooks.slack.com/prod
//
//     devSlack := slack.WithPrefix("DEV_").New()
//     prodSlack := slack.WithPrefix("PROD_").New()
//
// This pattern avoids YAML complexity while remaining 12-factor compliant.
package config
```

---

## Migration Guide

### For Package Maintainers (beaver-kit internal)

**Before:**
```go
import "github.com/gobeaver/beaver-kit/config"

type Config struct {
    Host string `env:"DB_HOST,default:localhost"`
}

config.Load(&cfg, config.LoadOptions{Prefix: "BEAVER_"})
```

**After:**
```go
import "github.com/gobeaver/beaver-kit/config"

type Config struct {
    Host string `env:"DB_HOST" envDefault:"localhost"`
}

config.Load(&cfg) // BEAVER_ prefix is now default
```

### Tag Changes

| Before | After |
|--------|-------|
| `env:"NAME,default:value"` | `env:"NAME" envDefault:"value"` |
| `config.LoadOptions{Prefix: "X_"}` | `config.WithPrefix("X_")` |

### For End Users

**No changes required.** Environment variables and usage patterns remain identical:

```bash
BEAVER_DB_HOST=localhost
BEAVER_DB_PORT=5432
```

```go
database.Init()  // Still works exactly the same
```

---

## Files to Delete

```
config/
├── config.go           # DELETE (replace with new wrapper)
├── config_test.go      # DELETE (rewrite with new tests)
└── doc.go              # DELETE (rewrite with new docs)
```

## Files to Keep

```
config/
└── dotenv/             # KEEP (joho/godotenv vendored)
    ├── CREDITS.md      # UPDATE with version tracking
    ├── godotenv.go
    ├── godotenv_test.go
    ├── parser.go
    └── README.md
```

## Files to Add

```
config/
└── env/                # NEW (caarlos0/env vendored)
    ├── CREDITS.md      # Version, source, update history
    ├── env.go
    ├── env_test.go
    ├── parser.go
    ├── options.go
    └── README.md
```

## No External Dependencies

```go
// go.mod - NO new external dependencies
// Everything is vendored internally
```

---

## Vendor Update Process

### When to Update

1. **Security fix** — Update immediately after review
2. **Bug fix you need** — Wait 1 month, then update
3. **New features** — Wait 3-6 months, then update
4. **No reason** — Don't update

### How to Update

```bash
# 1. Check upstream changes
git clone https://github.com/caarlos0/env /tmp/env-upstream
cd /tmp/env-upstream
git log --oneline v11.2.0..v11.3.0

# 2. Review diff
git diff v11.2.0..v11.3.0 -- *.go

# 3. Check for new dependencies
cat go.mod

# 4. Copy files (exclude tests, examples, CI)
cp *.go /path/to/beaver-kit/config/env/

# 5. Update CREDITS.md
# 6. Run beaver-kit tests
# 7. Commit with detailed message
```

### Update Checklist

- [ ] Upstream version is 3+ months old
- [ ] No open security issues on that version
- [ ] Reviewed all changed lines
- [ ] No new external dependencies added
- [ ] All beaver-kit tests pass
- [ ] CREDITS.md updated with version and date
- [ ] Commit message includes upstream diff summary

---

## Package Usage in Beaver Kit

### database package

```go
package database

import "github.com/gobeaver/beaver-kit/config"

type Config struct {
    Driver   string `env:"DB_DRIVER" envDefault:"sqlite"`
    Host     string `env:"DB_HOST" envDefault:"localhost"`
    Port     string `env:"DB_PORT"`
    Database string `env:"DB_DATABASE" envDefault:"beaver.db"`
    Username string `env:"DB_USERNAME"`
    Password string `env:"DB_PASSWORD"`
    URL      string `env:"DB_URL"` // Overrides individual fields
}

type Builder struct {
    prefix string
}

func WithPrefix(prefix string) *Builder {
    return &Builder{prefix: prefix}
}

func (b *Builder) Init() error {
    var cfg Config
    if err := config.Load(&cfg, config.WithPrefix(b.prefix)); err != nil {
        return err
    }
    return initWithConfig(cfg)
}

// Default initialization with BEAVER_ prefix
func Init() error {
    return WithPrefix("BEAVER_").Init()
}
```

### slack package

```go
package slack

import "github.com/gobeaver/beaver-kit/config"

type Config struct {
    WebhookURL string `env:"SLACK_WEBHOOK_URL,required"`
    Channel    string `env:"SLACK_CHANNEL"`
    Username   string `env:"SLACK_USERNAME" envDefault:"Beaver"`
    IconEmoji  string `env:"SLACK_ICON_EMOJI"`
}

type Builder struct {
    prefix string
}

func WithPrefix(prefix string) *Builder {
    return &Builder{prefix: prefix}
}

func (b *Builder) Init() error {
    var cfg Config
    if err := config.Load(&cfg, config.WithPrefix(b.prefix)); err != nil {
        return err
    }
    return initWithConfig(cfg)
}

func (b *Builder) New() (*Client, error) {
    var cfg Config
    if err := config.Load(&cfg, config.WithPrefix(b.prefix)); err != nil {
        return nil, err
    }
    return NewClient(cfg), nil
}

// Usage:
// slack.WithPrefix("DEV_").Init()      → DEV_SLACK_WEBHOOK_URL
// slack.WithPrefix("MARKETING_").New() → MARKETING_SLACK_WEBHOOK_URL
```

---

## Advanced Features (Now Available)

### Nested Structs with Prefix

```go
type Config struct {
    App      AppConfig      `envPrefix:"APP_"`
    Database DatabaseConfig `envPrefix:"DB_"`
    Cache    CacheConfig    `envPrefix:"CACHE_"`
}

type AppConfig struct {
    Host string `env:"HOST" envDefault:"localhost"`
    Port int    `env:"PORT" envDefault:"8080"`
}

// Results in: BEAVER_APP_HOST, BEAVER_APP_PORT, BEAVER_DB_HOST, etc.
```

### Required Fields

```go
type Config struct {
    APIKey string `env:"API_KEY,required"` // Fails if not set
    Debug  bool   `env:"DEBUG"`            // Optional
}
```

### Slices and Maps

```go
type Config struct {
    Hosts    []string          `env:"HOSTS" envSeparator:","`
    Ports    []int             `env:"PORTS" envSeparator:","`
    Metadata map[string]string `env:"METADATA"`
}

// BEAVER_HOSTS=host1,host2,host3
// BEAVER_PORTS=8080,8081,8082
// BEAVER_METADATA=key1:val1,key2:val2
```

### Custom Unmarshalers

```go
type LogLevel int

func (l *LogLevel) UnmarshalText(text []byte) error {
    switch string(text) {
    case "debug":
        *l = 0
    case "info":
        *l = 1
    case "warn":
        *l = 2
    case "error":
        *l = 3
    default:
        return fmt.Errorf("unknown log level: %s", text)
    }
    return nil
}

type Config struct {
    Level LogLevel `env:"LOG_LEVEL" envDefault:"info"`
}
```

---

## Validation (Optional Integration)

For projects wanting validation, use go-playground/validator:

```go
import (
    "github.com/gobeaver/beaver-kit/config"
    "github.com/go-playground/validator/v10"
)

type Config struct {
    Host  string `env:"HOST" validate:"required,hostname"`
    Port  int    `env:"PORT" validate:"required,min=1,max=65535"`
    Email string `env:"EMAIL" validate:"required,email"`
}

func LoadAndValidate(cfg interface{}, opts ...config.Option) error {
    if err := config.Load(cfg, opts...); err != nil {
        return err
    }
    return validator.New().Struct(cfg)
}
```

This is intentionally **not** built into beaver-kit/config to keep it minimal.

---

## Timeline

| Task | Duration |
|------|----------|
| Delete old implementation | 1 hour |
| Vendor caarlos0/env | 2 hours |
| Update dotenv CREDITS.md | 30 min |
| Write new wrapper | 2 hours |
| Write tests | 2 hours |
| Update all beaver-kit packages | 4 hours |
| Update documentation | 2 hours |
| Security review | 2 hours |

**Total: ~2 days**

---

## Benefits

| Benefit | Description |
|---------|-------------|
| **Less code to maintain** | ~60 lines wrapper vs ~150 lines custom |
| **More features** | Slices, maps, nested, required, custom types |
| **Battle-tested** | caarlos0/env has 5k+ stars, wide adoption |
| **Backward compatible** | Same env vars, same usage patterns |
| **Better errors** | caarlos0/env has descriptive error messages |
| **Security-first** | Vendored deps, no external runtime fetches |
| **Audit trail** | CREDITS.md tracks all versions and reviews |
| **Zero transitive deps** | Both libraries are pure stdlib |

---

## Security Considerations

### Why Vendor Instead of Import?

| Concern | Direct Import | Vendored |
|---------|---------------|----------|
| Malicious update | Auto-pulled on build | Manual review required |
| Supply chain attack | Exposed | Isolated |
| Audit capability | Must check upstream | Code in your repo |
| Reproducible builds | Depends on proxy | Always reproducible |
| Emergency response | Wait for upstream | Patch locally if needed |

### What We're Trusting

| Package | Lines | Deps | Risk |
|---------|-------|------|------|
| joho/godotenv | ~300 | 0 | Low |
| caarlos0/env | ~500 | 0 | Low |

Both are pure Go, no CGO, no network calls, no file system access beyond .env reading.

---

## Conclusion

This refactor:

1. **Reduces maintenance** by leveraging well-tested libraries
2. **Adds features** without adding complexity
3. **Preserves the multi-instance pattern** that justifies the wrapper
4. **Maintains backward compatibility** for all existing users
5. **Enforces security-first** via vendored, audited dependencies

The wrapper's unique value:
- Auto `.env` loading
- Default `BEAVER_` prefix
- Consistent API across beaver-kit packages
- **Vendored dependencies with audit trail**

Everything else comes from the vendored caarlos0/env and joho/godotenv.

**Recommendation:** Approve for implementation.
