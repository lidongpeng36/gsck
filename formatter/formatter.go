package formatter

import (
	"github.com/EvanLi/gsck/hostlist"
)

var info *Info

var _hostlist []string

func init() {
	info = new(Info)
}

// Output holds executor's output. And Formatter uses it for show.
type Output struct {
	Index    int    `json:"index"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error"`
	Hostname string `json:"hostname"`
	ExitCode int    `json:"exitcode"`
}

// Info is the shared infomation for all Formatters
// Formatters may want to change print style by Info
type Info struct {
	User        string
	Concurrency int64
}

// SetInfo sets package var: info
func SetInfo(user string, concurrency int64) {
	info.User = user
	info.Concurrency = concurrency
}

// SetHostlist sets global hostlist
func SetHostlist(list []string) {
	_hostlist = list
}

// Formatter formats Output for each host.
// Executor will send Output to Formatter with Add,
// and make Formatter Print when finished.
type Formatter interface {
	Add(*Output)
	Print()
}

type abstractFormatter struct {
	list  []string
	count int64
}

func newAbstractFormatter() *abstractFormatter {
	list := _hostlist
	if list == nil {
		list, _ = hostlist.GetHostList("")
	}
	count := int64(len(list))
	abf := &abstractFormatter{
		list:  list,
		count: count,
	}
	return abf
}

func (abf *abstractFormatter) hosts() []string {
	return abf.list
}

func (abf *abstractFormatter) length() int64 {
	return abf.count
}

// pragma mark - Formatter Interface

// Add do nothing
func (abf *abstractFormatter) Add(output *Output) {}

// Print do nothing
func (abf *abstractFormatter) Print() {}
