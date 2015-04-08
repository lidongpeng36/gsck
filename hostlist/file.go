package hostlist

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	RegisterHostlist(func(str string) Hostlist {
		return &fromFile{str}
	})
}

type fromFile struct {
	filepath string
}

// pragma mark - Hostlist Interface

func (hf *fromFile) Name() string {
	return "file"
}

func (hf *fromFile) Priority() int {
	return 0
}

// Get reads file content, and pass it to hostlistFromString
func (hf *fromFile) Get() (list []string, err error) {
	fi, e := os.Lstat(hf.filepath)
	if os.IsNotExist(e) {
		err = fmt.Errorf("No such file: %s", hf.filepath)
		return
	}
	if e != nil {
		err = e
		return
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		hf.filepath, err = filepath.EvalSymlinks(hf.filepath)
		if err != nil {
			return
		}
	}
	var buf []byte
	buf, err = ioutil.ReadFile(hf.filepath)
	if err != nil {
		return
	}
	hs := &fromString{string(buf)}
	list, err = hs.Get()
	return
}

// If the string
//   1. is a single-line text
//   2. has no space (/\s+/)
//   3. starts with "./", "/", "~/" or ".."
// then it indicates that user want it to be a filename.
func (hf *fromFile) ShouldBreak() bool {
	spaceRegexp := regexp.MustCompile("\\s+")
	seps := spaceRegexp.Split(hf.filepath, -1)
	if len(seps) == 1 &&
		(strings.HasPrefix(hf.filepath, "./") ||
			strings.HasPrefix(hf.filepath, "/") ||
			strings.HasPrefix(hf.filepath, "~/") ||
			strings.HasPrefix(hf.filepath, "..")) {
		return true
	}
	return false
}
