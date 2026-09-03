package detect

type rust struct{}

func (rust) Name() string { return "rust" }

func (rust) Match(path string) bool {
	return hasExt(path, ".rs")
}

func (rust) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "//")
	var out []Usage
	out = append(out, findCallKeys(src, path, "rust", "env::var")...)
	out = append(out, findCallKeys(src, path, "rust", "env::var_os")...)
	out = append(out, findCallKeys(src, path, "rust", "option_env!")...)
	return out
}
