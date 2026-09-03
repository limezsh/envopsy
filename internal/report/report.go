package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/limezsh/envopsy/internal/analyze"
)

const locShow = 3

const (
	cReset  = "\033[0m"
	cRed    = "\033[31m"
	cYellow = "\033[33m"
	cCyan   = "\033[36m"
	cBold   = "\033[1m"
)

// JSON writes machine-readable output.
func JSON(w io.Writer, res analyze.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

// Human writes the default CLI report.
func Human(w io.Writer, version string, res analyze.Result, color bool) {
	fmt.Fprintf(w, "envopsy %s  %d files  %d vars\n", version, res.Stats.Files, res.Stats.Vars)
	if len(res.Findings) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No issues found.")
		return
	}
	fmt.Fprintln(w)
	for _, f := range res.Findings {
		sev := string(f.Severity)
		if color {
			sev = paint(f.Severity) + string(f.Severity) + cReset
		}
		if f.Var != "" {
			fmt.Fprintf(w, "%s  %s  %s\n", sev, f.Rule, f.Var)
		} else {
			fmt.Fprintf(w, "%s  %s  %s\n", sev, f.Rule, f.Message)
		}
		if len(f.Locations) > 0 {
			fmt.Fprintf(w, "       %s\n", formatLocs(f.Locations))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, summary(res.Stats))
}

func formatLocs(locs []analyze.Location) string {
	n := len(locs)
	show := locs
	extra := 0
	if n > locShow {
		show = locs[:locShow]
		extra = n - locShow
	}
	parts := make([]string, 0, len(show)+1)
	for _, l := range show {
		if l.Line > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", l.File, l.Line))
		} else {
			parts = append(parts, l.File)
		}
	}
	if extra > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", extra))
	}
	return strings.Join(parts, "  ")
}

func summary(s analyze.Stats) string {
	var parts []string
	parts = append(parts, plural(s.Errors, "error", "errors"))
	parts = append(parts, plural(s.Warnings, "warning", "warnings"))
	if s.Infos > 0 {
		parts = append(parts, plural(s.Infos, "info", "infos"))
	}
	return strings.Join(parts, ", ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func paint(s analyze.Severity) string {
	switch s {
	case analyze.SevError:
		return cRed + cBold
	case analyze.SevWarning:
		return cYellow
	default:
		return cCyan
	}
}
