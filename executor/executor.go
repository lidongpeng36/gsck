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

// GetExecutor initiate, if it's nil, and returns _executor
var _executor *Executor
var constructorMap = map[string]WorkerConstructor{}

// NewExecutor returns an empty Executor
func NewExecutor() *Executor {
	usr, err := user.Current()
	var username string
	if err == nil {
		username = usr.Username
	}
	return &Executor{
		Data: &Data{
			User:          username,
			Hostlist:      make([]string, 0, 0),
			Concurrency:   1,
			indexMap:      map[string]int{},
			outputChannel: make(chan formatter.Output, 0),
		},
		formatters:    make(map[string]formatter.Formatter),
		err:           make([]error, 0, 1),
		finishChannel: make(chan bool, 0),
	}
}

// GetExecutor returns the singleton *Executor
func GetExecutor() *Executor {
	if _executor == nil {
		_executor = NewExecutor()
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

// Worker should able to
// 1. execute cmd string
// 2. copy file (from *Executor.transferFile)
type Worker interface {
	Init(*Data) error
	Name() string
	Execute() error
}

// WorkerWithRecommendedConcurrency could give its recommended concurrency
type WorkerWithRecommendedConcurrency interface {
	// Return
	// <= 0: Recommended Concurrency == Hostlist Length
	RecommendedConcurrency() int64
	Worker
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

// Data holds data for worker
type Data struct {
	User          string
	Passwd        string
	Cmd           string
	Account       string
	Concurrency   int64
	Hostlist      []string
	Timeout       int64
	Transfer      *TransferFile
	indexMap      map[string]int
	outputChannel chan formatter.Output
}

// WrapCmdWithHook returns wrapped cmd, e.g. add `-a` and `-b` args
func (data *Data) WrapCmdWithHook(cmd string) string {
	if data.Transfer == nil || data.Transfer.hook == nil {
		return cmd
	}
	hook := data.Transfer.hook
	return util.WrapCmd(cmd, hook.before, hook.after)
}

// NeedTransferFile returns whether Worker should handle file copying.
func (data *Data) NeedTransferFile() bool {
	if data.Transfer != nil {
		trans := data.Transfer
		if trans.Dst != "" && trans.Data != nil {
			return true
		}
		return false
	}
	return false
}

// AddOutput informs formatter with new output
func (data *Data) AddOutput(output formatter.Output) {
	if index, ok := data.indexMap[output.Hostname]; ok {
		output.Index = index
		data.outputChannel <- output
	}
}

// Executor is a singleton class. Holds meta info for worker.
type Executor struct {
	Data          *Data
	formatters    map[string]formatter.Formatter
	Worker        Worker
	err           []error
	finishChannel chan bool
}

// SetUser sets user for execution, if user is not empty.
func (exec *Executor) SetUser(user string) *Executor {
	if user != "" {
		exec.Data.User = user
	}
	return exec
}

// SetAccount sets account for execution, if account is not empty.
func (exec *Executor) SetAccount(account string) *Executor {
	if account != "" {
		exec.Data.Account = account
	}
	return exec
}

// SetPasswd sets passwd for execution, without check or modification.
func (exec *Executor) SetPasswd(passwd string) *Executor {
	exec.Data.Passwd = passwd
	return exec
}

// SetCmd sets cmd for execution, without check or modification.
func (exec *Executor) SetCmd(cmd string) *Executor {
	exec.Data.Cmd = cmd
	return exec
}

// SetConcurrency sets concurrency for execution, without check or modification.
func (exec *Executor) SetConcurrency(concurrency int64) *Executor {
	exec.Data.Concurrency = concurrency
	return exec
}

// SetTimeout sets timeout for execution, without check or modification.
func (exec *Executor) SetTimeout(timeout int64) *Executor {
	exec.Data.Timeout = timeout
	return exec
}

// SetHostlist sets hostlist for execution, without check or modification.
func (exec *Executor) SetHostlist(list []string) *Executor {
	exec.Data.Hostlist = list
	for index, hostname := range list {
		exec.Data.indexMap[hostname] = index
	}
	return exec
}

// AddFormatter adds formatter into a hash.
// Formatters are devided into categories, and each category can hold only one Formatter.
// Obviously, the key for hash is `category`.
func (exec *Executor) AddFormatter(category string, fmt formatter.Formatter) *Executor {
	exec.formatters[category] = fmt
	return exec
}

// SetMethod sets concreate worker.
func (exec *Executor) SetMethod(method string) *Executor {
	if method == "" {
		method = "ssh"
	}
	if constructor, ok := constructorMap[method]; ok {
		exec.Worker = constructor()
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
			exec.Data.Transfer = &TransferFile{
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
	if exec.Data.Transfer == nil {
		exec.Data.Transfer = new(TransferFile)
	}
	if before != "" || after != "" {
		exec.Data.Transfer.hook = &transferHook{
			before: before,
			after:  after,
		}
	}
	return exec
}

// HostCount returns len(@Hostlist)
func (exec *Executor) HostCount() int {
	return len(exec.Data.Hostlist)
}

// Check returns all errors that comes from initialization.
func (exec *Executor) Check() []error {
	return exec.err
}

// Run will initialize and drive worker and send output to Formatter(s)
func (exec *Executor) Run() (err error) {

	count := int64(exec.HostCount())
	if count == 0 {
		err = errors.New("Executor cannot Run: Empty Hostlist")
	}
	con := exec.Data.Concurrency
	recommendConcurrency := concurrencyRecommend

	done := make(chan bool, 0)
	go func() {
	loop:
		for {
			select {
			case output := <-exec.Data.outputChannel:
				for _, fmt := range exec.formatters {
					fmt.Add(output)
				}
			case <-exec.finishChannel:
				for _, fmt := range exec.formatters {
					fmt.Print()
				}
				break loop
			}
		}
		done <- true
	}()

	if exec.Worker == nil {
		err = errors.New("No Execute Method Set.")
		goto end
	}

	// Adjust concurrency
	if w, ok := exec.Worker.(WorkerWithRecommendedConcurrency); ok {
		recommendConcurrency = w.RecommendedConcurrency()
	}
	if recommendConcurrency <= 0 {
		recommendConcurrency = count
	}
	if con == 0 || con > recommendConcurrency {
		con = recommendConcurrency
	} else if con < 0 {
		con = count
	}
	if con > count {
		con = count
	}
	exec.Data.Concurrency = con

	// get worker to work and redirect its output to formatter
	if err = exec.Worker.Init(exec.Data); err != nil {
		goto end
	}
	err = exec.Worker.Execute()
end:
	exec.finishChannel <- true
	<-done
	close(done)
	close(exec.finishChannel)
	close(exec.Data.outputChannel)
	return
}
