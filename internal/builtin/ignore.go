package builtin

// Ignored is the set of platform / framework variables that almost never
// belong in .env.example. Names are matched exactly.
var platform = map[string]struct{}{
	"PATH":           {},
	"HOME":           {},
	"USER":           {},
	"USERNAME":       {},
	"SHELL":          {},
	"PWD":            {},
	"OLDPWD":         {},
	"TMPDIR":         {},
	"TEMP":           {},
	"TMP":            {},
	"TERM":           {},
	"CI":             {},
	"NODE_ENV":       {},
	"NO_COLOR":       {},
	"MODE":           {},
	"BASE_URL":       {},
	"DEV":            {},
	"PROD":           {},
	"SSR":            {},
	"GITHUB_ACTIONS": {},
	"LANG":           {},
	"LC_ALL":         {},
	"SHLVL":          {},
	"LOGNAME":        {},
	"GOPATH":         {},
	"GOROOT":         {},
	"PYTHONPATH":     {},
	"VIRTUAL_ENV":    {},
	"COLORTERM":      {},
	"HOSTNAME":       {},
}

// Ignored reports whether name is a well-known platform variable.
func Ignored(name string) bool {
	_, ok := platform[name]
	return ok
}
