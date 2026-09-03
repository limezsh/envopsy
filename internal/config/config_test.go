package config

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	src := `
# comment
fail_on = "warning"
ignore_vars = ["FOO", "BAR"]
ignore_paths = ["generated/"]
example_files = [".env.example"]
`
	cfg, err := Parse(strings.NewReader(src), "test.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FailOn != "warning" {
		t.Fatalf("fail_on=%q", cfg.FailOn)
	}
	if len(cfg.IgnoreVars) != 2 || cfg.IgnoreVars[0] != "FOO" || cfg.IgnoreVars[1] != "BAR" {
		t.Fatalf("ignore_vars=%v", cfg.IgnoreVars)
	}
	if len(cfg.IgnorePaths) != 1 || cfg.IgnorePaths[0] != "generated/" {
		t.Fatalf("ignore_paths=%v", cfg.IgnorePaths)
	}
	if len(cfg.ExampleFiles) != 1 || cfg.ExampleFiles[0] != ".env.example" {
		t.Fatalf("example_files=%v", cfg.ExampleFiles)
	}
}

func TestParseEmptyArray(t *testing.T) {
	cfg, err := Parse(strings.NewReader(`ignore_vars = []`), "t")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IgnoreVars != nil && len(cfg.IgnoreVars) != 0 {
		t.Fatalf("got %v", cfg.IgnoreVars)
	}
}

func TestUnknownKey(t *testing.T) {
	_, err := Parse(strings.NewReader(`nope = "x"`), "t")
	if err == nil {
		t.Fatal("expected error")
	}
}
