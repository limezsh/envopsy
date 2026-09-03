package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/limezsh/envopsy/internal/analyze"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", name)
	dst := t.TempDir()
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestCLIJSONJSApp(t *testing.T) {
	root := fixture(t, "js-app")
	var out, errb bytes.Buffer
	code := Run([]string{"--json", "--fail-on", "never", root}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb.String())
	}
	if strings.Contains(out.String(), "SUPER_SECRET_VALUE_XYZ") || strings.Contains(out.String(), "another-secret-value") {
		t.Fatal("secret leaked into JSON")
	}
	if strings.Contains(out.String(), "SHOULD_NOT_SEE") || strings.Contains(out.String(), "GITIGNORED_VAR") {
		t.Fatal("ignored path leaked into JSON")
	}
	var res analyze.Result
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	rules := map[string][]string{}
	for _, f := range res.Findings {
		rules[f.Rule] = append(rules[f.Rule], f.Var)
	}
	if !contains(rules["missing-example"], "VITE_API_URL") {
		t.Fatalf("findings=%v", rules)
	}
	if !contains(rules["unused-example"], "UNUSED_WEBHOOK") {
		t.Fatalf("findings=%v", rules)
	}
}

func TestCLICleanExitZero(t *testing.T) {
	root := fixture(t, "clean")
	var out, errb bytes.Buffer
	code := Run([]string{root}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s stdout=%s", code, errb.String(), out.String())
	}
	if !strings.Contains(out.String(), "No issues found") {
		t.Fatalf("stdout=%s", out.String())
	}
}

func TestCLIFailOnWarning(t *testing.T) {
	root := fixture(t, "js-app")
	var out, errb bytes.Buffer
	code := Run([]string{"--quiet", root}, &out, &errb)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Fatalf("quiet printed %q", out.String())
	}
}

func TestCLIFailOnErrorVsNever(t *testing.T) {
	root := fixture(t, "dynamic")
	var out, errb bytes.Buffer
	code := Run([]string{"--json", "--fail-on", "warning", root}, &out, &errb)
	if code != 0 {
		t.Fatalf("info-only should pass --fail-on warning, exit %d %s", code, errb.String())
	}
}

func TestCLIFix(t *testing.T) {
	root := fixture(t, "js-app")
	var out, errb bytes.Buffer
	code := Run([]string{"--fix", "--json", "--fail-on", "never", root}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb.String())
	}
	data, err := os.ReadFile(filepath.Join(root, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("SUPER_SECRET")) {
		t.Fatal("fix copied secrets")
	}
	if !bytes.Contains(data, []byte("VITE_API_URL=")) {
		t.Fatalf("fix did not append: %s", data)
	}
}

func TestCLICheckLocal(t *testing.T) {
	root := fixture(t, "js-app")
	var out, errb bytes.Buffer
	code := Run([]string{"--json", "--check-local", "--fail-on", "never", root}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "undocumented-local") {
		t.Fatalf("stdout=%s", out.String())
	}
	if strings.Contains(out.String(), "SUPER_SECRET") {
		t.Fatal("secret leaked")
	}
}

func TestCLIFlagsAfterPath(t *testing.T) {
	root := fixture(t, "js-app")
	var out, errb bytes.Buffer
	code := Run([]string{root, "--json", "--fail-on", "never"}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb.String())
	}
	if !strings.Contains(out.String(), "missing-example") {
		t.Fatalf("stdout=%s", out.String())
	}
}

func TestCLIVersion(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--version"}, &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "envopsy") {
		t.Fatalf("exit %d stdout=%s", code, out.String())
	}
}

func TestCLIBadPath(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"/no/such/envopsy-path"}, &out, &errb)
	if code != 2 {
		t.Fatalf("exit %d", code)
	}
}

func TestCLIConfigIgnore(t *testing.T) {
	root := fixture(t, "js-app")
	cfg := filepath.Join(root, ".envopsy.toml")
	body := "ignore_vars = [\"VITE_API_URL\", \"DENO_REGION\", \"UNUSED_WEBHOOK\"]\n"
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Run([]string{"--json", "--fail-on", "error", "--config", cfg, root}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s stdout=%s", code, errb.String(), out.String())
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
