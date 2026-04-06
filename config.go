package configkit

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/gobeaver/configkit/dotenv"
	"github.com/gobeaver/configkit/env"
)

// DefaultPrefix is the default environment variable prefix.
const DefaultPrefix = "BEAVER_"

// Options configures the config loader.
//
// Value precedence (highest to lowest):
//
//  1. Process environment variables (set by the OS, container, shell, etc.)
//  2. Earlier entries in EnvFiles (first file wins on conflict)
//  3. Later entries in EnvFiles
//  4. envDefault tag values on the target struct
//
// Process env always wins over file contents — this matches 12-factor app
// conventions and lets deployment platforms (Kubernetes, systemd, CI) override
// committed defaults without editing files.
type Options struct {
	// Prefix for environment variable names (default: "BEAVER_").
	Prefix string

	// EnvFiles to load before parsing (default: ".env"). Earlier entries
	// take precedence over later ones. See WithEnvFiles for the rationale.
	EnvFiles []string

	// SkipDotEnv skips loading .env files entirely. Useful in tests and
	// in environments where all configuration comes from the process env.
	SkipDotEnv bool

	// Required makes all fields required unless they have envDefault tags.
	// Equivalent to setting `,required` on every field.
	Required bool
}

// Load parses environment variables into a struct.
// Automatically loads .env files first (can be disabled via options).
//
// A missing .env file is not an error. A malformed one is.
//
// Example:
//
//	type Config struct {
//	    Host string `env:"HOST" envDefault:"localhost"`
//	    Port int    `env:"PORT" envDefault:"8080"`
//	}
//
//	var cfg Config
//	configkit.Load(&cfg)                               // Uses BEAVER_ prefix
//	configkit.Load(&cfg, configkit.WithPrefix(""))     // No prefix
//	configkit.Load(&cfg, configkit.WithPrefix("APP_")) // Custom prefix
func Load(cfg any, opts ...Option) error {
	options := Options{
		Prefix:   DefaultPrefix,
		EnvFiles: []string{".env"},
	}

	for _, opt := range opts {
		opt(&options)
	}

	if !options.SkipDotEnv {
		for _, file := range options.EnvFiles {
			if err := dotenv.Load(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("configkit: loading %s: %w", file, err)
			}
		}
	}

	envOpts := env.Options{
		Prefix: options.Prefix,
	}

	if options.Required {
		envOpts.RequiredIfNoDef = true
	}

	if err := env.ParseWithOptions(cfg, envOpts); err != nil {
		return fmt.Errorf("configkit: %w", err)
	}
	return nil
}

// MustLoad is like Load but panics on error.
// The panic value is an error wrapping the underlying cause, so callers
// can recover and use errors.Is / errors.As.
func MustLoad(cfg any, opts ...Option) {
	if err := Load(cfg, opts...); err != nil {
		panic(err)
	}
}

// Option configures Load behavior.
type Option func(*Options)

// WithPrefix sets a custom environment variable prefix.
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithEnvFiles sets custom .env files to load.
//
// Files are loaded with first-wins semantics: process env always takes
// precedence, then earlier files override later ones. To make a local
// override file win over a base file, list the override file first:
//
//	configkit.Load(&cfg, configkit.WithEnvFiles(".env.local", ".env"))
func WithEnvFiles(files ...string) Option {
	return func(o *Options) {
		o.EnvFiles = files
	}
}

// WithoutDotEnv disables automatic .env file loading.
func WithoutDotEnv() Option {
	return func(o *Options) {
		o.SkipDotEnv = true
	}
}

// WithRequired makes all fields required unless they have defaults.
func WithRequired() Option {
	return func(o *Options) {
		o.Required = true
	}
}
