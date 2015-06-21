package formatter

import (
	"github.com/EvanLi/gsck/hostlist"
)

var info *Info

var HostList *hostlist.HostInfoList

func init() {
	info = new(Info)
}

// Output holds executor's output. And Formatter uses it for show.
type Output struct {
	Index  int    `json:"index"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error"`
	// Hostname is the real host. Executor use this to connect.
	Hostname string `json:"hostname"`
	// Alias is the hostname that shown to user
	Alias    string `json:"alias"`
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

// SetHostInfoList sets global hostlist
func SetHostInfoList(list *hostlist.HostInfoList) {
	HostList = list
}

// Formatter formats Output for each host.
// Executor will send Output to Formatter with Add,
// and make Formatter Print when finished.
type Formatter interface {
	Add(Output)
	Print()
}

type abstractFormatter struct {
	aliasList []string
	hostList  []string
	hiList    *hostlist.HostInfoList
	count     int64
}

func newAbstractFormatter() *abstractFormatter {
	// hiList := _hostinfoList
	// if hiList == nil {
	// 	hiList, _ = hostlist.GetHostList("", "")
	// }
	count := int64(len(*HostList))
	abf := &abstractFormatter{
		hiList:    HostList,
		count:     count,
		aliasList: make([]string, count),
		hostList:  make([]string, count),
	}
	for i, hi := range *HostList {
		abf.aliasList[i] = hi.Alias
		abf.hostList[i] = hi.Host
	}
	return abf
}

func (abf *abstractFormatter) hosts() *hostlist.HostInfoList {
	return abf.hiList
}

func (abf *abstractFormatter) length() int64 {
	return abf.count
}

// pragma mark - Formatter Interface

// Add do nothing
func (abf *abstractFormatter) Add(output *Output) {}

// Print do nothing
func (abf *abstractFormatter) Print() {}
