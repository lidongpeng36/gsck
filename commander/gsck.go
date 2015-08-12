package commander

import (
	"fmt"
	"github.com/codegangsta/cli"
	"os"
)

// Default Action (gsck ...)
func action(c *cli.Context) {
	cmd := GetCmd(c)
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
