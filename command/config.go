package command

import (
	"fmt"
	"os"
	"path"

	"github.com/lidongpeng36/gsck/config"
	"github.com/urfave/cli"
	"github.com/mitchellh/go-homedir"
)

// SetupConfig will add `config` command
func SetupConfig(s *config.Setting) {
	if "" == s.Path {
		homeDir, _ := homedir.Dir()
		s.Path = homeDir
	}
	config.Setup(s)
	configPath := path.Join(s.Path, s.FileName)
	RegisterCommand(cli.Command{
		Name:   "config",
		Usage:  "Get/Set Defaults [ " + configPath + " ]",
		Action: configAction,
	})
}

// CONFIG Action (app config ...)
func configAction(c *cli.Context) {
	if 0 == len(c.Args()) {
		cli.ShowCommandHelp(c, "config")
		os.Exit(1)
	}
	key := c.Args()[0]
	if 1 == len(c.Args()) {
		fmt.Println(config.GetString(key))
	} else if 2 == len(c.Args()) {
		value := c.Args()[1]
		config.Set(key, value)
	}
}
