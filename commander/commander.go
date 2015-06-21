package commander

import (
	"errors"
	"fmt"
	"github.com/EvanLi/gsck/config"
	"github.com/EvanLi/gsck/executor"
	"github.com/EvanLi/gsck/formatter"
	"github.com/EvanLi/gsck/hostlist"
	"github.com/EvanLi/gsck/sig"
	"github.com/EvanLi/gsck/util"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

var app *cli.App
var commands []cli.Command
var exec *executor.Executor
var defaultUser string

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	app = cli.NewApp()
	app.Name = "gsck"
	app.Author = "lidongpeng36@gmail.com"
	app.Version = "2.0.0"
	app.Usage = "Execute commands on multiple machines over SSH (or other control system)"
	commands = make([]cli.Command, 0, 10)
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
	Name:   "method, m",
	Value:  "ssh",
	EnvVar: "GSCK_METHOD",
}

// PreferFlag `--prefer`
var PreferFlag = cli.StringFlag{
	Name:   "prefer",
	EnvVar: "GSCK_PREFER",
}

// PasswdFlag `-p`
var PasswdFlag = cli.BoolFlag{
	Name:  "passwd, p",
	Usage: "Add Password Authentication Method",
}

// ConcurrencyFlag `-c`
var ConcurrencyFlag = cli.IntFlag{
	Name:   "concurrency, c",
	Value:  1,
	Usage:  "Concurrency",
	EnvVar: "GSCK_CONCURRENCY",
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

// GetHostList returns HostInfoList
// @hostsArg: argument for hostlist to generate HostInfoList
func GetHostList(hostsArg, prefer string) (list hostlist.HostInfoList, err error) {
	fi, _ := os.Stdin.Stat()
	// Read Data From Pipe
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		if hostsArg != "" {
			err = fmt.Errorf("Cannot read host list from pipe and set -f/--hosts at the same time.")
			return
		}
		bytes, _ := ioutil.ReadAll(os.Stdin)
		hostsArg = string(bytes)
		_ = hostlist.SetPrefer("string")
	} else if hostsArg == "" {
		err = fmt.Errorf("Show me the host list.")
		return
	}
	list, err = hostlist.GetHostList(hostsArg, prefer)
	return
}

// CheckCmd checks last parameter
func CheckCmd(c *cli.Context) string {
	if len(c.Args()) != 1 {
		fmt.Println("Command Must Be the Last One Parameter!")
		cli.ShowAppHelp(c)
		os.Exit(1)
	}
	return c.Args()[0]
}

// SetupFormatter *MUST* be called after hostlist is ready
func SetupFormatter(c *cli.Context) {
	if exec.Data.HostInfoList == nil {
		panic(errors.New("Cannot SetupFormatter Before Hostlist is set!"))
	}
	user := c.String("user")
	formatter.SetInfo(user, int64(c.Int("concurrency")))
	if c.Bool("json") {
		exec.AddFormatter("merge", formatter.NewJSONFormatter())
	} else if c.Bool("window") {
		wf := formatter.NewWindowFormatter()
		exec.AddFormatter("rt", wf)
	} else {
		exec.AddFormatter("rt", formatter.NewAnsiFormatter())
	}
}

// PrepareExecutorExceptHostlist fills all fields but hostlist
func PrepareExecutorExceptHostlist(c *cli.Context) {
	var passwd string
	if c.Bool("passwd") {
		fmt.Printf("Password: ")
		passwd = string(util.GetPasswd())
	}
	user := c.String("user")
	if user == "" {
		user = defaultUser
	}
	exec.SetConcurrency(int64(c.Int("concurrency")))
	exec.SetTimeout(int64(c.Int("timeout"))).SetMethod(c.String("method"))
	exec.SetUser(c.String("user")).SetPasswd(passwd).SetAccount(c.String("account"))
}

// PrepareExecutor fills Executor
func PrepareExecutor(c *cli.Context) {
	// if err := hostlist.SetPrefer(c.String("prefer")); err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	list, err := GetHostList(c.String("hosts"), c.String("prefer"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	PrepareExecutorExceptHostlist(c)
	exec.SetHostInfoList(list)
	SetupFormatter(c)
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
	sig.Run()
	_ = app.Run(os.Args)
}
