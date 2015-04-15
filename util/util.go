package util

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetPasswd Only Available in *nix OS
func GetPasswd() []byte {
	ttyFile := "/dev/tty"
	tty, err := os.Open(ttyFile)
	passwdErrCallback := func(err error) {
		fmt.Printf("\nRead Password Error: %s\n", err.Error())
		os.Exit(2)
	}
	if err != nil {
		passwdErrCallback(err)
	}
	defer tty.Close()
	fd := int(tty.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		passwdErrCallback(err)
	}
	defer terminal.Restore(fd, oldState)
	passwd, _ := terminal.ReadPassword(fd)
	fmt.Println()
	return passwd
}

// WrapCmd returns `before && cmd && after`
func WrapCmd(cmd, before, after string) string {
	wrapped := cmd
	if after != "" {
		if wrapped != "" {
			wrapped = wrapped + " && " + after
		} else {
			wrapped = after
		}
	}
	if before != "" {
		if wrapped != "" {
			wrapped = before + " && " + wrapped
		} else {
			wrapped = before
		}
	}
	return wrapped
}

// WrapCmdBefore adds `before &&` before cmd
func WrapCmdBefore(cmd, before string) string {
	return WrapCmd(cmd, before, "")
}

// WrapCmdAfter appends `&& after` for cmd
func WrapCmdAfter(cmd, after string) string {
	return WrapCmd(cmd, "", after)
}

// IsDir gives whether @path is a directory
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.Mode().IsDir() {
		return true
	}
	return false
}

// FilePath gives the real file path. If is a link, gives its origin path.
func FilePath(file string) (real string, err error) {
	fi, err := os.Lstat(file)
	if err != nil {
		return
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		real, err = filepath.EvalSymlinks(file)
		if err != nil {
			return
		}
	} else {
		real = file
	}
	return
}

var spaceRegexp = regexp.MustCompile("\\s+")

// SplitBySpace split a string with /\s+/
func SplitBySpace(str string) []string {
	return spaceRegexp.Split(str, -1)
}

// JustifyText splits a string into array with each item has the same length.
func JustifyText(text string, width int) []string {
	if width <= 0 {
		return []string{}
	}
	start, end, residue, textLen := 0, width, 0, len(text)
	if end > textLen {
		end = textLen
	}
	ret := make([]string, 0, textLen/width+1)
	for start < textLen {
		slice := text[start:end]

		// Break Work with `-`
		// if len(ret) > 0 {
		// 	prev := ret[len(ret)-1]
		// 	if len(prev) == width && !strings.HasSuffix(prev, " ") && !strings.HasPrefix(slice, " ") {
		// 		ret[len(ret)-1] = prev + "-"
		// 	} else if end < textLen && strings.HasPrefix(slice, " ") {
		// 		start++
		// 		end++
		// 		slice = text[start:end]
		// 	}
		// }

		brokenSlice := strings.Split(slice, "\n")
		if brokenSliceLen := len(brokenSlice); brokenSliceLen > 1 {
			for j := 0; j < brokenSliceLen-1; j++ {
				ret = append(ret, brokenSlice[j])
			}
			residue = len(brokenSlice[brokenSliceLen-1])
		} else {
			// no "\n" in slice
			residue = 0
			if end < textLen {
				th := text[end-1 : end+1]
				// Do not break word at the end of the line, move to next line instead.
				if !strings.Contains(th, " ") {
					lastSpaceIndex := strings.LastIndex(slice, " ")
					if lastSpaceIndex > 0 {
						slice = slice[:lastSpaceIndex]
						residue = width - lastSpaceIndex - 1
					}
				}
			}
			ret = append(ret, slice)
		}
		start = start + width - residue
		end = start + width
		if end > textLen {
			end = textLen
		}
	}
	return ret
}
