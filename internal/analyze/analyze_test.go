package analyze

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

func findingVars(res Result, rule string) []string {
	var out []string
	for _, f := range res.Findings {
		if f.Rule == rule {
			out = append(out, f.Var)
		}
	}
	return out
}

func hasRule(res Result, rule string) bool {
	for _, f := range res.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestJSApp(t *testing.T) {
	root := fixture(t, "js-app")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(findingVars(res, "missing-example"), ",")
	for _, name := range []string{"VITE_API_URL", "DENO_REGION"} {
		if !contains(findingVars(res, "missing-example"), name) {
			t.Errorf("missing-example should include %s, got %s", name, got)
		}
	}
	if !contains(findingVars(res, "unused-example"), "UNUSED_WEBHOOK") {
		t.Error("expected UNUSED_WEBHOOK unused")
	}
	if contains(findingVars(res, "missing-example"), "NODE_ENV") {
		t.Error("NODE_ENV should be builtin-ignored")
	}
	if contains(findingVars(res, "missing-example"), "SHOULD_NOT_SEE") {
		t.Error("node_modules should be skipped")
	}
	if contains(findingVars(res, "missing-example"), "GITIGNORED_VAR") {
		t.Error("gitignored file should be skipped")
	}
	if contains(findingVars(res, "missing-example"), "DATABASE_URL") {
		t.Error("DATABASE_URL is documented")
	}
}

func TestPythonApp(t *testing.T) {
	root := fixture(t, "python-app")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(findingVars(res, "missing-example"), "REDIS_URL") {
		t.Fatal("REDIS_URL missing")
	}
	if !contains(findingVars(res, "unused-example"), "DEAD_PY_VAR") {
		t.Fatal("DEAD_PY_VAR unused")
	}
}

func TestGoApp(t *testing.T) {
	root := fixture(t, "go-app")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(findingVars(res, "missing-example"), "API_TOKEN") {
		t.Fatal("API_TOKEN missing")
	}
	if !contains(findingVars(res, "unused-example"), "DEAD_GO") {
		t.Fatal("DEAD_GO unused")
	}
}

func TestMixed(t *testing.T) {
	root := fixture(t, "mixed")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"MIXED_JS", "MIXED_PY", "MIXED_GO", "MIXED_RB", "MIXED_PHP", "APP_KEY", "MIXED_RS"} {
		if !contains(findingVars(res, "missing-example"), name) {
			t.Errorf("expected missing %s", name)
		}
	}
	if contains(findingVars(res, "missing-example"), "DATABASE_URL") {
		t.Error("prisma DATABASE_URL is documented")
	}
	if !contains(findingVars(res, "unused-example"), "UNUSED_MIXED") {
		t.Error("UNUSED_MIXED")
	}
}

func TestClean(t *testing.T) {
	root := fixture(t, "clean")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 {
		t.Fatalf("clean fixture had findings: %+v", res.Findings)
	}
}

func TestDynamic(t *testing.T) {
	root := fixture(t, "dynamic")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(res, "dynamic-access") {
		t.Fatal("expected dynamic-access")
	}
	if res.Stats.Errors != 0 {
		t.Fatalf("dynamic should not invent missing vars: %+v", res.Findings)
	}
}

func TestNoExampleFile(t *testing.T) {
	root := fixture(t, "no-example")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(res, "no-example-file") {
		t.Fatal("expected no-example-file")
	}
	if !contains(findingVars(res, "missing-example"), "UNDOCUMENTED") {
		t.Fatal("expected UNDOCUMENTED missing")
	}
}

func TestCheckLocal(t *testing.T) {
	root := fixture(t, "js-app")
	res, err := Run(root, Options{CheckLocal: true})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(findingVars(res, "undocumented-local"), "LOCAL_ONLY") {
		t.Fatalf("expected LOCAL_ONLY, got %v", findingVars(res, "undocumented-local"))
	}
}

func TestIgnoreVars(t *testing.T) {
	root := fixture(t, "js-app")
	res, err := Run(root, Options{IgnoreVars: []string{"VITE_API_URL", "DENO_REGION", "UNUSED_WEBHOOK"}})
	if err != nil {
		t.Fatal(err)
	}
	if contains(findingVars(res, "missing-example"), "VITE_API_URL") {
		t.Fatal("ignored var still reported")
	}
	if contains(findingVars(res, "unused-example"), "UNUSED_WEBHOOK") {
		t.Fatal("ignored unused still reported")
	}
}

func TestFix(t *testing.T) {
	root := fixture(t, "js-app")
	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	rel, err := ApplyFix(root, res)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, "SUPER_SECRET_VALUE_XYZ") {
		t.Fatal("fix copied a secret")
	}
	if !strings.Contains(body, "VITE_API_URL=\n") {
		t.Fatalf("expected empty VITE_API_URL, got:\n%s", body)
	}
	res2, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if contains(findingVars(res2, "missing-example"), "VITE_API_URL") {
		t.Fatal("fix did not clear missing-example")
	}
}

func TestTrackedEnv(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := fixture(t, "js-app")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@t")
	run("config", "user.name", "t")
	run("add", ".env")
	run("commit", "-m", "add env", "--no-gpg-sign")

	res, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasRule(res, "tracked-env") {
		t.Fatalf("expected tracked-env, findings=%+v", res.Findings)
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
