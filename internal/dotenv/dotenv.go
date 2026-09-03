package dotenv

import (
	"bufio"
	"io"
	"os"
	"strings"
	"unicode"
)

// Key is an environment variable name found in a dotenv file.
// Values are never stored.
type Key struct {
	Name string
	File string
	Line int
}

// ParseFile reads a dotenv file and returns keys only.
func ParseFile(path, rel string) ([]Key, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	keys, err := Parse(f)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		keys[i].File = rel
	}
	return keys, nil
}

// Parse extracts KEY names from a dotenv-formatted stream.
// Values, including multiline quoted values, are discarded.
func Parse(r io.Reader) ([]Key, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var keys []Key
	seen := make(map[string]struct{})
	lineNo := 0
	var inQuote byte // non-zero while inside a multiline quoted value

	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if inQuote != 0 {
			inQuote = updateQuoteState(line, inQuote)
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		rest := trimmed
		if strings.HasPrefix(rest, "export") && (len(rest) == 6 || unicode.IsSpace(rune(rest[6]))) {
			rest = strings.TrimSpace(rest[6:])
		}
		if rest == "" || strings.HasPrefix(rest, "#") {
			continue
		}

		name, ok, remainder := splitKey(rest)
		if !ok {
			continue
		}
		if _, dup := seen[name]; !dup {
			seen[name] = struct{}{}
			keys = append(keys, Key{Name: name, Line: lineNo})
		}
		inQuote = quoteOpenAfterValue(remainder)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func splitKey(line string) (name string, ok bool, afterEq string) {
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", false, ""
	}
	raw := strings.TrimSpace(line[:eq])
	if !validKey(raw) {
		return "", false, ""
	}
	return raw, true, line[eq+1:]
}

func validKey(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func quoteOpenAfterValue(val string) byte {
	s := strings.TrimSpace(val)
	if i := strings.IndexByte(s, '#'); i >= 0 && !oddUnescapedQuotes(s[:i]) {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" {
		return 0
	}
	if s[0] != '"' && s[0] != '\'' {
		return 0
	}
	q := s[0]
	if closedQuote(s[1:], q) {
		return 0
	}
	return q
}

func oddUnescapedQuotes(s string) bool {
	n := 0
	esc := false
	for i := 0; i < len(s); i++ {
		if esc {
			esc = false
			continue
		}
		if s[i] == '\\' {
			esc = true
			continue
		}
		if s[i] == '"' {
			n++
		}
	}
	return n%2 == 1
}

func closedQuote(s string, q byte) bool {
	esc := false
	for i := 0; i < len(s); i++ {
		if esc {
			esc = false
			continue
		}
		if q == '"' && s[i] == '\\' {
			esc = true
			continue
		}
		if s[i] == q {
			return true
		}
	}
	return false
}

func updateQuoteState(line string, q byte) byte {
	if closedQuote(line, q) {
		return 0
	}
	return q
}

// IsTemplate reports whether basename is an env template/example file.
// If extra is non-empty, only those exact basenames count as templates.
func IsTemplate(base string, extra []string) bool {
	if len(extra) > 0 {
		for _, n := range extra {
			if base == n {
				return true
			}
		}
		return false
	}
	switch base {
	case ".env.example", ".env.sample", ".env.template":
		return true
	}
	if strings.HasPrefix(base, ".env") &&
		(strings.HasSuffix(base, ".example") ||
			strings.HasSuffix(base, ".sample") ||
			strings.HasSuffix(base, ".template")) {
		return true
	}
	return strings.HasSuffix(base, ".env.example")
}

// IsLocal reports whether basename is a local dotenv file (not a template).
func IsLocal(base string, exampleFiles []string) bool {
	if IsTemplate(base, exampleFiles) {
		return false
	}
	if base == ".env" {
		return true
	}
	return strings.HasPrefix(base, ".env.")
}

// IsDotenvName reports whether basename looks like any dotenv file
// using default template names.
func IsDotenvName(base string) bool {
	return IsTemplate(base, nil) || IsLocal(base, nil)
}
