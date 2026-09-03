package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/limezsh/envopsy/internal/analyze"
	"github.com/limezsh/envopsy/internal/config"
	"github.com/limezsh/envopsy/internal/report"
)

// Version is set at link time by GoReleaser.
var Version = "0.1.0"

const usage = `Usage: envopsy [flags] [path]

Analyze a project for environment variable drift between source code
and .env.example / .env.sample / .env.template.

Exit codes:
  0  no findings at or above --fail-on
  1  findings at or above --fail-on
  2  tool error

Flags:
`

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// Run is the CLI entry point. args should not include the program name.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("envopsy", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}

	var (
		jsonOut    bool
		fix        bool
		failOn     string
		ignores    stringList
		configPath string
		checkLocal bool
		noIgnore   bool
		quiet      bool
		showVer    bool
	)
	fs.BoolVar(&jsonOut, "json", false, "print JSON instead of text")
	fs.BoolVar(&fix, "fix", false, "append missing keys as empty values to .env.example")
	fs.StringVar(&failOn, "fail-on", "warning", "exit 1 on `error`, `warning`, or `never`")
	fs.Var(&ignores, "ignore", "variable name to ignore (repeatable)")
	fs.StringVar(&configPath, "config", "", "path to .envopsy.toml")
	fs.BoolVar(&checkLocal, "check-local", false, "also flag keys in .env that are missing from the example file")
	fs.BoolVar(&noIgnore, "no-ignore", false, "do not respect .gitignore")
	fs.BoolVar(&quiet, "q", false, "print nothing; still set the exit code")
	fs.BoolVar(&quiet, "quiet", false, "print nothing; still set the exit code")
	fs.BoolVar(&showVer, "v", false, "print version")
	fs.BoolVar(&showVer, "version", false, "print version")

	if err := fs.Parse(reorderArgs(args)); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if showVer {
		fmt.Fprintln(stdout, "envopsy "+Version)
		return 0
	}

	path := "."
	switch fs.NArg() {
	case 0:
	case 1:
		path = fs.Arg(0)
	default:
		fmt.Fprintln(stderr, "envopsy: extra arguments")
		return 2
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(stderr, "envopsy: %v\n", err)
		return 2
	}
	if _, err := os.Stat(abs); err != nil {
		fmt.Fprintf(stderr, "envopsy: %v\n", err)
		return 2
	}

	cfg, _, err := config.Load(abs, configPath)
	if err != nil {
		fmt.Fprintf(stderr, "envopsy: %v\n", err)
		return 2
	}

	if cfg.FailOn != "" && !flagSet(fs, "fail-on") {
		failOn = cfg.FailOn
	}
	switch failOn {
	case "error", "warning", "never":
	default:
		fmt.Fprintf(stderr, "envopsy: invalid --fail-on %q (want error, warning, or never)\n", failOn)
		return 2
	}

	ignoreVars := append([]string{}, cfg.IgnoreVars...)
	ignoreVars = append(ignoreVars, ignores...)

	opts := analyze.Options{
		IgnoreVars:   ignoreVars,
		IgnorePaths:  cfg.IgnorePaths,
		ExampleFiles: cfg.ExampleFiles,
		CheckLocal:   checkLocal,
		NoIgnore:     noIgnore,
	}

	res, err := analyze.Run(abs, opts)
	if err != nil {
		fmt.Fprintf(stderr, "envopsy: %v\n", err)
		return 2
	}

	if fix {
		if _, err := analyze.ApplyFix(abs, res); err != nil {
			fmt.Fprintf(stderr, "envopsy: --fix: %v\n", err)
			return 2
		}
		res, err = analyze.Run(abs, opts)
		if err != nil {
			fmt.Fprintf(stderr, "envopsy: %v\n", err)
			return 2
		}
	}

	if !quiet {
		if jsonOut {
			if err := report.JSON(stdout, res); err != nil {
				fmt.Fprintf(stderr, "envopsy: %v\n", err)
				return 2
			}
		} else {
			report.Human(stdout, Version, res, useColor(stdout))
		}
	}

	return exitCode(res, failOn)
}

func exitCode(res analyze.Result, failOn string) int {
	switch failOn {
	case "never":
		return 0
	case "error":
		if res.Stats.Errors > 0 {
			return 1
		}
		return 0
	default:
		if res.Stats.Errors > 0 || res.Stats.Warnings > 0 {
			return 1
		}
		return 0
	}
}

// reorderArgs lets flags appear before or after the path, since Go's flag
// package otherwise stops at the first non-flag argument.
func reorderArgs(args []string) []string {
	takesValue := map[string]bool{
		"-fail-on": true, "--fail-on": true,
		"-ignore": true, "--ignore": true,
		"-config": true, "--config": true,
	}
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			name, _, hasEq := strings.Cut(a, "=")
			if hasEq {
				continue
			}
			if takesValue[name] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

func flagSet(fs *flag.FlagSet, name string) bool {
	seen := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			seen = true
		}
	})
	return seen
}

func useColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
