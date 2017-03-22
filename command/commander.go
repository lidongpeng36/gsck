package command

import (
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/codegangsta/cli"
)

type App struct {
	Pipe     string
	Fifo     string
	FifoFile string
	Args     []string
	*cli.App
}

// Input From Pipe/HereDoc
const (
	PIPEEMTPY = "_PIPE_EMPTY"
)

var app *App
var commands map[string]cli.Command
var commandInUse []cli.Command

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	app = &App{
		App: cli.NewApp(),
	}
	commands = make(map[string]cli.Command)
	commandInUse = make([]cli.Command, 0, 10)

	// Read Data From Pipe/HereDoc
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & (os.ModeCharDevice | os.ModeDir)) == 0 {
		bytes, _ := ioutil.ReadAll(os.Stdin)
		app.Pipe = string(bytes)
	}
	// Read Data From FIFO
	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		fii, err := os.Stat(arg)
		if err == nil && (fii.Mode()&os.ModeNamedPipe) != 0 {
			bytes, _ := ioutil.ReadFile(arg)
			app.Fifo = string(bytes)
			app.FifoFile = arg
		}
	}
}

// RegisterCommand add new Available command
func RegisterCommand(cmd cli.Command) {
	commands[cmd.Name] = cmd
}

// UseCommand tells app which commands it will use
func UseCommand(cmd ...string) {
	for _, name := range cmd {
		if cmd, ok := commands[name]; ok {
			commandInUse = append(commandInUse, cmd)
		}
	}
}

// Instance returns package variable app
func Instance() *App {
	return app
}

// Change Env Flag for cli. e.g. USER -> GSCK_USER
func changeFlagEnv(env string, flags *[]cli.Flag) {
	if nil != flags {
		for i := range *flags {
			f := (*flags)[i]
			fv := reflect.ValueOf(f)
			origEnv := fv.FieldByName("EnvVar").String()
			if "" == origEnv {
				continue
			}
			newEnv := env + "_" + origEnv
			newfs := reflect.New(fv.Type())
			fe := newfs.Elem()
			for i := 0; i < fv.NumField(); i++ {
				fe.Field(i).Set(fv.Field(i))
			}
			fe.FieldByName("EnvVar").Set(reflect.ValueOf(newEnv))
			(*flags)[i] = fe.Interface().(cli.Flag)
		}
	}

}

func bindEnvPrefix(env string, commands *cli.Commands) {
	for i := range *commands {
		ptr := &(*commands)[i]
		subs := &ptr.Subcommands
		flags := &ptr.Flags
		if nil != subs {
			bindEnvPrefix(env, subs)
		}
		if nil != flags {
			changeFlagEnv(env, flags)
		}
	}
}

// work around
func appBindEnvPrefix(env string, commands *[]cli.Command) {
	for i := range *commands {
		ptr := &(*commands)[i]
		subs := &ptr.Subcommands
		flags := &ptr.Flags
		if nil != subs {
			bindEnvPrefix(env, subs)
		}
		if nil != flags {
			changeFlagEnv(env, flags)
		}
	}
}

var EnvPrefix string

// Run starts app
func Run() error {
	EnvPrefix = strings.ToUpper(EnvPrefix)
	app.Commands = commandInUse
	if "" != EnvPrefix {
		appBindEnvPrefix(EnvPrefix, &(app.Commands))
		changeFlagEnv(EnvPrefix, &app.Flags)
	}
	RunSignal()
	return app.Run(os.Args)
}
