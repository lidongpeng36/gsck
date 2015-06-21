package hostlist

import (
	"fmt"
	"sort"
	"strings"
)

var _list HostInfoList

// RegisterHostlist used in each realization's init function
func RegisterHostlist(builder constructor) {
	if constructorMap == nil {
		constructorMap = make(map[string]constructor)
	}
	name := builder("").Name()
	constructorMap[name] = builder
}

// HostInfo describes a host more precisely
type HostInfo struct {
	User  string
	Cmd   string
	Index int
	Host  string
	Alias string
}

// HostInfoList is updated version for Hostlist
type HostInfoList []*HostInfo

// MakeHostInfoListFromStringList helps migrate old implementation.
func MakeHostInfoListFromStringList(list []string) HostInfoList {
	hiList := make([]*HostInfo, len(list))
	for i, host := range list {
		hiList[i] = &HostInfo{
			Index: i,
			Host:  host,
			Alias: host,
		}
	}
	return hiList
}

// Hostlist should be able to give a hostname list by a single string.
type Hostlist interface {
	Name() string
	Priority() int // `0` has a higher Priority over `1`
	// Get() ([]string, error)
	Get() (HostInfoList, error)
	// if you're definitely sure that user want to use this Hostlist, return true.
	// for example, if user gives `-f ./xxx`, then obviously that he wants use this file: ./xxx.
	// in this case, no matter Get() method succeed or not, break.
	ShouldBreak() bool
}

// WithFilter has a Filter over Hostlist
// Implementation then has the ability to filter hosts
type WithFilter interface {
	Hostlist
	// Filter([]string) []string
	Filter(HostInfoList) HostInfoList
}

// SplitString is used for split Hostlist
const SplitString = "\\s+|;|,"

type constructor func(string) Hostlist

var constructorMap = make(map[string]constructor)
var preferHostlist string

type hostlistFinder struct {
	hash       map[string]Hostlist
	array      []Hostlist
	prefer     string
	realFinder string
}

func newHostlistFinder(str, prefer string) *hostlistFinder {
	count := len(constructorMap)
	finder := &hostlistFinder{
		hash:   make(map[string]Hostlist),
		array:  make([]Hostlist, 0, count),
		prefer: prefer,
	}
	for name, builder := range constructorMap {
		hl := builder(str)
		if hl != nil {
			finder.hash[name] = hl
			finder.array = append(finder.array, hl)
		}
	}
	sort.Sort(finder)
	return finder
}

func (finder *hostlistFinder) find() (list HostInfoList, err error) {
	err = fmt.Errorf("Cannot Get Host List!")
	if finder.prefer == "" {
		for _, hl := range finder.array {
			list, err = hl.Get()
			list = filter(list)
			if hlf, ok := hl.(WithFilter); ok {
				list = hlf.Filter(list)
			}
			if err == nil && len(list) == 0 {
				err = fmt.Errorf("List is empty.")
			}
			if err == nil {
				_list = list
				finder.realFinder = hl.Name()
				return
			}
			if hl.ShouldBreak() {
				break
			}
		}
		return
	}
	hl := finder.hash[finder.prefer]
	list, err = hl.Get()
	list = filter(list)
	if hlf, ok := hl.(WithFilter); ok {
		list = hlf.Filter(list)
	}
	if err == nil && len(list) == 0 {
		err = fmt.Errorf("List is empty.")
	}
	return
}

// Sort Interface

func (finder *hostlistFinder) Less(i, j int) bool {
	return finder.array[i].Priority() < finder.array[j].Priority()
}

func (finder *hostlistFinder) Len() int {
	return len(finder.array)
}

func (finder *hostlistFinder) Swap(i, j int) {
	finder.array[i], finder.array[j] = finder.array[j], finder.array[i]
}

// remove duplicated/begin with '#' hosts
func filter(in HostInfoList) HostInfoList {
	out := make(HostInfoList, 0, len(in))
	track := make(map[string]int)
	for _, x := range in {
		trackKey := x.Alias
		_, exists := track[trackKey]
		if trackKey != "" && !exists && !strings.HasPrefix(trackKey, "#") {
			out = append(out, x)
		}
		track[trackKey]++
	}
	return out
}

// Available returns all Hostlist's Name
func Available() []string {
	ret := make([]string, 0, len(constructorMap))
	for name := range constructorMap {
		ret = append(ret, name)
	}
	return ret
}

// GetHostList returns the final host list.
// It'll use cache if possible.
func GetHostList(str, prefer string) (list HostInfoList, err error) {
	if _list != nil && len(_list) > 0 {
		list = _list
		return
	}
	list, err = GetHostListNoCache(str, prefer)
	return
}

// GetHostListNoCache returns the final host list.
func GetHostListNoCache(str, prefer string) (list HostInfoList, err error) {
	if prefer != "" {
		if _, ok := constructorMap[prefer]; !ok {
			avail := strings.Join(Available(), ", ")
			err = fmt.Errorf("Use `%s` to get hostlist is not implemtented. \nAvailable: %s", prefer, avail)
			return
		}
	}
	finder := newHostlistFinder(str, prefer)
	list, err = finder.find()
	_list = list
	return
}
