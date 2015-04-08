package hostlist

import (
	"fmt"
	"sort"
	"strings"
)

// RegisterHostlist used in each realization's init function
func RegisterHostlist(builder constructor) {
	if constructorMap == nil {
		constructorMap = make(map[string]constructor)
	}
	name := builder("").Name()
	constructorMap[name] = builder
}

// Hostlist should be able to give a hostname list by a single string.
type Hostlist interface {
	Name() string
	Priority() int // `0` has a higher Priority over `1`
	Get() ([]string, error)
	// if you're definitely sure that user want to use this Hostlist, return true.
	// for example, if user gives `-f ./xxx`, then obviously that he wants use this file: ./xxx.
	// in this case, no matter Get() method succeed or not, break.
	ShouldBreak() bool
}

// WithFilter has a Filter over Hostlist
// Implementation then has the ability to filter hosts
type WithFilter interface {
	Hostlist
	Filter([]string) []string
}

// SplitString is used for split Hostlist
const SplitString = "\\s+|;|,"

type constructor func(string) Hostlist

var constructorMap = make(map[string]constructor)
var preferHostlist string

type hostlistFinder struct {
	hash       map[string]Hostlist
	array      []Hostlist
	realFinder string
}

func newHostlistFinder(str string) *hostlistFinder {
	count := len(constructorMap)
	finder := &hostlistFinder{
		hash:  make(map[string]Hostlist),
		array: make([]Hostlist, 0, count),
	}
	for name, builder := range constructorMap {
		hl := builder(str)
		finder.hash[name] = hl
		finder.array = append(finder.array, hl)
	}
	sort.Sort(finder)
	return finder
}

func (finder *hostlistFinder) find() (list []string, err error) {
	err = fmt.Errorf("Cannot Get Host List!")
	if preferHostlist == "" {
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
	} else {
		hl := finder.hash[preferHostlist]
		list, err = hl.Get()
		list = filter(list)
		if hlf, ok := hl.(WithFilter); ok {
			list = hlf.Filter(list)
		}
		if err == nil && len(list) == 0 {
			err = fmt.Errorf("List is empty.")
		}
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

var _list []string
var finder *hostlistFinder

func filter(in []string) []string {
	out := make([]string, 0, len(in))
	track := make(map[string]int)
	for _, x := range in {
		_, exists := track[x]
		if x != "" && !exists && !strings.HasPrefix(x, "#") {
			out = append(out, x)
		}
		track[x]++
	}
	return out
}

// SetPrefer sets preferred Hostlist
func SetPrefer(name string) (err error) {
	if name == "" {
		return
	}
	if _, ok := constructorMap[name]; ok {
		preferHostlist = name
	} else {
		avail := strings.Join(Available(), ", ")
		err = fmt.Errorf("Use `%s` to get hostlist is not implemtented. \nAvailable: %s", name, avail)
	}
	return
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
func GetHostList(str string) (list []string, err error) {
	if _list != nil && len(_list) > 0 {
		list = _list
		return
	}
	finder := newHostlistFinder(str)
	list, err = finder.find()
	return
}
