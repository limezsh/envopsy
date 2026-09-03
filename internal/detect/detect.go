package detect

import "path/filepath"

// Usage is a single environment variable access in source code.
type Usage struct {
	Name    string
	File    string
	Line    int
	Col     int
	Lang    string
	Dynamic bool
}

// Detector finds env var accesses in a family of source files.
type Detector interface {
	Name() string
	Match(path string) bool
	Detect(path string, src []byte) []Usage
}

// All returns the built-in language detectors.
func All() []Detector {
	return []Detector{
		javascript{},
		python{},
		golang{},
		ruby{},
		php{},
		rust{},
		prisma{},
	}
}

// Scan runs every matching detector against src.
func Scan(path string, src []byte) []Usage {
	var out []Usage
	for _, d := range All() {
		if d.Match(path) {
			out = append(out, d.Detect(path, src)...)
		}
	}
	return out
}

// MatchAny reports whether any detector claims this path.
func MatchAny(path string) bool {
	for _, d := range All() {
		if d.Match(path) {
			return true
		}
	}
	return false
}

func extLower(path string) string {
	return stringsToLowerExt(filepath.Ext(path))
}

func stringsToLowerExt(ext string) string {
	b := []byte(ext)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func hasExt(path string, exts ...string) bool {
	e := extLower(path)
	for _, want := range exts {
		if e == want {
			return true
		}
	}
	return false
}
