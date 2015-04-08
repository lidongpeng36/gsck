package util

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"path/filepath"
	"regexp"
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
