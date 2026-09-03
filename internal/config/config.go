package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const fileName = ".envopsy.toml"

// Config is the optional project configuration.
type Config struct {
	FailOn       string
	IgnoreVars   []string
	IgnorePaths  []string
	ExampleFiles []string
}

// Load reads an explicit config path, or walks up from start looking for .envopsy.toml.
func Load(start, explicit string) (Config, string, error) {
	if explicit != "" {
		cfg, err := ParseFile(explicit)
		return cfg, explicit, err
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return Config{}, "", err
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return Config{}, "", err
	}
	if !fi.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		p := filepath.Join(dir, fileName)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			cfg, err := ParseFile(p)
			return cfg, p, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return Config{}, "", nil
}

// ParseFile parses a TOML subset used by Envopsy.
func ParseFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	return Parse(f, path)
}

// Parse parses fail_on / ignore_vars / ignore_paths / example_files.
func Parse(r io.Reader, name string) (Config, error) {
	sc := bufio.NewScanner(r)
	var cfg Config
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return cfg, fmt.Errorf("%s:%d: expected key = value", name, lineNo)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		var err error
		switch key {
		case "fail_on":
			cfg.FailOn, err = parseString(raw)
		case "ignore_vars":
			cfg.IgnoreVars, err = parseStringArray(raw)
		case "ignore_paths":
			cfg.IgnorePaths, err = parseStringArray(raw)
		case "example_files":
			cfg.ExampleFiles, err = parseStringArray(raw)
		default:
			return cfg, fmt.Errorf("%s:%d: unknown key %q", name, lineNo, key)
		}
		if err != nil {
			return cfg, fmt.Errorf("%s:%d: %w", name, lineNo, err)
		}
	}
	return cfg, sc.Err()
}

func parseString(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("empty value")
	}
	if raw[0] == '"' {
		return unquote(raw)
	}
	return raw, nil
}

func parseStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, fmt.Errorf("expected array")
	}
	inner := strings.TrimSpace(raw[1 : len(raw)-1])
	if inner == "" {
		return nil, nil
	}
	var out []string
	rest := inner
	for rest != "" {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}
		if rest[0] != '"' {
			return nil, fmt.Errorf("array values must be quoted strings")
		}
		s, err := unquoteHead(rest)
		if err != nil {
			return nil, err
		}
		out = append(out, s.val)
		rest = strings.TrimSpace(s.rest)
		if rest == "" {
			break
		}
		if rest[0] != ',' {
			return nil, fmt.Errorf("expected comma in array")
		}
		rest = rest[1:]
	}
	return out, nil
}

type quoted struct {
	val  string
	rest string
}

func unquote(raw string) (string, error) {
	q, err := unquoteHead(raw)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(q.rest) != "" {
		return "", fmt.Errorf("trailing junk after string")
	}
	return q.val, nil
}

func unquoteHead(raw string) (quoted, error) {
	if len(raw) < 2 || raw[0] != '"' {
		return quoted{}, fmt.Errorf("expected quoted string")
	}
	var b strings.Builder
	esc := false
	for i := 1; i < len(raw); i++ {
		c := raw[i]
		if esc {
			b.WriteByte(c)
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '"' {
			return quoted{val: b.String(), rest: raw[i+1:]}, nil
		}
		b.WriteByte(c)
	}
	return quoted{}, fmt.Errorf("unterminated string")
}
