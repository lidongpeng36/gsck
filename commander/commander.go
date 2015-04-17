package commander

import (
	"fmt"
	"github.com/EvanLi/gsck/config"
	"github.com/EvanLi/gsck/executor"
	"github.com/EvanLi/gsck/formatter"
	"github.com/EvanLi/gsck/hostlist"
	"github.com/EvanLi/gsck/util"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"strings"
)

var app *cli.App
var commands []cli.Command
var exec *executor.Executor
var defaultUser string

func init() {
	app = cli.NewApp()
	app.Name = "gsck"
	app.Author = "lidongpeng36@gmail.com"
	app.Version = "1.2.1"
	app.Usage = "Execute commands on multiple machines over SSH (or other control system)"
	commands = make([]cli.Command, 0, 2)
	defaultUser = config.GetString("user")
	exec = executor.GetExecutor()
}

// RegisterCommand add new command to app
func RegisterCommand(cmd cli.Command) {
	commands = append(commands, cmd)
}

// HostsFlag `-f`
var HostsFlag = cli.StringFlag{
	Name:  "hosts, f",
	Usage: "Hostname List",
}

// MethodFlag `-m`
var MethodFlag = cli.StringFlag{
	Name:  "method, m",
	Value: "ssh",
}

// PreferFlag `--prefer`
var PreferFlag = cli.StringFlag{
	Name: "prefer",
}

// PasswdFlag `-p`
var PasswdFlag = cli.BoolFlag{
	Name:  "passwd, p",
	Usage: "Add Password Authentication Method",
}

// ConcurrencyFlag `-c`
var ConcurrencyFlag = cli.IntFlag{
	Name:  "concurrency, c",
	Value: 1,
	Usage: "Concurrency",
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
	EnvVar: "GSCK_USER,USER",
}

// AccountFlag `--account`
var AccountFlag = cli.StringFlag{
	Name:   "account",
	Usage:  "Some Executors need an Account to initiate",
	EnvVar: "GSCK_ACCOUNT",
}

// TimeoutFlag `-t`
var TimeoutFlag = cli.IntFlag{
	Name:  "timeout, t",
	Usage: "Timeout. Unit: s",
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
	PreferFlag.Usage = "Override default hostlist lookup chain"
	PreferFlag.Usage += "\n\tAnd set the only preferred method to get hostlist"
	PreferFlag.Usage += "\n\tHostlist Methods Available: " + hostlistAvail
	MethodFlag.Usage = "How to execute the commands"
	MethodFlag.Usage += "\n\tExecutor Methods Available: " + workerAvail
	app.Commands = commands
	setupMainCommand()
}

func getHostList(hostsArg string) (list []string, err error) {
	fi, _ := os.Stdin.Stat()
	// Read Data From Pipe
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		if hostsArg != "" {
			err = fmt.Errorf("Cannot read host list from pipe and set -f/--hosts at the same time.")
			return
		}
		bytes, _ := ioutil.ReadAll(os.Stdin)
		hostsArg = string(bytes)
		hostlist.SetPrefer("string")
	} else if hostsArg == "" {
		err = fmt.Errorf("Show me the host list.")
		return
	}
	list, err = hostlist.GetHostList(hostsArg)
	return
}

// PrepareExecutor fills Executor
func PrepareExecutor(c *cli.Context) {
	if err := hostlist.SetPrefer(c.String("prefer")); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	list, err := getHostList(c.String("hosts"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var passwd string
	if c.Bool("passwd") {
		fmt.Printf("Password: ")
		passwd = string(util.GetPasswd())
	}
	user := c.String("user")
	if user == "" {
		user = defaultUser
	}
	formatter.SetHostlist(list)
	formatter.SetInfo(user, int64(c.Int("concurrency")))
	if c.Bool("json") {
		exec.AddFormatter("merge", formatter.NewJSONFormatter())
	} else if c.Bool("window") {
		wf := formatter.NewWindowFormatter()
		exec.AddFormatter("rt", wf)
	} else {
		exec.AddFormatter("rt", formatter.NewAnsiFormatter())
	}
	exec.SetHostlist(list).SetConcurrency(int64(c.Int("concurrency")))
	exec.SetTimeout(int64(c.Int("timeout"))).SetMethod(c.String("method"))
	exec.SetUser(c.String("user")).SetPasswd(passwd).SetAccount(c.String("account"))
}

// CheckExecutor checks errors in executor. Exit if find any.
func CheckExecutor() {
	errs := exec.Check()
	if len(errs) > 0 {
		fmt.Println("Init Error:")
		for _, e := range errs {
			fmt.Println(e.Error())
		}
		os.Exit(2)
	}
}

// App returns package variable app
func App() *cli.App {
	return app
}

// Run starts gsck
func Run() {
	app.Run(os.Args)
}
