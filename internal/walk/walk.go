package walk

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/limezsh/envopsy/internal/dotenv"
)

const (
	maxFileSize = 1 << 20 // 1 MiB
	peekSize    = 8 << 10 // 8 KiB
)

// File is a candidate file under the scan root.
type File struct {
	Path string // absolute
	Rel  string // slash-separated, relative to root
	Size int64
}

// Options control directory walking.
type Options struct {
	NoIgnore      bool
	ExtraPatterns []string
}

var skipDirs = map[string]struct{}{
	"node_modules":  {},
	".git":          {},
	"vendor":        {},
	"dist":          {},
	"build":         {},
	"target":        {},
	".next":         {},
	".nuxt":         {},
	"coverage":      {},
	"__pycache__":   {},
	".venv":         {},
	"venv":          {},
	"Pods":          {},
	"site-packages": {},
}

// Walk lists files under root. Dotenv files are included even when gitignored
// so --check-local and tracked-env can still see them. Source files honor
// gitignore. Directory names in skipDirs are always skipped; .git is always
// skipped even with NoIgnore.
func Walk(root string, opts Options) ([]File, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}

	var extra *ignore.GitIgnore
	if len(opts.ExtraPatterns) > 0 {
		extra = ignore.CompileIgnoreLines(opts.ExtraPatterns...)
	}
	giCache := map[string]*ignore.GitIgnore{}

	if !info.IsDir() {
		return walkSingle(root, info)
	}

	var files []File
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}

		name := d.Name()
		if d.IsDir() {
			if name == ".git" {
				return filepath.SkipDir
			}
			if _, skip := skipDirs[name]; skip {
				return filepath.SkipDir
			}
			if d.Type()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr == nil && rel != "." && isIgnored(root, path, true, opts.NoIgnore, extra, giCache) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			if !within(root, target) {
				return nil
			}
		}

		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if fi.Size() > maxFileSize {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if extra != nil && extra.MatchesPath(slash) {
			return nil
		}
		dotenvFile := dotenv.IsDotenvName(name)
		if !dotenvFile && isIgnored(root, path, false, opts.NoIgnore, extra, giCache) {
			return nil
		}
		if isBinaryFile(path) {
			return nil
		}

		files = append(files, File{
			Path: path,
			Rel:  slash,
			Size: fi.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func walkSingle(path string, info os.FileInfo) ([]File, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		st, err := os.Stat(target)
		if err != nil {
			return nil, err
		}
		info = st
		path = target
	}
	if info.IsDir() {
		return Walk(path, Options{NoIgnore: true})
	}
	if info.Size() > maxFileSize || isBinaryFile(path) {
		return nil, nil
	}
	return []File{{Path: path, Rel: filepath.Base(path), Size: info.Size()}}, nil
}

func isIgnored(root, abs string, isDir, noIgnore bool, extra *ignore.GitIgnore, cache map[string]*ignore.GitIgnore) bool {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	relSlash := filepath.ToSlash(rel)
	if extra != nil {
		if extra.MatchesPath(relSlash) || (isDir && extra.MatchesPath(relSlash+"/")) {
			return true
		}
	}
	if noIgnore {
		return false
	}

	dir := filepath.Dir(abs)
	for {
		gi := loadGitignore(dir, cache)
		if gi != nil {
			relToGI, err := filepath.Rel(dir, abs)
			if err == nil {
				p := filepath.ToSlash(relToGI)
				if gi.MatchesPath(p) || (isDir && gi.MatchesPath(p+"/")) {
					return true
				}
			}
		}
		if dir == root {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

func loadGitignore(dir string, cache map[string]*ignore.GitIgnore) *ignore.GitIgnore {
	if gi, ok := cache[dir]; ok {
		return gi
	}
	p := filepath.Join(dir, ".gitignore")
	gi, err := ignore.CompileIgnoreFile(p)
	if err != nil {
		cache[dir] = nil
		return nil
	}
	cache[dir] = gi
	return gi
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()
	buf := make([]byte, peekSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return true
	}
	return containsNUL(buf[:n])
}

func containsNUL(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return true
		}
	}
	return false
}
