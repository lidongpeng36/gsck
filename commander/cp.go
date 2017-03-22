package commander

import (
	"fmt"
	"os"

	"github.com/lidongpeng36/gsck/command"
	"github.com/lidongpeng36/gsck/config"
	"github.com/lidongpeng36/gsck/p2p"
	"github.com/lidongpeng36/gsck/util"
	"github.com/codegangsta/cli"
)

func init() {
	command.RegisterCommand(cli.Command{
		Name:    "copy",
		Aliases: []string{"cp", "c"},
		Usage:   "Copy src to dst directory while keep perm",
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
	exec := PrepareExecutor(c)
	src := c.String("src")
	useScp := func() {
		exec.SetTransfer(src, c.String("dst"))
		exec.SetTransferHook(c.String("before"), c.String("after"))
	}
	useP2P := func() {
		p2pMgr := p2p.GetMgr()
		p2pMgr.SetTransfer(src, c.String("dst"))
		err := p2pMgr.Mkseed()
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		if p2pMgr.NeedTransferFile() {
			remoteTmp := config.GetString("remote.tmpdir")
			exec.SetTransfer(p2pMgr.TransferFilePath(), remoteTmp)
		}
		cmd := util.WrapCmd(p2pMgr.ClientCmd(), c.String("before"), c.String("after"))
		exec.Parameter.Cmd = cmd
		exec.Parameter.Concurrency = -1
	}
	if !p2p.Available() {
		useScp()
	} else if util.IsDir(src) {
		useP2P()
	} else {
		stat, err := os.Lstat(src)
		if nil != err {
			fmt.Println(err)
			os.Exit(2)
		}
		size := stat.Size()
		sizeInMB := size / 1024 / 1024
		transferTotal := sizeInMB * int64(exec.HostCount())
		// If transferTotal is less than 1 MB * 10 machine, use scp.
		if 10 > transferTotal {
			useScp()
		} else {
			useP2P()
		}
	}
	failed, err := exec.Run()
	if nil != err {
		fmt.Println(err)
		os.Exit(2)
	}
	os.Exit(failed)
}
