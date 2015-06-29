package commander

import (
	"fmt"
	"github.com/EvanLi/gsck/config"
	"github.com/EvanLi/gsck/p2p"
	"github.com/EvanLi/gsck/util"
	"github.com/codegangsta/cli"
	"os"
)

func init() {
	RegisterCommand(cli.Command{
		Name:    "cp",
		Aliases: []string{"c"},
		Usage:   "Copy src to dst directory while keep files' perm",
		Flags: []cli.Flag{
			UserFlag,
			HostsFlag,
			MethodFlag,
			PreferFlag,
			PasswdFlag,
			WindowFlag,
			AccountFlag,
			TimeoutFlag,
			ConcurrencyFlag,
			cli.StringFlag{
				Name:  "src, s",
				Usage: "Source file/directory",
			},
			cli.StringFlag{
				Name:  "dst, d",
				Usage: "Destination DIRECTORY",
			},
			cli.StringFlag{
				Name:  "before, b",
				Usage: "CMD before copy",
			},
			cli.StringFlag{
				Name:  "after, a",
				Usage: "CMD after copy",
			},
		},
		Action: scpAction,
	})
}

// CP Action (gsck cp ...)
func scpAction(c *cli.Context) {
	PrepareExecutor(c)
	src := c.String("src")
	useScp := func() {
		exec.SetTransfer(src, c.String("dst"))
		exec.SetTransferHook(c.String("before"), c.String("after"))
	}
	useP2P := func() {
		p2pMgr := p2p.GetMgr()
		p2pMgr.SetTransfer(src, c.String("dst"))
		p2pMgr.Mkseed()
		if p2pMgr.NeedTransferFile() {
			remoteTmp := config.GetString("remote.tmpdir")
			exec.SetTransfer(p2pMgr.TransferFilePath(), remoteTmp)
		}
		exec.SetCmd(p2pMgr.ClientCmd()).SetConcurrency(-1)
	}
	if !p2p.Available() {
		useScp()
	} else if util.IsDir(src) {
		useP2P()
	} else {
		stat, err := os.Lstat(src)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		size := stat.Size()
		sizeInMB := size / 1024 / 1024
		transferTotal := sizeInMB * int64(exec.HostCount())
		// If transferTotal is less than 1 MB * 10 machine, use scp.
		if transferTotal < 10 {
			useScp()
		} else {
			useP2P()
		}
	}
	CheckExecutor()
	err, failed := exec.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	os.Exit(failed)
}
