package detect

type php struct{}

func (php) Name() string { return "php" }

func (php) Match(path string) bool {
	return hasExt(path, ".php")
}

func (php) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "//", "#")
	var out []Usage
	out = append(out, findCallKeys(src, path, "php", "getenv")...)
	out = append(out, findCallKeys(src, path, "php", "env")...)
	out = append(out, findBracketKeys(src, path, "php", "$_ENV")...)
	return out
}
