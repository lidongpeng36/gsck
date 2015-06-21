package formatter

import (
	"fmt"
	"github.com/EvanLi/gsck/sig"
	"github.com/EvanLi/gsck/util"
	ui "github.com/gizak/termui"
	tm "github.com/nsf/termbox-go"
	"math"
	"os"
	"strconv"
	"strings"
	// "time"
)

// log to file
var debugLog = ""
var logger *util.Logger

func init() {
	if debugLog != "" {
		logger = util.NewLogger(debugLog)
		logger.Debug("Logger's Ready!")
	}
}

const usage = "C-c/q:QUIT; h/j/k/l/gg/G/C-u/C-d/n/N:VIM-LIKE; C-n/C-p:SCROLL OUTPUT;"

var windowFormatterExitCode = 2

const (
	up   = -1
	down = 1
)

type direction int

func adjustSection(start, end, total int) (s, e int) {
	if end < 0 || total < end {
		e = total
	} else {
		e = end
	}
	if start < 0 {
		s = 0
	} else {
		s = start
	}
	if s > e {
		s = e
	}
	return
}

type view interface {
	accessories() []ui.Bufferer
	beforeRender()
	block() *ui.Block
}

type listView interface {
	add(int, interface{})
	area(int, int) []string
	choose(index int)
	currentLine() int
}

type resizableView interface {
	resize()
	setNeedResize()
}

type masterView interface {
	addSlave(view)
}

type scrollView interface {
	view
	listView
	resizableView
	gotoLine(int)
	scroll(direction direction, step int)
	circularScroll(direction direction, step int)
}

type machineOutput struct {
	output Output
	text   string
	height int
	width  int
	start  int
	end    int
	margin int
	lines  []string
}

func (mo *machineOutput) area(start, end int) []string {
	start, end = adjustSection(start, end, len(mo.lines))
	return mo.lines[start:end]
}

func (mo *machineOutput) currentLine() int {
	return mo.start
}

func (mo *machineOutput) visible() []string {
	return mo.area(mo.start, mo.end)
}

func (mo *machineOutput) gotoLine(no int) {
	start := no
	mo.end = len(mo.lines)
	if start < 0 {
		start = 0
	}
	if start >= mo.end {
		start = mo.end - 1
	}
	mo.start = start
}

func (mo *machineOutput) scroll(direction direction, step int) {
	start := int(direction)*step + mo.start
	mo.gotoLine(start)
}

func (mo *machineOutput) circularScroll(direction direction, step int) {
	mo.scroll(direction, step)
}

func (mo *machineOutput) resize(height, width int) {
	if width <= mo.margin || height <= mo.margin {
		return
	}
	mo.width = width
	mo.lines = util.JustifyText(mo.text, width-mo.margin)
	length := len(mo.lines)
	height -= mo.margin
	mo.height = mo.height
	mo.end = length
	if mo.start > mo.end {
		mo.start = mo.end - 1
	}
}

func (mo *machineOutput) add(output Output) {
	mo.output = output
	mo.text = output.Stdout + output.Stderr + output.Error
}

// outputUI implements scrollView interface
type outputUI struct {
	outputs    []*machineOutput
	needResize bool
	*machineOutput
	*ui.List
}

func newOutputUI(hosts []string) *outputUI {
	oui := &outputUI{
		outputs:    make([]*machineOutput, 0, len(hosts)),
		needResize: true,
		List:       ui.NewList(),
	}
	oui.Border.Label = "OUTPUT"
	for i := 0; i < len(hosts); i++ {
		mo := &machineOutput{
			margin: 2,
			output: *new(Output),
		}
		oui.outputs = append(oui.outputs, mo)
	}
	oui.machineOutput = oui.outputs[0]
	oui.setNeedResize()
	return oui
}

func (oui *outputUI) block() *ui.Block {
	return &oui.List.Block
}

func (oui *outputUI) add(index int, output interface{}) {
	oui.outputs[index].add(output.(Output))
}

func (oui *outputUI) choose(index int) {
	oui.machineOutput = oui.outputs[index]
	if len(oui.visible()) == 0 {
		oui.setNeedResize()
		oui.resize()
	}
}

// resizableView interface
func (oui *outputUI) accessories() []ui.Bufferer {
	return []ui.Bufferer{}
}

func (oui *outputUI) beforeRender() {
	oui.Items = oui.visible()
}

func (oui *outputUI) resize() {
	if oui.needResize {
		oui.needResize = false
		oui.machineOutput.resize(oui.Height, oui.Width)
	}
}

func (oui *outputUI) setNeedResize() {
	oui.needResize = true
}

const (
	success = 1
	failed  = -1
	ready   = 0
)

type machineStatus int

// hostlistUI implements scrollView interface
type hostlistUI struct {
	list          []ui.Bufferer
	lines         []string
	status        []machineStatus
	visible       []ui.Bufferer
	count         int
	margin        int
	shiftX        int
	shiftY        int
	selectedIndex int
	selectedPage  int
	pageSize      int
	needResize    bool
	arrow         ui.Bufferer
	slaves        []scrollView
	// container
	*ui.Par
}

func newHostlistUI(hosts []string, shiftX, shiftY int) *hostlistUI {
	count := len(hosts)
	hui := &hostlistUI{
		list:       make([]ui.Bufferer, 0, count),
		lines:      make([]string, 0, count),
		status:     make([]machineStatus, count, count),
		visible:    []ui.Bufferer{},
		count:      count,
		shiftX:     shiftX,
		shiftY:     shiftY,
		slaves:     []scrollView{},
		needResize: true,
	}
	arrow := ui.NewPar("->")
	arrow.Height = 1
	arrow.X = shiftX + 2
	arrow.TextFgColor = ui.ColorCyan
	arrow.HasBorder = false

	hui.arrow = arrow
	container := ui.NewPar("")
	container.Border.Label = "HOSTS"
	hui.Par = container

	digits := len(strconv.FormatInt(int64(count), 10))
	fmtStr := fmt.Sprintf("[%%%dd] %%s", digits)
	for index, host := range hosts {
		text := fmt.Sprintf(fmtStr, index+1, host)
		mPar := ui.NewPar(text)
		mPar.Height = 1
		mPar.Width = ui.TermWidth()/2 - 6
		mPar.HasBorder = false
		mPar.X = shiftX + 5
		mPar.Y = shiftY + index
		hui.list = append(hui.list, mPar)
		hui.lines = append(hui.lines, text)
	}
	return hui
}

func (hui *hostlistUI) fillVisibleList() {
	start := hui.pageSize * hui.selectedPage
	end := hui.pageSize
	if end > hui.count {
		end = hui.count
	}
	hui.visible = hui.list[0:end]
	for index, machine := range hui.visible {
		no := start + index
		if no < hui.count {
			machineWidget := machine.(*ui.Par)
			machineWidget.Text = hui.lines[no]
			switch hui.status[no] {
			case success:
				machineWidget.TextFgColor = ui.ColorGreen
				machineWidget.TextBgColor = ui.ColorBlack
			case failed:
				machineWidget.TextFgColor = ui.ColorRed
				machineWidget.TextBgColor = ui.ColorBlack
			default:
				machineWidget.TextFgColor = ui.ColorBlue
				machineWidget.TextBgColor = ui.ColorWhite
			}
		} else {
			machine.(*ui.Par).Text = ""
		}
	}
}

func (hui *hostlistUI) block() *ui.Block {
	return &hui.Par.Block
}

// masterView interface

func (hui *hostlistUI) addSlave(slave scrollView) {
	hui.slaves = append(hui.slaves, slave)
}

// scrollView interface

func (hui *hostlistUI) accessories() (list []ui.Bufferer) {
	list = append([]ui.Bufferer(nil), hui.visible...)
	list = append(list, hui.arrow)
	return
}

func (hui *hostlistUI) gotoLine(no int) {
	newIndex := no
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= hui.count {
		newIndex = hui.count - 1
	}
	hui.selectedIndex = newIndex
	hui.choose(newIndex)
}

func (hui *hostlistUI) scroll(direction direction, step int) {
	newIndex := hui.selectedIndex + step*int(direction)
	hui.gotoLine(newIndex)
}

func (hui *hostlistUI) circularScroll(direction direction, step int) {
	newIndex := hui.selectedIndex + step*int(direction)
	hui.selectedIndex = (hui.count + newIndex) % hui.count
	hui.choose(hui.selectedIndex)
}

// resizableView interface
func (hui *hostlistUI) resize() {
	if hui.needResize {
		hui.needResize = false
		if hui.Height > 2 {
			hui.pageSize = hui.Height - 2
		} else {
			hui.pageSize = 0
		}
		width := hui.Width - 6
		if width < 0 {
			width = 0
		}
		for _, m := range hui.visible {
			m.(*ui.Par).Width = width
		}
	}
}

func (hui *hostlistUI) setNeedResize() {
	hui.needResize = true
}

func (hui *hostlistUI) beforeRender() {
	if hui.pageSize == 0 {
		hui.setNeedResize()
		hui.resize()
	} else {
		residue := hui.selectedIndex % hui.pageSize
		hui.arrow.(*ui.Par).Y = residue + hui.shiftY
		hui.selectedPage = hui.selectedIndex / hui.pageSize
	}
	hui.fillVisibleList()
	hui.choose(hui.selectedIndex)
}

// listView interface

func (hui *hostlistUI) add(index int, data interface{}) {
	output := data.(Output)
	for _, slave := range hui.slaves {
		slave.add(index, output)
	}
	if output.ExitCode == 0 {
		hui.status[index] = success
	} else {
		hui.status[index] = failed
	}
}

func (hui *hostlistUI) choose(index int) {
	for _, slave := range hui.slaves {
		slave.choose(hui.selectedIndex)
	}
}

func (hui *hostlistUI) area(start, end int) []string {
	start, end = adjustSection(start, end, len(hui.lines))
	return hui.lines[start:end]
}

func (hui *hostlistUI) currentLine() int {
	return hui.selectedIndex
}

// WindowFormatter shows Outputs with termui
type WindowFormatter struct {
	widgets       map[string]ui.GridBufferer
	machineCount  int
	hostlistView  *hostlistUI
	outputView    *outputUI
	mainViews     []scrollView
	mainHeight    int
	focus         int
	event         chan tm.Event
	progress      float64
	step          float64
	searchKeyword string
	refreshChan   chan bool
	needResize    bool
	*abstractFormatter
}

// NewWindowFormatter initializes a WindowFormatter, and starts it.
func NewWindowFormatter() *WindowFormatter {
	bf := newAbstractFormatter()
	count := int(bf.length())

	wf := &WindowFormatter{
		widgets:      make(map[string]ui.GridBufferer),
		machineCount: count,
		event:        make(chan tm.Event),
		step:         100 / float64(count),
		refreshChan:  make(chan bool, 1),
		needResize:   true,
	}
	wf.abstractFormatter = bf
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	ui.UseTheme("helloworld")

	// Footer
	// help Box
	helpWidget := ui.NewPar(usage)
	helpWidget.Height = 3
	helpWidget.Border.Label = "HELP"
	helpWidget.Border.FgColor = ui.ColorCyan
	helpWidget.TextFgColor = ui.ColorWhite
	wf.widgets["help"] = helpWidget
	// Header
	// progess Gauge
	headerHeight := 3
	gauge := ui.NewGauge()
	gauge.Height = headerHeight
	gauge.Percent = 0
	gauge.Border.Label = "PROGRESS"
	gauge.Border.FgColor = ui.ColorCyan
	gauge.BarColor = ui.ColorGreen
	wf.widgets["progress"] = gauge
	// info Textbox
	infoText := fmt.Sprintf("USER : %s", info.User)
	infoWidget := ui.NewPar(infoText)
	infoWidget.Height = headerHeight
	if info.User == "root" {
		infoWidget.Border.FgColor = ui.ColorRed
		infoWidget.TextFgColor = ui.ColorRed
		infoWidget.Border.LabelFgColor = ui.ColorRed
	} else {
		infoWidget.Border.FgColor = ui.ColorCyan
		infoWidget.TextFgColor = ui.ColorGreen
		infoWidget.TextBgColor = ui.ColorBlack
	}
	infoWidget.Border.Label = "INFO"
	wf.widgets["info"] = infoWidget
	// Main
	wf.outputView = newOutputUI(bf.aliasList)
	wf.hostlistView = newHostlistUI(bf.aliasList, 0, helpWidget.Height+1)
	wf.hostlistView.addSlave(wf.outputView)
	wf.mainViews = []scrollView{wf.hostlistView, wf.outputView}

	ui.Body.AddRows(
		// header
		ui.NewRow(
			ui.NewCol(8, 0, gauge),
			ui.NewCol(4, 0, infoWidget),
		),
		// main
		ui.NewRow(
			ui.NewCol(6, 0, wf.hostlistView),
			ui.NewCol(6, 0, wf.outputView),
		),
		// footer
		ui.NewRow(
			ui.NewCol(12, 0, helpWidget),
		),
	)
	go wf.run()
	return wf
}

func (wf *WindowFormatter) debug(text string) {
	infoWidget := wf.widgets["info"].(*ui.Par)
	infoWidget.Text = text
}

// run starts UI.
func (wf *WindowFormatter) run() {
	helpWidget := wf.widgets["help"].(*ui.Par)
	// return `true` indicates that this leaderKey wants more args
	type leaderHandler func(tm.Event, string) bool
	leaderHandlers := map[string]leaderHandler{
		"/": func(e tm.Event, args string) bool {
			if e.Key == tm.KeyEnter {
				wf.searchKeyword = args
				wf.searchForward(0)
				return false
			}
			return true
		},
		":": func(e tm.Event, args string) bool {
			if e.Key == tm.KeyEnter {
				if args == "q" || args == "Q" {
					wf.quit()
				}
				focusView := wf.mainViews[wf.focus]
				num, err := strconv.ParseInt(args, 10, 32)
				if err == nil {
					focusView.gotoLine(int(num) - 1)
				}
				return false
			}
			return true
		},
		"g": func(e tm.Event, args string) bool {
			switch e.Ch {
			case 'g':
				focusView := wf.mainViews[wf.focus]
				focusView.scroll(up, math.MaxInt32)
			}
			return false
		},
	}
	numberLeader := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	numberLeaderHandler := func(leader string) leaderHandler {
		return func(e tm.Event, args string) bool {
			if '0' <= e.Ch && e.Ch <= '9' {
				return true
			}
			number, err := strconv.ParseInt(leader+args, 10, 32)
			if err != nil || number > math.MaxInt32 {
				return false
			}
			wf.kbdHandler(e, int(number))
			return false
		}
	}
	for _, nl := range numberLeader {
		leaderHandlers[nl] = numberLeaderHandler(nl)
	}
	leaderKey := ""
	args := ""
	resetPrefix := func() {
		leaderKey = ""
		args = ""
		helpWidget.Border.Label = "HELP"
		helpWidget.Text = usage
	}
	// WindowFormatter will only be used by command-line tools (such as gsck, opposite to long-stay processes),
	// so we can happily ignore these infinite goroutine.
	go func() {
		for {
			wf.event <- tm.PollEvent()
		}
	}()
	wf.setNeedRefresh()
	for {
		select {
		case <-wf.refreshChan:
			wf.refresh()
		case e := <-wf.event:
			if e.Type == tm.EventResize {
				wf.needResize = true
			} else if e.Type == tm.EventKey {
				input := string(e.Ch)
				if handler, ok := leaderHandlers[leaderKey]; ok {
					// Has leader
					if e.Key == tm.KeyBackspace || e.Key == tm.KeyBackspace2 {
						// backspace pressed
						argsLen := len(args)
						if argsLen >= 1 {
							args = args[:argsLen-1]
						}
						helpWidget.Text = leaderKey + args
					} else if e.Key != tm.KeyEsc && handler(e, args) {
						// invoke leader's handler
						args += input
						helpWidget.Text = leaderKey + args
					} else {
						// esc pressed or handler() already processed the args.
						resetPrefix()
					}
				} else if _, ok := leaderHandlers[input]; ok {
					// No leader, but input is leader
					leaderKey = input
					helpWidget.Border.Label = "Cancel with `Esc`"
					helpWidget.Text = leaderKey
				} else {
					// No leader, and input is not leader
					wf.kbdHandler(e, 1)
				}
			}
			wf.setNeedRefresh()
		}
	}
}

func (wf *WindowFormatter) search(direction direction, nth int) {
	if wf.searchKeyword == "" {
		return
	}
	focusView := wf.mainViews[wf.focus]
	currentLine := focusView.currentLine()
	var lines []string
	var start, end, origin int
	step := int(direction)
	if direction == up {
		lines = focusView.area(0, currentLine)
		start = len(lines) - 1
		end = 0
		origin = 0
	} else {
		lines = focusView.area(currentLine, -1)
		start = 0
		end = len(lines) - 1
		origin = currentLine
	}
	if start < 0 || end < 0 {
		return
	}
	skipped := 0
	lastFind := -1
	end += step
	for i := start; i != end; i += step {
		line := lines[i]
		if strings.Contains(line, wf.searchKeyword) {
			lastFind = origin + i
			if skipped >= nth {
				focusView.gotoLine(lastFind)
				return
			}
			skipped++
		}
	}
	if lastFind > 0 {
		focusView.gotoLine(lastFind)
	}
}

func (wf *WindowFormatter) searchForward(nth int) {
	wf.search(down, nth)
}

func (wf *WindowFormatter) searchBackward(nth int) {
	wf.search(up, nth)
}

func (wf *WindowFormatter) quit() {
	wf.terminate()
	gauge := wf.widgets["progress"].(*ui.Gauge)
	if gauge.Percent != 100 {
		windowFormatterExitCode = -1
		sig.CleanUp()
	}
	os.Exit(windowFormatterExitCode)
}

func (wf *WindowFormatter) kbdHandler(e tm.Event, repeat int) {
	if repeat <= 0 || wf.mainHeight < 2 {
		return
	}
	mainViewCount := len(wf.mainViews)
	if e.Type == tm.EventKey {
		focusView := wf.mainViews[wf.focus]
		switch e.Key {
		case tm.KeyCtrlC:
			wf.quit()
		case tm.KeyCtrlU:
			focusView.scroll(up, wf.mainHeight*repeat)
		case tm.KeyCtrlD:
			focusView.scroll(down, wf.mainHeight*repeat)
		case tm.KeyCtrlN:
			wf.outputView.scroll(down, repeat)
		case tm.KeyCtrlP:
			wf.outputView.scroll(up, repeat)
		default:
			switch e.Ch {
			case 'q':
				wf.quit()
			case 'h':
				if wf.focus-1 < 0 {
					wf.focus = 0
				} else {
					wf.focus--
				}
			case 'j':
				focusView.circularScroll(down, repeat)
			case 'k':
				focusView.circularScroll(up, repeat)
			case 'l':
				if wf.focus+1 >= mainViewCount {
					wf.focus = mainViewCount - 1
				} else {
					wf.focus++
				}
			case 'n':
				wf.searchForward(repeat)
			case 'N':
				// skip current line
				wf.searchBackward(repeat - 1)
			case 'G':
				focusView.scroll(down, math.MaxInt32)
			}
		}
	}
}

func (wf *WindowFormatter) terminate() {
	ui.Close()
}

func (wf *WindowFormatter) setNeedRefresh() {
	go func() {
		wf.refreshChan <- true
	}()
}

func (wf *WindowFormatter) refresh() {
	if wf.needResize {
		wf.needResize = false
		wf.updateMain()
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		for _, v := range wf.mainViews {
			v.setNeedResize()
		}
	}
	bufferers := []ui.Bufferer{ui.Body}
	for i, v := range wf.mainViews {
		view := v.block()
		if i == wf.focus {
			view.Border.FgColor = ui.ColorRed
		} else {
			view.Border.FgColor = ui.ColorWhite
		}
		v.resize()
		v.beforeRender()
		bufferers = append(bufferers, v.accessories()...)
	}
	ui.Render(bufferers...)
}

func (wf *WindowFormatter) updateMain() bool {
	helpWidget := wf.widgets["help"]
	progress := wf.widgets["progress"]
	mainHeight := ui.TermHeight() - helpWidget.GetHeight() - progress.GetHeight()

	if wf.mainHeight != mainHeight {
		wf.mainHeight = mainHeight
		for _, v := range wf.mainViews {
			v.block().Height = mainHeight
		}
		return true
	}
	return false
}

// pragma mark - Formatter Interface

// Add binds Output to machine, and refreshes contents on screen
func (wf *WindowFormatter) Add(output Output) {
	wf.progress += wf.step
	gauge := wf.widgets["progress"].(*ui.Gauge)
	progressInt := int(wf.progress)
	if progressInt > 100 {
		progressInt = 100
	}
	gauge.Percent = progressInt
	index := output.Index

	wf.hostlistView.add(index, output)
	wf.setNeedRefresh()
}

// Print waits util q/C-c.
func (wf *WindowFormatter) Print() {
	if windowFormatterExitCode < 0 {
		return
	}
	windowFormatterExitCode = 0
	gauge := wf.widgets["progress"].(*ui.Gauge)
	gauge.Percent = 100
	wf.setNeedRefresh()
	if gauge.Percent > 0 {
		<-make(chan bool, 1)
	}
	wf.terminate()
}
