package main

import (
	"fmt"
	"os"

	"github.com/lidongpeng36/gsck/command"
	"github.com/lidongpeng36/gsck/commander"
	"github.com/lidongpeng36/gsck/config"
	"github.com/urfave/cli"
)

var envPrefix = "GSCK"

func setupConfig() {
	setting := &config.Setting{
		FileName:  ".gsckconfig",
		EnvPrefix: envPrefix,
		Defaults: map[string]string{
			"user":          os.Getenv("USER"),
			"retry":         "2",
			"method":        "ssh",
			"concurrency":   "1",
			"formatter":     "ansi",
			"local.tmpdir":  "/tmp",
			"remote.tmpdir": "/tmp",
			"json.pretty":   "true",
		},
	}
	command.SetupConfig(setting)
}

func init() {
	setupConfig()
	app := command.Instance()
	app.Name = "gsck"
	app.Authors = []cli.Author{cli.Author{Name: "Li Dongpeng", Email: "lidongpeng36@gmail.com"}}
	app.Version = "4.0.0"
	app.Usage = "Execute commands on multiple machines over SSH (or other control system)"
	command.EnvPrefix = envPrefix
}

// Default Action (gsck ...)
func action(c *cli.Context) {
	if 1 == len(os.Args) {
		cli.ShowAppHelp(c)
		os.Exit(0)
	}
	cmd := commander.GetCmd(c)
	exec := commander.PrepareExecutor(c)
	exec.Parameter.Cmd = cmd
	failed, err := exec.Run()
	if err != nil {
		fmt.Println("Execute Error: ", err)
		os.Exit(2)
	}
	os.Exit(failed)
}

func setupMainCommand() {
	app := command.Instance()
	app.Flags = []cli.Flag{
		commander.JSONFlag,
		commander.UserFlag,
		commander.HostsFlag,
		commander.PreferFlag,
		commander.PasswdFlag,
		commander.WindowFlag,
		commander.MethodFlag,
		commander.AccountFlag,
		commander.TimeoutFlag,
		commander.PasswordFlag,
		commander.ConcurrencyFlag,
	}
	app.Action = action
}

func main() {
	command.UseCommand("hostlist", "copy", "config")
	commander.Init()
	setupMainCommand()
	commander.Run()
}
