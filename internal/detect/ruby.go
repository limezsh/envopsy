package detect

type ruby struct{}

func (ruby) Name() string { return "ruby" }

func (ruby) Match(path string) bool {
	return hasExt(path, ".rb")
}

func (ruby) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "#")
	var out []Usage
	out = append(out, findBracketKeys(src, path, "ruby", "ENV")...)
	out = append(out, findCallKeys(src, path, "ruby", "ENV.fetch")...)
	return out
}
