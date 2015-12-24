package executor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/EvanLi/gsck/formatter"
	"github.com/EvanLi/gsck/hostlist"
	"github.com/EvanLi/gsck/util"
)

func debug(prefix, format string, v ...interface{}) {
}

const concurrencyRecommend int64 = 60

// GetExecutor initiate, if it's nil, and returns _executor
var constructorMap = map[string]WorkerConstructor{}

// NewExecutor returns an empty Executor
func NewExecutor(p Parameter) (exec *Executor, err error) {
	p1 := p
	setupParameter(&p1)
	exec = &Executor{
		Parameter:  &p1,
		indexMap:   make(map[string]int),
		formatters: make(map[string]formatter.Formatter),
	}
	if nil != p1.HostInfoList {
		exec.SetHostInfoList(p1.HostInfoList)
	}

	if constructor, ok := constructorMap[p1.Method]; ok {
		exec.worker = constructor()
	} else {
		return nil, errors.New("No Such Method: " + p1.Method)
	}
	return
}

func setupParameter(p *Parameter) {
	if "" == p.User {
		usr, err := user.Current()
		var username string
		if err == nil {
			username = usr.Username
		}
		p.User = username
	}
	if p.Method == "" {
		p.Method = "ssh"
	}
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
	Init(*Parameter) error
	Name() string
	Execute(done <-chan struct{}) (<-chan *formatter.Output, <-chan error)
}

// WorkerWithRecommendedConcurrency could give its recommended concurrency
type WorkerWithRecommendedConcurrency interface {
	// @Return
	//   == 0: Recommended Concurrency
	//   < 0: Hostlist Length
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
	Destination string // Destination DIRECTORY
	Src         string // Final Destination, Destination/Basename
	Dst         string
	hook        *transferHook
}

// Parameter holds data for worker
type Parameter struct {
	Cmd          string
	User         string
	Passwd       string
	Script       string
	Account      string
	Method       string
	Retry        int
	Concurrency  int64
	Hostlist     []string
	HostInfoList hostlist.HostInfoList
	Timeout      int64
	Transfer     *TransferFile
}

// WrapCmdWithHook returns wrapped cmd, e.g. add `-a` and `-b` args
func (data *Parameter) WrapCmdWithHook(cmd string) string {
	if data.Transfer == nil || data.Transfer.hook == nil {
		return cmd
	}
	hook := data.Transfer.hook
	cmdAfter := hook.after
	if cmdAfter != "" {
		cmdAfter = fmt.Sprintf("cd %s && %s", data.Transfer.Destination, cmdAfter)
	}
	return util.WrapCmd(cmd, hook.before, cmdAfter)
}

// NeedTransferFile returns whether Worker should handle file copying.
func (data *Parameter) NeedTransferFile() bool {
	if data.Transfer != nil {
		trans := data.Transfer
		if trans.Dst != "" && trans.Data != nil {
			return true
		}
		return false
	}
	return false
}

// Executor is a singleton class. Holds meta info for worker.
type Executor struct {
	Parameter  *Parameter
	worker     Worker
	indexMap   map[string]int
	formatters map[string]formatter.Formatter
	err        []error
}

// SetHostlist sets hostlist for execution, without check or modification.
// DEPRECATED
func (exec *Executor) SetHostlist(list []string) *Executor {
	exec.Parameter.Hostlist = list
	if exec.Parameter.HostInfoList == nil {
		exec.SetHostInfoList(hostlist.MakeHostInfoListFromStringList(list))
	}
	return exec
}

// SetHostInfoList is extended version of SetHostlist.
// Executor will adjust it at the beginning of Run, which would set correct User & Cmd.
func (exec *Executor) SetHostInfoList(list hostlist.HostInfoList) *Executor {
	if list != nil {
		exec.Parameter.HostInfoList = list
		for i, info := range list {
			exec.indexMap[info.Alias] = i
		}
		formatter.SetHostInfoList(&list)
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
			exec.Parameter.Transfer = &TransferFile{
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
	if exec.Parameter.Transfer == nil {
		exec.Parameter.Transfer = new(TransferFile)
	}
	if before != "" || after != "" {
		exec.Parameter.Transfer.hook = &transferHook{
			before: before,
			after:  after,
		}
	}
	return exec
}

// HostCount returns length of HostInfoList
func (exec *Executor) HostCount() int {
	if exec.Parameter.HostInfoList != nil {
		return len(exec.Parameter.HostInfoList)
	}
	return 0
}

func (exec *Executor) integration() (err error) {
	// Worker
	if exec.worker == nil {
		err = errors.New("No Execute Method Set.")
		return
	}
	// Concurrency
	count := int64(exec.HostCount())
	if count == 0 {
		err = errors.New("Executor cannot Run: Empty Hostlist")
		return
	}
	con := exec.Parameter.Concurrency
	recommendConcurrency := concurrencyRecommend

	if w, ok := exec.worker.(WorkerWithRecommendedConcurrency); ok {
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
	exec.Parameter.Concurrency = con

	// Set User & Cmd of HostInfoList
	for _, hi := range exec.Parameter.HostInfoList {
		if hi.Cmd == "" {
			hi.Cmd = exec.Parameter.Cmd
		}
		if hi.User == "" {
			hi.User = exec.Parameter.User
		}
	}
	return
}

// Run will initialize and drive worker and send output to Formatter(s)
func (exec *Executor) Run() (failed int, err error) {

	if err = exec.integration(); err != nil {
		return
	}

	if err = exec.worker.Init(exec.Parameter); err != nil {
		return
	}
	done := make(chan struct{})

	ch, errc := exec.worker.Execute(done)

	defer func() {
		close(done)
		for _, f := range exec.formatters {
			f.Print()
		}
	}()

	for {
		select {
		case o := <-ch:
			if nil == o {
				return
			}
			if 0 != o.ExitCode {
				failed++
			}
			o.Index = exec.indexMap[o.Alias]
			for _, f := range exec.formatters {
				f.Add(*o)
			}
		case e := <-errc:
			if nil != e {
				err = e
				return
			}
		}
	}

}
