package dotenv

import (
	"strings"
	"testing"
)

func TestParseKeysOnly(t *testing.T) {
	src := `
# comment
FOO=bar
export BAR=baz
EMPTY=
BAZ="quoted value"
QUOTED='also quoted'
# SKIP=not-a-key
NOT_A_KEY
export
INVALID-KEY=no
_UNDERSCORE=1
port=lowercase
MULTILINE="line1
line2
line3"
AFTER_MULTI=ok
FOO=duplicate-ignored
HASH=value # inline comment
`
	keys, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(keys))
	for _, k := range keys {
		got = append(got, k.Name)
	}
	want := []string{"FOO", "BAR", "EMPTY", "BAZ", "QUOTED", "_UNDERSCORE", "port", "MULTILINE", "AFTER_MULTI", "HASH"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
	// Duplicate FOO keeps first line.
	if keys[0].Line != 3 {
		t.Errorf("FOO line = %d, want 3", keys[0].Line)
	}
}

func TestParseNeverKeepsValues(t *testing.T) {
	secret := "SUPER_SECRET_VALUE_XYZ"
	src := "DATABASE_URL=postgres://" + secret + "\n"
	keys, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Name != "DATABASE_URL" {
		t.Fatalf("unexpected keys: %+v", keys)
	}
	raw, _ := keys[0].Name, keys[0].File
	if strings.Contains(raw, secret) {
		t.Fatal("secret leaked into key name")
	}
}

func TestClassify(t *testing.T) {
	if !IsTemplate(".env.example", nil) {
		t.Fatal("expected .env.example template")
	}
	if !IsTemplate(".env.local.example", nil) {
		t.Fatal("expected .env.local.example template")
	}
	if !IsTemplate(".env.sample", nil) {
		t.Fatal("expected .env.sample template")
	}
	if IsLocal(".env.example", nil) {
		t.Fatal(".env.example must not be local")
	}
	if !IsLocal(".env", nil) {
		t.Fatal(".env should be local")
	}
	if !IsLocal(".env.development", nil) {
		t.Fatal(".env.development should be local")
	}
	if IsTemplate(".env.development", nil) {
		t.Fatal(".env.development is not a template")
	}
	if !IsTemplate(".env.example", []string{".env.example"}) {
		t.Fatal("custom example list")
	}
	if IsTemplate(".env.sample", []string{".env.example"}) {
		t.Fatal(".env.sample should not be template when example_files is set")
	}
}
