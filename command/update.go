package command

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/kardianos/osext"
)

// FilePath gives the real file path. If is a link, gives its origin path.
func FilePath(file string) (real string, err error) {
	fi, err := os.Lstat(file)
	if nil != err {
		return
	}
	if os.ModeSymlink == fi.Mode()&os.ModeSymlink {
		real, err = filepath.EvalSymlinks(file)
		if nil != err {
			return
		}
	} else {
		real = file
	}
	return
}

var binURL string

// SetupUpdate will add an `update` command, to update self from a http url
func SetupUpdate(src string) {
	binURL = src
	RegisterCommand(cli.Command{
		Name:   "update",
		Usage:  "Update to latest version from " + src,
		Action: updateAction,
	})
}

// UPDATE Action (app update)
func updateAction(c *cli.Context) {
	currentPath, err := osext.Executable()
	if nil != err {
		fmt.Println("Cannot get current binary's path")
		return
	}
	realPath, err := FilePath(currentPath)
	if nil != err {
		return
	}
	fi, err := os.Stat(realPath)
	if nil != err {
		fmt.Println("Cannot stat ", realPath, " !")
		return
	}
	uid := fi.Sys().(*syscall.Stat_t).Uid
	gid := fi.Sys().(*syscall.Stat_t).Gid
	tmpDir := path.Dir(realPath)
	tmpFilePath := filepath.Join(tmpDir, Instance().Name+".new")
	tmpFile, err := os.Create(tmpFilePath)
	if nil != err {
		fmt.Println("Cannot create tmp file ", tmpFilePath, ". Error: ", err)
		return
	}
	defer tmpFile.Close()
	resp, err := http.Get(binURL)
	if nil != err {
		fmt.Println("Download Error: ", err)
		return
	}
	defer resp.Body.Close()
	_, err = io.Copy(tmpFile, resp.Body)
	if nil != err {
		fmt.Println("Cannot save tmp file: ", err)
		return
	}
	tmpFile.Chmod(fi.Mode())
	tmpFile.Chown(int(uid), int(gid))
	tmpFile.Close()
	err = os.Rename(tmpFilePath, realPath)
	if nil != err {
		fmt.Println("mv file error: ", err)
	}
	return
}
