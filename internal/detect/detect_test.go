package detect

import (
	"sort"
	"testing"
)

func names(us []Usage) []string {
	var out []string
	for _, u := range us {
		if !u.Dynamic {
			out = append(out, u.Name)
		}
	}
	sort.Strings(out)
	return out
}

func hasDynamic(us []Usage) bool {
	for _, u := range us {
		if u.Dynamic {
			return true
		}
	}
	return false
}

func eqNames(t *testing.T, got []Usage, want ...string) {
	t.Helper()
	g := names(got)
	if len(g) != len(want) {
		t.Fatalf("names=%v want=%v (full=%+v)", g, want, got)
	}
	for i := range want {
		if g[i] != want[i] {
			t.Fatalf("names=%v want=%v", g, want)
		}
	}
}

func TestJavaScript(t *testing.T) {
	src := []byte(`
const a = process.env.DATABASE_URL;
const b = process.env['STRIPE_KEY'];
const c = process.env["REDIS_URL"];
const d = import.meta.env.VITE_API_URL;
const e = Deno.env.get("DENO_KV");
const f = process.env[key];
const g = process.env.NODE_ENV;
// process.env.COMMENTED_OUT
lookalike process.env
const x = myprocess.env.NOT_US;
`)
	got := Scan("src/app.ts", src)
	eqNames(t, got, "DATABASE_URL", "DENO_KV", "NODE_ENV", "REDIS_URL", "STRIPE_KEY", "VITE_API_URL")
	if !hasDynamic(got) {
		t.Fatal("expected dynamic process.env[key]")
	}
	for _, u := range got {
		if u.Name == "COMMENTED_OUT" || u.Name == "NOT_US" {
			t.Fatalf("false positive: %+v", u)
		}
	}
}

func TestPython(t *testing.T) {
	src := []byte(`
import os
os.getenv("DATABASE_URL")
os.environ["REDIS_URL"]
os.environ.get('API_KEY')
os.environ.setdefault("DEFAULT_VAR", "x")
os.getenv(f"PREFIX_{name}")
os.getenv(key)
# os.getenv("COMMENTED")
environ["NO"]
`)
	got := Scan("app.py", src)
	eqNames(t, got, "API_KEY", "DATABASE_URL", "DEFAULT_VAR", "REDIS_URL")
	if !hasDynamic(got) {
		t.Fatal("expected dynamic getenv")
	}
	for _, u := range got {
		if u.Name == "COMMENTED" || u.Name == "NO" {
			t.Fatalf("false positive: %+v", u)
		}
	}
}

func TestGo(t *testing.T) {
	src := []byte(`
package main
import "os"
func main() {
	os.Getenv("DATABASE_URL")
	os.LookupEnv("API_TOKEN")
	os.Getenv(dynamicKey)
	os.Getenv(` + "`RAW_KEY`" + `)
	// os.Getenv("COMMENTED")
}
`)
	got := Scan("main.go", src)
	eqNames(t, got, "API_TOKEN", "DATABASE_URL", "RAW_KEY")
	if !hasDynamic(got) {
		t.Fatal("expected dynamic Getenv")
	}
}

func TestRuby(t *testing.T) {
	src := []byte(`
ENV["DATABASE_URL"]
ENV.fetch('REDIS_URL')
ENV[key]
# ENV["COMMENTED"]
`)
	got := Scan("app.rb", src)
	eqNames(t, got, "DATABASE_URL", "REDIS_URL")
	if !hasDynamic(got) {
		t.Fatal("expected dynamic ENV[key]")
	}
}

func TestPHP(t *testing.T) {
	src := []byte(`
<?php
getenv("DATABASE_URL");
$_ENV['REDIS_URL'];
env('APP_KEY');
$_SERVER['HTTP_HOST'];
getenv($name);
`)
	got := Scan("index.php", src)
	eqNames(t, got, "APP_KEY", "DATABASE_URL", "REDIS_URL")
	for _, u := range got {
		if u.Name == "HTTP_HOST" {
			t.Fatal("$_SERVER should not be scanned")
		}
	}
	if !hasDynamic(got) {
		t.Fatal("expected dynamic getenv")
	}
}

func TestRust(t *testing.T) {
	src := []byte(`
std::env::var("DATABASE_URL");
env::var("REDIS_URL");
env::var_os("HOME_OVERRIDE");
option_env!("COMPILE_TIME");
env::var(key);
`)
	got := Scan("main.rs", src)
	eqNames(t, got, "COMPILE_TIME", "DATABASE_URL", "HOME_OVERRIDE", "REDIS_URL")
	if !hasDynamic(got) {
		t.Fatal("expected dynamic env::var")
	}
}

func TestPrisma(t *testing.T) {
	src := []byte(`
datasource db {
  url = env("DATABASE_URL")
}
`)
	got := Scan("schema.prisma", src)
	eqNames(t, got, "DATABASE_URL")
}

func TestNoMatchWrongExtension(t *testing.T) {
	src := []byte(`process.env.FOO`)
	if got := Scan("readme.md", src); len(got) != 0 {
		t.Fatalf("md should not scan: %+v", got)
	}
}

func TestLineNumbers(t *testing.T) {
	src := []byte("const x = 1;\nconst y = process.env.FOO;\n")
	got := Scan("a.js", src)
	if len(got) != 1 || got[0].Line != 2 || got[0].Name != "FOO" {
		t.Fatalf("got %+v", got)
	}
}
