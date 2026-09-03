package detect

import "bytes"

func lineCol(src []byte, offset int) (line, col int) {
	line = 1
	col = 1
	if offset > len(src) {
		offset = len(src)
	}
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func isWordChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

func atWordStart(src []byte, i int) bool {
	if i <= 0 {
		return true
	}
	return !isWordChar(src[i-1])
}

func skipSpace(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
		i++
	}
	return i
}

func identEnd(src []byte, i int) int {
	if i >= len(src) {
		return i
	}
	if !isWordChar(src[i]) && src[i] != '$' {
		return i
	}
	i++
	for i < len(src) && (isWordChar(src[i]) || src[i] == '$') {
		i++
	}
	return i
}

func identEndStrict(src []byte, i int) int {
	if i >= len(src) || !isWordChar(src[i]) {
		return i
	}
	i++
	for i < len(src) && isWordChar(src[i]) {
		i++
	}
	return i
}

// blankLineComments replaces full-line comments with spaces, preserving newlines
// so match line numbers stay stable.
func blankLineComments(src []byte, prefixes ...string) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	start := 0
	for start <= len(src) {
		end := start
		for end < len(src) && src[end] != '\n' {
			end++
		}
		line := src[start:end]
		trim := 0
		for trim < len(line) && (line[trim] == ' ' || line[trim] == '\t') {
			trim++
		}
		body := line[trim:]
		comment := false
		for _, p := range prefixes {
			if len(body) >= len(p) && string(body[:len(p)]) == p {
				comment = true
				break
			}
		}
		if comment {
			for i := start + trim; i < end; i++ {
				out[i] = ' '
			}
		}
		if end == len(src) {
			break
		}
		start = end + 1
	}
	return out
}

func usage(path, lang, name string, src []byte, nameOff int) Usage {
	line, col := lineCol(src, nameOff)
	return Usage{Name: name, File: path, Line: line, Col: col, Lang: lang}
}

func dynUsage(path, lang string, src []byte, off int) Usage {
	line, col := lineCol(src, off)
	return Usage{File: path, Line: line, Col: col, Lang: lang, Dynamic: true}
}

func indexAll(src []byte, needle string) []int {
	if needle == "" {
		return nil
	}
	var out []int
	n := []byte(needle)
	from := 0
	for {
		i := indexAt(src, n, from)
		if i < 0 {
			break
		}
		out = append(out, i)
		from = i + 1
	}
	return out
}

func indexAt(src, needle []byte, from int) int {
	if from >= len(src) || len(needle) == 0 {
		return -1
	}
	i := bytes.Index(src[from:], needle)
	if i < 0 {
		return -1
	}
	return from + i
}

// findDotKeys finds prefix+IDENT such as process.env.FOO.
func findDotKeys(src []byte, path, lang, prefix string) []Usage {
	var out []Usage
	for _, i := range indexAll(src, prefix) {
		if !atWordStart(src, i) {
			continue
		}
		j := i + len(prefix)
		end := identEnd(src, j)
		if end == j {
			continue
		}
		name := string(src[j:end])
		out = append(out, usage(path, lang, name, src, j))
	}
	return out
}

// findCallKeys finds fn("NAME") / fn('NAME') / fn(`NAME`) and flags non-literal args as dynamic.
func findCallKeys(src []byte, path, lang, fn string) []Usage {
	var out []Usage
	needle := fn + "("
	for _, i := range indexAll(src, needle) {
		if !atWordStart(src, i) {
			continue
		}
		j := skipSpace(src, i+len(needle))
		if j >= len(src) {
			continue
		}
		// Python string prefixes: f/F → dynamic, r/b/u → still a string.
		if j+1 < len(src) && (src[j] == 'f' || src[j] == 'F') && (src[j+1] == '"' || src[j+1] == '\'') {
			out = append(out, dynUsage(path, lang, src, i))
			continue
		}
		if j+1 < len(src) && (src[j] == 'r' || src[j] == 'R' || src[j] == 'b' || src[j] == 'B' || src[j] == 'u' || src[j] == 'U') &&
			(src[j+1] == '"' || src[j+1] == '\'') {
			j++
		}
		if src[j] == '"' || src[j] == '\'' || src[j] == '`' {
			q := src[j]
			k := j + 1
			for k < len(src) && src[k] != q && src[k] != '\n' {
				k++
			}
			if k < len(src) && src[k] == q {
				name := string(src[j+1 : k])
				if name != "" {
					out = append(out, usage(path, lang, name, src, j+1))
				}
			}
			continue
		}
		if src[j] != ')' {
			out = append(out, dynUsage(path, lang, src, i))
		}
	}
	return out
}

// findBracketKeys finds recv['NAME'] / recv["NAME"] and flags recv[expr] as dynamic.
func findBracketKeys(src []byte, path, lang, recv string) []Usage {
	var out []Usage
	needle := recv + "["
	for _, i := range indexAll(src, needle) {
		if !atWordStart(src, i) {
			continue
		}
		j := skipSpace(src, i+len(needle))
		if j >= len(src) {
			continue
		}
		if src[j] == '"' || src[j] == '\'' {
			q := src[j]
			k := j + 1
			for k < len(src) && src[k] != q && src[k] != '\n' {
				k++
			}
			if k < len(src) && src[k] == q {
				name := string(src[j+1 : k])
				if name != "" {
					out = append(out, usage(path, lang, name, src, j+1))
				}
			}
			continue
		}
		out = append(out, dynUsage(path, lang, src, i))
	}
	return out
}
