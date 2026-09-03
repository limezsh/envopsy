package detect

type golang struct{}

func (golang) Name() string { return "go" }

func (golang) Match(path string) bool {
	return hasExt(path, ".go")
}

func (golang) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "//")
	var out []Usage
	out = append(out, findCallKeys(src, path, "go", "os.Getenv")...)
	out = append(out, findCallKeys(src, path, "go", "os.LookupEnv")...)
	return out
}
