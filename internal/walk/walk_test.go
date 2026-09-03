package walk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkSkipsNodeModulesAndGitignore(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src", "app.js"), "ok")
	mustWrite(t, filepath.Join(root, "node_modules", "dep.js"), "no")
	mustWrite(t, filepath.Join(root, ".gitignore"), "secret.js\n")
	mustWrite(t, filepath.Join(root, "secret.js"), "no")
	mustWrite(t, filepath.Join(root, ".env"), "FOO=bar\n")
	mustWrite(t, filepath.Join(root, ".env.example"), "FOO=\n")

	files, err := Walk(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Rel] = true
	}
	if !got["src/app.js"] {
		t.Fatalf("missing src/app.js in %#v", got)
	}
	if got["node_modules/dep.js"] {
		t.Fatal("node_modules should be skipped")
	}
	if got["secret.js"] {
		t.Fatal("gitignored secret.js should be skipped")
	}
	if !got[".env"] || !got[".env.example"] {
		t.Fatalf("dotenv files should be included: %#v", got)
	}
}

func TestWalkNoIgnoreStillSkipsNodeModules(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "src.js"), "ok")
	mustWrite(t, filepath.Join(root, ".gitignore"), "src.js\n")
	mustWrite(t, filepath.Join(root, "node_modules", "x.js"), "no")

	files, err := Walk(root, Options{NoIgnore: true})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Rel] = true
	}
	if !got["src.js"] {
		t.Fatal("--no-ignore should include gitignored source")
	}
	if got["node_modules/x.js"] {
		t.Fatal("node_modules still skipped")
	}
}

func TestWalkExtraPatterns(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "keep.js"), "ok")
	mustWrite(t, filepath.Join(root, "generated", "x.js"), "no")
	files, err := Walk(root, Options{ExtraPatterns: []string{"generated/"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Rel == "generated/x.js" {
			t.Fatal("generated/ should be skipped")
		}
	}
}

func TestWalkSkipsBinary(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "ok.js"), "ok")
	if err := os.WriteFile(filepath.Join(root, "blob.bin"), []byte{0, 1, 2, 0}, 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := Walk(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Rel == "blob.bin" {
			t.Fatal("binary should be skipped")
		}
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
