package detect

type prisma struct{}

func (prisma) Name() string { return "prisma" }

func (prisma) Match(path string) bool {
	return hasExt(path, ".prisma")
}

func (prisma) Detect(path string, src []byte) []Usage {
	src = blankLineComments(src, "//")
	return findCallKeys(src, path, "prisma", "env")
}
