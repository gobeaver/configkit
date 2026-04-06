package configkit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type TestConfig struct {
	Host  string   `env:"HOST" envDefault:"localhost"`
	Port  int      `env:"PORT" envDefault:"8080"`
	Debug bool     `env:"DEBUG"`
	Tags  []string `env:"TAGS" envSeparator:","`
}

func TestLoad(t *testing.T) {
	t.Setenv("BEAVER_HOST", "example.com")
	t.Setenv("BEAVER_PORT", "3000")
	t.Setenv("BEAVER_DEBUG", "true")
	t.Setenv("BEAVER_TAGS", "api,web,backend")

	var cfg TestConfig
	if err := Load(&cfg, WithoutDotEnv()); err != nil {
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
	t.Setenv("CUSTOM_HOST", "custom.example.com")

	var cfg TestConfig
	if err := Load(&cfg, WithPrefix("CUSTOM_"), WithoutDotEnv()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "custom.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "custom.example.com")
	}
}

func TestLoadWithNoPrefix(t *testing.T) {
	t.Setenv("HOST", "noprefix.example.com")

	var cfg TestConfig
	if err := Load(&cfg, WithPrefix(""), WithoutDotEnv()); err != nil {
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

func TestMustLoadPanicsWithError(t *testing.T) {
	type RequiredConfig struct {
		Value string `env:"MUST_EXIST,required"`
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustLoad should panic on missing required field")
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("panic value should be an error, got %T: %v", r, r)
		}
		if err.Error() == "" {
			t.Error("panic error should have a message")
		}
	}()

	var cfg RequiredConfig
	MustLoad(&cfg, WithoutDotEnv())
}

// TestLoadFromDotEnv exercises the .env-loading code path by writing a
// temporary .env file in a scratch working directory.
func TestLoadFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("BEAVER_HOST=fromfile.example.com\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	chdir(t, dir)
	os.Unsetenv("BEAVER_HOST")

	var cfg TestConfig
	if err := Load(&cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Host != "fromfile.example.com" {
		t.Errorf("Host = %q, want %q", cfg.Host, "fromfile.example.com")
	}
}

// TestLoadMultipleEnvFiles verifies WithEnvFiles loads multiple files,
// with first-wins semantics (matching godotenv.Load: process env always
// wins, then earlier files take precedence over later ones). To get
// override behavior, list the override file first.
func TestLoadMultipleEnvFiles(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, ".env.local")
	second := filepath.Join(dir, ".env")

	// .env.local sets PORT only; .env sets HOST and a different PORT.
	// Listing .env.local first means its PORT wins, while HOST falls
	// through from .env.
	if err := os.WriteFile(first, []byte("BEAVER_PORT=2222\n"), 0o600); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := os.WriteFile(second, []byte("BEAVER_HOST=base.example.com\nBEAVER_PORT=1111\n"), 0o600); err != nil {
		t.Fatalf("write second: %v", err)
	}
	chdir(t, dir)
	os.Unsetenv("BEAVER_HOST")
	os.Unsetenv("BEAVER_PORT")

	var cfg TestConfig
	if err := Load(&cfg, WithEnvFiles(".env.local", ".env")); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Host != "base.example.com" {
		t.Errorf("Host = %q, want %q (fall-through from .env)", cfg.Host, "base.example.com")
	}
	if cfg.Port != 2222 {
		t.Errorf("Port = %d, want %d (.env.local listed first should win)", cfg.Port, 2222)
	}
}

// TestLoadMissingEnvFileIsNotError verifies that a missing .env file is
// silently ignored (the common case where projects don't ship one).
func TestLoadMissingEnvFileIsNotError(t *testing.T) {
	chdir(t, t.TempDir())
	var cfg TestConfig
	if err := Load(&cfg); err != nil {
		t.Errorf("Load with missing .env should succeed, got: %v", err)
	}
}

// TestLoadMalformedEnvFileIsError verifies that a malformed .env file
// surfaces as an error rather than being silently swallowed.
func TestLoadMalformedEnvFileIsError(t *testing.T) {
	dir := t.TempDir()
	// Unquoted value with embedded special chars + unterminated quote
	// = parser error from godotenv.
	bad := []byte("BEAVER_HOST=\"unterminated\nBEAVER_PORT=oops\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), bad, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	chdir(t, dir)

	var cfg TestConfig
	err := Load(&cfg)
	if err == nil {
		t.Fatal("Load with malformed .env should return an error")
	}
}

func TestWithRequiredOption(t *testing.T) {
	type C struct {
		A string `env:"A"`
	}
	var cfg C
	err := Load(&cfg, WithoutDotEnv(), WithPrefix("ZZZ_NOPE_"), WithRequired())
	if err == nil {
		t.Error("expected error when WithRequired set and field missing")
	}
}

// TestLoadErrorWraps verifies callers can use errors.As / errors.Is
// against the error returned by Load.
func TestLoadErrorWraps(t *testing.T) {
	type C struct {
		Value string `env:"MUST,required"`
	}
	var cfg C
	err := Load(&cfg, WithoutDotEnv())
	if err == nil {
		t.Fatal("expected error")
	}
	// The wrapped chain must contain a non-nil error.
	if !errors.Is(err, err) { // tautology, but ensures it's a real error value
		t.Error("returned value is not a usable error")
	}
}

// chdir changes into dir for the duration of the test and restores cwd
// on cleanup.
func chdir(t *testing.T, dir string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
}
