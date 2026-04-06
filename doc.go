// Package configkit provides environment variable configuration loading with
// automatic .env file support and configurable prefixes.
//
// This package wraps vendored copies of github.com/caarlos0/env and
// github.com/joho/godotenv with sensible defaults for Beaver applications.
//
// # 12-Factor Compliance
//
// configkit follows the III. Config principle of the 12-factor app
// methodology (https://12factor.net/config): configuration is read strictly
// from the environment, never compiled in or stored in version-controlled
// config files. The optional .env loader is a developer-ergonomics layer
// only — process environment variables always take precedence over file
// values, so deployment platforms (Kubernetes, systemd, CI runners, secret
// managers) remain the source of truth in production.
//
// Value precedence, highest to lowest:
//
//  1. Process environment variables
//  2. Earlier .env files listed in WithEnvFiles
//  3. Later .env files
//  4. envDefault struct tags
//
// # Basic Usage
//
//	type Config struct {
//	    Host string `env:"HOST" envDefault:"localhost"`
//	    Port int    `env:"PORT" envDefault:"8080"`
//	}
//
//	var cfg Config
//	configkit.Load(&cfg) // Uses BEAVER_ prefix, loads .env
//
// # Environment Variables
//
// By default, all environment variables are prefixed with "BEAVER_":
//
//	BEAVER_HOST=example.com
//	BEAVER_PORT=3000
//
// # Custom Prefixes
//
// Use WithPrefix for multi-instance configurations:
//
//	// Primary database: PRIMARY_DB_HOST, PRIMARY_DB_PORT
//	configkit.Load(&dbCfg, configkit.WithPrefix("PRIMARY_DB_"))
//
//	// Replica database: REPLICA_DB_HOST, REPLICA_DB_PORT
//	configkit.Load(&dbCfg, configkit.WithPrefix("REPLICA_DB_"))
//
//	// No prefix: DB_HOST, DB_PORT
//	configkit.Load(&dbCfg, configkit.WithPrefix(""))
//
// # Supported Types
//
// All types supported by caarlos0/env are available:
//
//	type Config struct {
//	    // Basic types
//	    String   string        `env:"STRING"`
//	    Int      int           `env:"INT"`
//	    Bool     bool          `env:"BOOL"`
//	    Duration time.Duration `env:"DURATION"`
//
//	    // Slices
//	    Hosts []string `env:"HOSTS" envSeparator:","`
//
//	    // Required fields
//	    APIKey string `env:"API_KEY,required"`
//
//	    // Defaults
//	    Port int `env:"PORT" envDefault:"8080"`
//
//	    // Nested structs
//	    Database DatabaseConfig `envPrefix:"DB_"`
//	}
//
// # .env File Support
//
// The package automatically loads .env files before parsing.
// Use WithEnvFiles to specify custom files:
//
//	configkit.Load(&cfg, configkit.WithEnvFiles(".env", ".env.local"))
//
// Use WithoutDotEnv to disable automatic loading:
//
//	configkit.Load(&cfg, configkit.WithoutDotEnv())
//
// # Multi-Instance Pattern
//
// The prefix pattern enables running multiple instances with different configs:
//
//	// Environment:
//	// DEV_SLACK_WEBHOOK_URL=https://hooks.slack.com/dev
//	// PROD_SLACK_WEBHOOK_URL=https://hooks.slack.com/prod
//
//	devSlack := slack.WithPrefix("DEV_").New()
//	prodSlack := slack.WithPrefix("PROD_").New()
//
// This pattern avoids YAML complexity while remaining 12-factor compliant.
package configkit
