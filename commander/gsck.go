package commander

import (
	"fmt"
	"github.com/codegangsta/cli"
	"os"
)

// Default Action (gsck ...)
func action(c *cli.Context) {
	if len(c.Args()) != 1 {
		fmt.Println("Command Must Be the Last One Parameter!")
		cli.ShowAppHelp(c)
		os.Exit(1)
	}
	cmd := c.Args()[0]
	PrepareExecutor(c)
	exec.SetCmd(cmd)
	CheckExecutor()
	err, failed := exec.Run()
	if err != nil {
		fmt.Println("Execute Error: ", err)
		os.Exit(2)
	}
	os.Exit(failed)
}

func setupMainCommand() {
	app.Flags = []cli.Flag{
		JSONFlag,
		UserFlag,
		HostsFlag,
		PreferFlag,
		PasswdFlag,
		WindowFlag,
		MethodFlag,
		AccountFlag,
		TimeoutFlag,
		ConcurrencyFlag,
	}
	app.Action = action
}
