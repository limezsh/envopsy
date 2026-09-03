package detect

type javascript struct{}

func (javascript) Name() string { return "javascript" }

func (javascript) Match(path string) bool {
	return hasExt(path, ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx")
}

func (javascript) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "//")
	var out []Usage
	out = append(out, findDotKeys(src, path, "javascript", "process.env.")...)
	out = append(out, findBracketKeys(src, path, "javascript", "process.env")...)
	out = append(out, findDotKeys(src, path, "javascript", "import.meta.env.")...)
	out = append(out, findBracketKeys(src, path, "javascript", "import.meta.env")...)
	out = append(out, findCallKeys(src, path, "javascript", "Deno.env.get")...)
	return out
}
