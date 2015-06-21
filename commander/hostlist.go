package commander

import (
	"fmt"
	"github.com/codegangsta/cli"
	"os"
)

func init() {
	RegisterCommand(cli.Command{
		Name:   "host",
		Usage:  "Show host list",
		Action: hostAction,
		Flags: []cli.Flag{
			PreferFlag,
		},
	})
}

// HOST Action (gsck host ...)
func hostAction(c *cli.Context) {
	if len(c.Args()) != 1 {
		cli.ShowCommandHelp(c, "host")
		os.Exit(1)
	}
	list, err := GetHostList(c.Args()[0], c.String("prefer"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, host := range list {
		fmt.Printf("%s\n", host.Alias)
	}
}
