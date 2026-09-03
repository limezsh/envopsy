package detect

type python struct{}

func (python) Name() string { return "python" }

func (python) Match(path string) bool {
	return hasExt(path, ".py")
}

func (python) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "#")
	var out []Usage
	out = append(out, findCallKeys(src, path, "python", "os.getenv")...)
	out = append(out, findCallKeys(src, path, "python", "os.environ.get")...)
	out = append(out, findCallKeys(src, path, "python", "os.environ.setdefault")...)
	out = append(out, findBracketKeys(src, path, "python", "os.environ")...)
	return out
}
