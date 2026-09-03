package analyze

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/limezsh/envopsy/internal/builtin"
	"github.com/limezsh/envopsy/internal/detect"
	"github.com/limezsh/envopsy/internal/dotenv"
	"github.com/limezsh/envopsy/internal/walk"
)

type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
	SevInfo    Severity = "info"
)

type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col,omitempty"`
}

type Finding struct {
	Rule      string     `json:"rule"`
	Severity  Severity   `json:"severity"`
	Var       string     `json:"var,omitempty"`
	Message   string     `json:"message"`
	Locations []Location `json:"locations,omitempty"`
}

type Stats struct {
	Files    int `json:"files"`
	Vars     int `json:"vars"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Infos    int `json:"infos"`
}

type Result struct {
	Findings   []Finding `json:"findings"`
	Stats      Stats     `json:"stats"`
	ExampleRel string    `json:"-"`
	Root       string    `json:"-"`
	missing    []string
}

type Options struct {
	IgnoreVars   []string
	IgnorePaths  []string
	ExampleFiles []string
	CheckLocal   bool
	NoIgnore     bool
}

const maxLocations = 32

// Run walks root, detects usages, and diffs them against example dotenv files.
func Run(root string, opts Options) (Result, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	files, err := walk.Walk(root, walk.Options{
		NoIgnore:      opts.NoIgnore,
		ExtraPatterns: opts.IgnorePaths,
	})
	if err != nil {
		return Result{}, err
	}

	ignore := map[string]struct{}{}
	for _, n := range opts.IgnoreVars {
		ignore[n] = struct{}{}
	}

	var (
		usages       []detect.Usage
		exampleKeys  []dotenv.Key
		localKeys    []dotenv.Key
		exampleFiles []string
		localFiles   []string
		scanned      int
	)

	for _, f := range files {
		base := filepath.Base(f.Path)
		switch {
		case dotenv.IsTemplate(base, opts.ExampleFiles):
			keys, err := dotenv.ParseFile(f.Path, f.Rel)
			if err != nil {
				return Result{}, err
			}
			exampleKeys = append(exampleKeys, keys...)
			exampleFiles = append(exampleFiles, f.Rel)
			scanned++
		case dotenv.IsLocal(base, opts.ExampleFiles):
			keys, err := dotenv.ParseFile(f.Path, f.Rel)
			if err != nil {
				return Result{}, err
			}
			localKeys = append(localKeys, keys...)
			localFiles = append(localFiles, f.Rel)
			scanned++
		case detect.MatchAny(f.Rel):
			src, err := os.ReadFile(f.Path)
			if err != nil {
				continue
			}
			usages = append(usages, detect.Scan(f.Rel, src)...)
			scanned++
		}
	}

	used := map[string][]Location{}
	dynamicByFile := map[string]Location{}
	for _, u := range usages {
		if u.Dynamic {
			if _, ok := dynamicByFile[u.File]; !ok {
				dynamicByFile[u.File] = Location{File: u.File, Line: u.Line, Col: u.Col}
			}
			continue
		}
		if skipVar(u.Name, ignore) {
			continue
		}
		used[u.Name] = append(used[u.Name], Location{File: u.File, Line: u.Line, Col: u.Col})
	}

	example := map[string]Location{}
	for _, k := range exampleKeys {
		if skipVar(k.Name, ignore) {
			continue
		}
		if _, ok := example[k.Name]; !ok {
			example[k.Name] = Location{File: k.File, Line: k.Line}
		}
	}

	local := map[string]Location{}
	for _, k := range localKeys {
		if skipVar(k.Name, ignore) {
			continue
		}
		if _, ok := local[k.Name]; !ok {
			local[k.Name] = Location{File: k.File, Line: k.Line}
		}
	}

	var findings []Finding
	var missing []string

	if len(exampleFiles) == 0 && len(used) > 0 {
		findings = append(findings, Finding{
			Rule:     "no-example-file",
			Severity: SevWarning,
			Message:  "no .env.example / .env.sample / .env.template found",
		})
	}

	for name, locs := range used {
		if _, ok := example[name]; ok {
			continue
		}
		sort.Slice(locs, func(i, j int) bool {
			if locs[i].File != locs[j].File {
				return locs[i].File < locs[j].File
			}
			return locs[i].Line < locs[j].Line
		})
		missing = append(missing, name)
		findings = append(findings, Finding{
			Rule:      "missing-example",
			Severity:  SevError,
			Var:       name,
			Message:   fmt.Sprintf("%s is used in code but missing from .env.example", name),
			Locations: capLocs(locs),
		})
	}

	for name, loc := range example {
		if _, ok := used[name]; ok {
			continue
		}
		findings = append(findings, Finding{
			Rule:      "unused-example",
			Severity:  SevWarning,
			Var:       name,
			Message:   fmt.Sprintf("%s is documented but never referenced in code", name),
			Locations: []Location{loc},
		})
	}

	if opts.CheckLocal && len(exampleFiles) > 0 {
		for name, loc := range local {
			if _, ok := example[name]; ok {
				continue
			}
			findings = append(findings, Finding{
				Rule:      "undocumented-local",
				Severity:  SevWarning,
				Var:       name,
				Message:   fmt.Sprintf("%s is in a local .env file but not in .env.example", name),
				Locations: []Location{loc},
			})
		}
	}

	for file, loc := range dynamicByFile {
		findings = append(findings, Finding{
			Rule:      "dynamic-access",
			Severity:  SevInfo,
			Message:   fmt.Sprintf("dynamic environment access in %s cannot be resolved statically", file),
			Locations: []Location{loc},
		})
	}

	for _, rel := range localFiles {
		tracked, ok := gitTracked(root, rel)
		if !ok {
			break
		}
		if tracked {
			findings = append(findings, Finding{
				Rule:      "tracked-env",
				Severity:  SevWarning,
				Message:   fmt.Sprintf("%s is tracked by git", rel),
				Locations: []Location{{File: rel, Line: 1}},
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return sevRank(findings[i].Severity) < sevRank(findings[j].Severity)
		}
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		if findings[i].Var != findings[j].Var {
			return findings[i].Var < findings[j].Var
		}
		return findings[i].Message < findings[j].Message
	})
	sort.Strings(missing)

	res := Result{
		Findings:   findings,
		ExampleRel: pickExampleRel(exampleFiles),
		Root:       root,
		missing:    missing,
		Stats: Stats{
			Files: scanned,
			Vars:  len(used),
		},
	}
	for _, f := range findings {
		switch f.Severity {
		case SevError:
			res.Stats.Errors++
		case SevWarning:
			res.Stats.Warnings++
		case SevInfo:
			res.Stats.Infos++
		}
	}
	return res, nil
}

func skipVar(name string, extra map[string]struct{}) bool {
	if builtin.Ignored(name) {
		return true
	}
	_, ok := extra[name]
	return ok
}

func capLocs(locs []Location) []Location {
	if len(locs) <= maxLocations {
		return locs
	}
	return locs[:maxLocations]
}

func sevRank(s Severity) int {
	switch s {
	case SevError:
		return 0
	case SevWarning:
		return 1
	default:
		return 2
	}
}

func pickExampleRel(files []string) string {
	if len(files) == 0 {
		return ".env.example"
	}
	for _, f := range files {
		if filepath.Base(f) == ".env.example" {
			return f
		}
	}
	return files[0]
}

// Missing returns sorted names that are used but undocumented.
func (r Result) Missing() []string {
	return append([]string(nil), r.missing...)
}

func gitTracked(root, rel string) (tracked bool, gitOK bool) {
	cmd := exec.Command("git", "-C", root, "ls-files", "--error-unmatch", filepath.FromSlash(rel))
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	if err == nil {
		return true, true
	}
	if ee, ok := err.(*exec.Error); ok && ee.Err == exec.ErrNotFound {
		return false, false
	}
	return false, true
}

// ApplyFix appends missing keys as empty assignments to the example file.
// It never copies values from local .env files.
func ApplyFix(root string, res Result) (string, error) {
	missing := res.Missing()
	if len(missing) == 0 {
		return "", nil
	}
	rel := res.ExampleRel
	if rel == "" {
		rel = ".env.example"
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if !withinRoot(rootAbs, abs) {
		return "", fmt.Errorf("refusing to write outside scan root: %s", abs)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}

	existing := map[string]struct{}{}
	var buf []byte
	if data, err := os.ReadFile(abs); err == nil {
		buf = data
		keys, err := dotenv.Parse(strings.NewReader(string(data)))
		if err != nil {
			return "", err
		}
		for _, k := range keys {
			existing[k.Name] = struct{}{}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	var b strings.Builder
	if len(buf) > 0 && !strings.HasSuffix(string(buf), "\n") {
		b.WriteByte('\n')
	}
	wrote := false
	for _, name := range missing {
		if _, ok := existing[name]; ok {
			continue
		}
		if !validFixKey(name) {
			continue
		}
		b.WriteString(name)
		b.WriteString("=\n")
		wrote = true
	}
	if !wrote {
		return rel, nil
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(b.String()); err != nil {
		return "", err
	}
	return rel, nil
}

func validFixKey(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func withinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
