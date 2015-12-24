package commander

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/EvanLi/gsck/command"
	"github.com/EvanLi/gsck/executor"
	"github.com/EvanLi/gsck/formatter"
	"github.com/EvanLi/gsck/hostlist"
	"github.com/EvanLi/gsck/util"
	"github.com/codegangsta/cli"
)

// Input From Pipe/HereDoc
const (
	PIPEEMTPY          = ""
	PIPEUSEDBYHOSTLIST = "_PIPE_USED_BY_HOSTLIST"
	PIPEUSEDBYCMD      = "_PIPE_USED_BY_CMD"
)

// var exec *executor.Executor

// HostsFlag `-f`
var HostsFlag = cli.StringFlag{
	Name:  "hosts, f",
	Usage: "Hostname List",
}

// MethodFlag `-m`
var MethodFlag = cli.StringFlag{
	Name:   "method, m",
	Value:  "ssh",
	EnvVar: "METHOD",
	Usage:  "How to execute the commands",
}

// PreferFlag `--prefer`
var PreferFlag = cli.StringFlag{
	Name:   "prefer",
	EnvVar: "PREFER",
	Usage:  "Override default hostlist lookup chain",
}

// PasswdFlag `-p`
var PasswdFlag = cli.BoolFlag{
	Name:  "passwd, p",
	Usage: "Add Password Authentication Method",
}

// PasswordFlag `--password`
var PasswordFlag = cli.StringFlag{
	Name:   "password",
	EnvVar: "PASSWORD",
	Usage:  "Specify password in Command Line",
}

// ConcurrencyFlag `-c`
var ConcurrencyFlag = cli.IntFlag{
	Name:   "concurrency, c",
	Value:  1,
	Usage:  "Concurrency",
	EnvVar: "CONCURRENCY",
}

// JSONFlag `-j`
var JSONFlag = cli.BoolFlag{
	Name:  "json, j",
	Usage: "Prints in JSON",
}

// UserFlag `-u`
var UserFlag = cli.StringFlag{
	Name:   "user, u",
	Usage:  "Username",
	EnvVar: "USER,USER",
}

// AccountFlag `--account`
var AccountFlag = cli.StringFlag{
	Name:   "account",
	Usage:  "Some Executors need an Account to initiate",
	EnvVar: "ACCOUNT",
}

// TimeoutFlag `-t`
var TimeoutFlag = cli.IntFlag{
	Name:  "timeout, t",
	Usage: "Timeout. Unit: s",
	Value: 0,
}

// RetryFlag `--retry`
var RetryFlag = cli.IntFlag{
	Name:  "retry",
	Usage: "Retry",
	Value: 0,
}

// WindowFlag `-w`
var WindowFlag = cli.BoolFlag{
	Name:  "window, w",
	Usage: "Vim-like Command-Line User Interface",
}

// Init collects all Commands
func Init() {
	hostlistAvail := strings.Join(hostlist.Available(), ", ")
	workerAvail := strings.Join(executor.Available(), ", ")
	PreferFlag.Usage += "\n\tAnd set the only preferred method to get hostlist"
	PreferFlag.Usage += "\n\tHostlist Methods Available: " + hostlistAvail
	MethodFlag.Usage += "\n\tExecutor Methods Available: " + workerAvail
}

func getPipe(used string) (data string) {
	app := command.Instance()
	switch app.Pipe {
	case "":
		fmt.Printf("Cannot Get %s: Pipe/HereDoc Empty.\n", strings.Split(used, "USED_BY_")[1])
	case PIPEUSEDBYHOSTLIST, PIPEUSEDBYCMD:
		fmt.Printf("Cannot Get %s From Pipe/HereDoc since it's used by *%s*.\n", strings.Split(used, "USED_BY_")[1], strings.Split(app.Pipe, "USED_BY_")[1])
	default:
		data = app.Pipe
		app.Pipe = used
		return
	}
	os.Exit(1)
	return
}

// GetHostList gets hostlist from flag/pipe
//   @hostsArg: argument for hostlist to generate HostInfoList
//   @prefer: preferred method to get hostlist(PreferFlag)
func GetHostList(hostsArg, prefer string) (list hostlist.HostInfoList, err error) {
	app := command.Instance()
	if hostsArg == "" {
		if app.Fifo != "" {
			hostsArg = app.Fifo
		} else {
			hostsArg = getPipe(PIPEUSEDBYHOSTLIST)
		}
	}
	if hostsArg == "" {
		err = fmt.Errorf("Show me the host list.")
		return
	}
	list, err = hostlist.GetHostList(hostsArg, prefer)
	return
}

// GetCmd extracts cmd from command line
func GetCmd(c *cli.Context) (cmd string) {
	argc := len(c.Args())
	app := command.Instance()
	if argc >= 2 {
		if app.FifoFile == "" || argc > 2 {
			fmt.Println("Too Many Arguments (Command Must Be the Last One Parameter) !")
			cli.ShowAppHelp(c)
			os.Exit(1)
		}
	loop:
		for _, arg := range c.Args() {
			if arg != app.FifoFile {
				cmd = arg
				break loop
			}
		}
	} else if argc == 1 && c.Args()[0] == app.FifoFile || argc == 0 {
		cmd = getPipe(PIPEUSEDBYCMD)
		return
	} else {
		cmd = c.Args()[argc-1]
	}
	return
}

// SetupFormatter *MUST* be called after hostlist is ready
func SetupFormatter(c *cli.Context, exe *executor.Executor) (err error) {
	if nil == exe {
		return errors.New("Cannot SetupFormatter for nil")
	}
	if nil == exe.Parameter.HostInfoList {
		return errors.New("Cannot SetupFormatter Before Hostlist is set")
	}
	user := exe.Parameter.User
	formatter.SetInfo(user, int64(c.Int("concurrency")))
	if c.Bool("json") {
		exe.AddFormatter("merge", formatter.NewJSONFormatter())
	} else if c.Bool("window") {
		wf := formatter.NewWindowFormatter()
		exe.AddFormatter("rt", wf)
	} else {
		exe.AddFormatter("rt", formatter.NewAnsiFormatter())
	}
	return
}

// SetupParameter translates cli flags into Parameter
func SetupParameter(c *cli.Context) executor.Parameter {
	var passwd string
	passwd = c.String("password")
	if c.Bool("passwd") {
		fmt.Printf("Password: ")
		passwd = string(util.GetPasswd())
	}
	return executor.Parameter{
		User:        c.String("user"),
		Passwd:      passwd,
		Account:     c.String("account"),
		Concurrency: int64(c.Int("concurrency")),
		Timeout:     int64(c.Int("timeout")),
		Method:      c.String("method"),
		Retry:       1,
	}
}

// PrepareExecutor fills Executor
func PrepareExecutor(c *cli.Context) *executor.Executor {
	list, err := GetHostList(c.String("hosts"), c.String("prefer"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var exec *executor.Executor
	exec, err = executor.NewExecutor(SetupParameter(c))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	exec.SetHostInfoList(list)
	SetupFormatter(c, exec)
	return exec
}

// Run starts gsck
func Run() error {
	return command.Run()
}
