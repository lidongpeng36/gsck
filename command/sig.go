package command

import (
	"container/heap"
	"fmt"
	"os"
	"os/signal"
)

// Global switcher
var enable = false

// SignalHandler is handler for SIGINT and SIGKILL
type SignalHandler func() error

// RegisterSignalHandler will add @handler to signal callback list
//   @name: mark for the handler
//   @handler: callback, returns error.
//   @priority: all Handlers will run in priority order.
//     - `0` has a higher Priority over `1`.
func RegisterSignalHandler(name string, handler SignalHandler, priority int) {
	if !enable {
		return
	}
	sh := &sigHandler{
		handler:  handler,
		name:     name,
		enabled:  true,
		priority: priority,
	}
	shpq.Push(sh)
	heap.Fix(shpq, sh.index)
}

// DisableSignalHandler can Disable SignalHandler by name
func DisableSignalHandler(name string) {
	shpq.Disable(name)
}

var signalc = make(chan os.Signal, 1)

// RunSignal will subscribe to SIGINT and SIGKILL and do CleanUp if triggered
func RunSignal() {
	enable = true
	signal.Notify(signalc, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-signalc:
			CleanUpSignals()
		}
	}()
}

// CleanUpSignals runs all enabled handlers in priority order
func CleanUpSignals() {
	if !enable {
		return
	}
	signal.Stop(signalc)
	rc := 0
	if err := shpq.Run(); nil != err {
		rc = 2
	}
	os.Exit(rc)
}

func init() {
	shpq = &signalHandlerPQ{
		queue: make([]*sigHandler, 0),
		hash:  make(map[string]*sigHandler),
	}
	heap.Init(shpq)
}

type sigHandler struct {
	handler  SignalHandler
	name     string
	enabled  bool
	priority int
	index    int // Needed by heap
}

var shpq *signalHandlerPQ

// SignalHandler Priority Queue
type signalHandlerPQ struct {
	queue []*sigHandler
	hash  map[string]*sigHandler
}

func (pq *signalHandlerPQ) Run() (err error) {
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*sigHandler)
		if !item.enabled {
			continue
		}
		if err = item.handler(); nil != err {
			fmt.Printf("Signal Handler %s Failed: %s\n", item.name, err.Error())
			return
		}
	}
	return
}

func (pq signalHandlerPQ) Disable(name string) {
	if item, ok := pq.hash[name]; ok {
		item.enabled = false
	}
}

// pragma mark - heap.Interface interface
// - pragma mark - sort.Interface interface

func (pq signalHandlerPQ) Len() int { return len(pq.queue) }
func (pq signalHandlerPQ) Less(i, j int) bool {
	return pq.queue[i].priority < pq.queue[j].priority
}
func (pq signalHandlerPQ) Swap(i, j int) {
	pq.queue[i], pq.queue[j] = pq.queue[j], pq.queue[i]
	pq.queue[i].index, pq.queue[j].index = j, i
}

// - end of sort.Interface interface

func (pq *signalHandlerPQ) Push(x interface{}) {
	item := x.(*sigHandler)
	if _, ok := pq.hash[item.name]; ok {
		return
	}
	n := len(pq.queue)
	item.index = n
	pq.queue = append(pq.queue, item)
	pq.hash[item.name] = item
}
func (pq *signalHandlerPQ) Pop() interface{} {
	old := pq.queue
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	pq.queue = old[0 : n-1]
	delete(pq.hash, item.name)
	return item
}

// end of head.Interface interface
