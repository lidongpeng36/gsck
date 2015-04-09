package executor

import (
	"errors"
	"fmt"
	"github.com/EvanLi/gsck/formatter"
	"github.com/EvanLi/gsck/util"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
)

const concurrencyRecommend int64 = 60

var _executor *Executor
var constructorMap = map[string]WorkerConstructor{}
var outputChannel = make(chan *formatter.Output, 1)
var finishChannel = make(chan bool, 1)

func init() {
}

// GetExecutor returns the singleton *Executor
func GetExecutor() *Executor {
	if _executor == nil {
		usr, err := user.Current()
		var username string
		if err == nil {
			username = usr.Username
		}
		_executor = &Executor{
			User:        username,
			Hostlist:    make([]string, 0, 0),
			formatters:  make(map[string]formatter.Formatter),
			Concurrency: 1,
			err:         make([]error, 0, 1),
		}
	}
	return _executor
}

// Available returns all Worker's Name
func Available() []string {
	ret := make([]string, 0, len(constructorMap))
	for name := range constructorMap {
		ret = append(ret, name)
	}
	return ret
}

// package helper functions

// AddOutput informs new output to formatter
func AddOutput(output *formatter.Output) {
	if output != nil {
		if index, ok := _executor.indexMap[output.Hostname]; ok {
			output.Index = index
			outputChannel <- output
		}
	}
}

// Finish informs formatter that it should stop.
func Finish() {
	finishChannel <- true
}

// WrapCmdWithHook returns wrapped cmd, e.g. add `-a` and `-b` args
func WrapCmdWithHook(cmd string) string {
	if _executor.Transfer == nil || _executor.Transfer.hook == nil {
		return cmd
	}
	hook := _executor.Transfer.hook
	return util.WrapCmd(cmd, hook.before, hook.after)
}

// NeedTransferFile returns whether Worker should handle file copying.
func NeedTransferFile() bool {
	if _executor.Transfer != nil {
		trans := _executor.Transfer
		if trans.Dst != "" && trans.Data != nil {
			return true
		}
		return false
	}
	return false
}

// Worker should able to
// 1. execute cmd string
// 2. copy file (from *Executor.transferFile)
type Worker interface {
	Init() error
	Name() string
	Execute() error
}

// WorkerConstructor is function that receives no parameter and returns Worker
type WorkerConstructor func() Worker

// RegisterWorker used for plugins to register new Worker
func RegisterWorker(constructor WorkerConstructor) {
	name := constructor().Name()
	constructorMap[name] = constructor
}

type transferHook struct {
	before string
	after  string
}

// TransferFile describes file information that should be transferred.
type TransferFile struct {
	Data        []byte
	Perm        string
	Basename    string
	Destination string
	Src         string
	Dst         string
	hook        *transferHook
}

// Executor is a singleton class. Holds meta info for worker.
type Executor struct {
	User        string
	Passwd      string
	Cmd         string
	Account     string
	Concurrency int64
	Hostlist    []string
	Timeout     int64
	formatters  map[string]formatter.Formatter
	indexMap    map[string]int
	worker      Worker
	Transfer    *TransferFile
	err         []error
}

// SetUser sets user for execution, if user is not empty.
func (exec *Executor) SetUser(user string) *Executor {
	if user != "" {
		exec.User = user
	}
	return exec
}

// SetPasswd sets passwd for execution, without check or modification.
func (exec *Executor) SetPasswd(passwd string) *Executor {
	exec.Passwd = passwd
	return exec
}

// SetCmd sets cmd for execution, without check or modification.
func (exec *Executor) SetCmd(cmd string) *Executor {
	exec.Cmd = cmd
	return exec
}

// SetConcurrency sets concurrency for execution, without check or modification.
func (exec *Executor) SetConcurrency(concurrency int64) *Executor {
	exec.Concurrency = concurrency
	return exec
}

// SetTimeout sets timeout for execution, without check or modification.
func (exec *Executor) SetTimeout(timeout int64) *Executor {
	exec.Timeout = timeout
	return exec
}

// SetHostlist sets hostlist for execution, without check or modification.
func (exec *Executor) SetHostlist(list []string) *Executor {
	exec.Hostlist = list
	exec.indexMap = map[string]int{}
	for index, hostname := range list {
		exec.indexMap[hostname] = index
	}
	return exec
}

// AddFormatter adds formatter into a hash.
// Formatters are devided into categories, and each category can hold only one Formatter.
// Obviously, the key for hash is `category`.
func (exec *Executor) AddFormatter(category string, fmt formatter.Formatter) {
	exec.formatters[category] = fmt
}

// SetMethod sets concreate worker.
func (exec *Executor) SetMethod(method string) *Executor {
	if constructor, ok := constructorMap[method]; ok {
		exec.worker = constructor()
	} else {
		exec.err = append(exec.err, errors.New("No Such Method: "+method))
	}
	return exec
}

// SetTransfer sets source(local) and destination(remote) for file copy.
func (exec *Executor) SetTransfer(src, dst string) *Executor {
	if fi, err := os.Stat(src); err != nil {
		exec.err = append(exec.err, err)
	} else {
		perm := fmt.Sprintf("%#o", fi.Mode())
		data, err := ioutil.ReadFile(src)
		if err != nil {
			exec.err = append(exec.err, err)
		} else {
			basename := filepath.Base(src)
			dstPath := path.Join(dst, basename)
			exec.Transfer = &TransferFile{
				Data:        data,
				Perm:        perm,
				Basename:    basename,
				Destination: dst,
				Src:         src,
				Dst:         dstPath,
			}
		}
	}
	return exec
}

// SetTransferHook sets a hook, which would be executed before and after, for the file copying.
func (exec *Executor) SetTransferHook(before, after string) *Executor {
	if exec.Transfer == nil {
		exec.Transfer = new(TransferFile)
	}
	if before != "" || after != "" {
		exec.Transfer.hook = &transferHook{
			before: before,
			after:  after,
		}
	}
	return exec
}

// HostCount returns len(@Hostlist)
func (exec *Executor) HostCount() int {
	return len(exec.Hostlist)
}

// Check returns all errors that comes from initialization.
func (exec *Executor) Check() []error {
	return exec.err
}

// Run will initialize and drive worker and send output to Formatter(s)
func (exec *Executor) Run() (err error) {
	// Adjust concurrency
	con := exec.Concurrency
	count := int64(exec.HostCount())
	if con == 0 || con > concurrencyRecommend {
		con = concurrencyRecommend
	} else if exec.Concurrency < 0 {
		con = count
	}
	if con > count {
		con = count
	}
	exec.Concurrency = con

	done := make(chan bool, 1)
	go func() {
		for {
			select {
			case output := <-outputChannel:
				if output != nil {
					for _, fmt := range exec.formatters {
						fmt.Add(output)
					}
				}
			case <-finishChannel:
				for _, fmt := range exec.formatters {
					fmt.Print()
				}
				done <- true
			}
		}
	}()

	if exec.worker == nil {
		err = errors.New("No Execute Method Set.")
		goto end
	}

	// get worker to work and redirect its output to formatter
	err = exec.worker.Init()
	if err != nil {
		goto end
	}
	err = exec.worker.Execute()
end:
	Finish()
	<-done
	close(outputChannel)
	close(finishChannel)
	return
}
