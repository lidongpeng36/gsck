package formatter

import (
	"fmt"
	ui "github.com/gizak/termui"
	tm "github.com/nsf/termbox-go"
	"os"
	"strconv"
	"time"
)

// WindowFormatter shows Outputs with termui
type WindowFormatter struct {
	widgets       map[string]ui.GridBufferer
	machineWidets []ui.Bufferer
	machineOutput []*Output
	machineCount  int
	headerYShift  int
	event         chan tm.Event
	list          []string
	progress      float64
	step          float64
	selectedIndex int
	pageSize      int
	selectedPage  int
	*abstractFormatter
}

// NewWindowFormatter initializes a WindowFormatter, and starts it.
func NewWindowFormatter() *WindowFormatter {
	bf := newAbstractFormatter()
	count := len(bf.hosts())
	wf := &WindowFormatter{
		widgets:       make(map[string]ui.GridBufferer),
		machineWidets: make([]ui.Bufferer, 0, count),
		machineOutput: make([]*Output, 0, count),
		machineCount:  count,
		event:         make(chan tm.Event),
		list:          make([]string, 0, count),
		progress:      0,
		step:          100 / float64(count),
		selectedIndex: 0,
		selectedPage:  0,
	}
	wf.abstractFormatter = bf
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	ui.UseTheme("helloworld")

	// Footer
	// help Box
	helpWidget := ui.NewPar("q:QUIT; j:DOWN; k:UP")
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
	wf.headerYShift = gauge.Height + 1
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
	// Machine List
	list := ui.NewList()
	list.Border.Label = "HOSTS"
	wf.widgets["machine"] = list
	// Machine Output
	outputWidget := ui.NewPar("0")
	outputWidget.Text = "INIT"
	outputWidget.Border.Label = "OUTPUT"
	wf.widgets["output"] = outputWidget
	// Hostlist
	digits := len(strconv.FormatInt(int64(count), 10))
	for index, host := range bf.hosts() {
		fmtStr := fmt.Sprintf("[%%%dd] %%s", digits)
		mPar := ui.NewPar(fmt.Sprintf(fmtStr, index+1, host))
		mPar.Height = 1
		mPar.Width = ui.TermWidth()/2 - 6
		mPar.HasBorder = false
		mPar.X = 5
		mPar.Y = 4 + index
		mPar.TextFgColor = ui.ColorBlue
		mPar.TextBgColor = ui.ColorWhite
		mPar.BgColor = ui.ColorYellow
		mPar.IsDisplay = false
		wf.machineWidets = append(wf.machineWidets, mPar)
		wf.machineOutput = append(wf.machineOutput, new(Output))
	}
	// Arrow
	arrow := ui.NewPar("->")
	arrow.Height = 1
	arrow.X = 2
	arrow.TextFgColor = ui.ColorCyan
	arrow.HasBorder = false
	wf.widgets["arrow"] = arrow
	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(8, 0, gauge),
			ui.NewCol(4, 0, infoWidget),
		),
		ui.NewRow(
			ui.NewCol(6, 0, list),
			ui.NewCol(6, 6, outputWidget),
		),
		ui.NewRow(
			ui.NewCol(12, 0, helpWidget),
		),
	)
	wf.selectItem(0)
	go func() {
		for {
			wf.event <- tm.PollEvent()
		}
	}()
	go wf.run()
	return wf
}

// run starts UI.
func (wf *WindowFormatter) run() {
	for {
		select {
		case e := <-wf.event:
			if e.Type == tm.EventResize {
				wf.updateMain()
				wf.refresh()
			} else if e.Type == tm.EventKey {
				switch e.Ch {
				case 'q':
					wf.terminate()
					os.Exit(2)
				case 'j':
					wf.selectedIndex = (wf.selectedIndex + 1) % len(wf.machineWidets)
					wf.selectItem(wf.selectedIndex)
				case 'k':
					machineCount := len(wf.machineWidets)
					if wf.selectedIndex == 0 {
						wf.selectedIndex = machineCount - 1
					} else {
						wf.selectedIndex = (wf.selectedIndex - 1) % machineCount
					}
					wf.selectItem(wf.selectedIndex)
				}
			}
		}
	}
}

func (wf *WindowFormatter) terminate() {
	ui.Close()
}

func (wf *WindowFormatter) refresh() {
	ui.Body.Width = ui.TermWidth()
	ui.Body.Align()
	bufferers := []ui.Bufferer{ui.Body}
	ms := wf.machineSlice()
	for index, machine := range ms {
		machine.(*ui.Par).Y = index + wf.headerYShift
	}
	bufferers = append(bufferers, ms...)
	arrow := wf.widgets["arrow"]
	bufferers = append(bufferers, arrow)
	ui.Render(bufferers...)
}

func (wf *WindowFormatter) mainHeightCalc() int {
	helpWidget := wf.widgets["help"]
	progress := wf.widgets["progress"]
	mainHeight := ui.TermHeight() - helpWidget.GetHeight() - progress.GetHeight()
	return mainHeight
}

func (wf *WindowFormatter) updateMain() {
	list := wf.widgets["machine"].(*ui.List)
	outputWidget := wf.widgets["output"].(*ui.Par)
	mainHeight := wf.mainHeightCalc()
	list.Height = mainHeight
	if mainHeight > 2 {
		wf.pageSize = mainHeight - 2
	} else {
		wf.pageSize = 0
	}
	width := ui.TermWidth()/2 - 6
	for _, m := range wf.machineWidets {
		m.(*ui.Par).Width = width
	}
	outputWidget.Height = mainHeight
}

func (wf *WindowFormatter) machineSlice() []ui.Bufferer {
	start := wf.pageSize * wf.selectedPage
	end := start + wf.pageSize
	if end >= wf.machineCount {
		end = wf.machineCount
	}
	return wf.machineWidets[start:end]
}

func (wf *WindowFormatter) selectItem(index int) {
	wf.updateMain()
	arrow := wf.widgets["arrow"].(*ui.Par)
	if wf.pageSize > 0 {
		residue := index % wf.pageSize
		arrow.Y = residue + wf.headerYShift
		wf.selectedPage = index / wf.pageSize
	}
	outputWidget := wf.widgets["output"].(*ui.Par)
	output := wf.machineOutput[index]
	if output.Stdout != "" {
		outputWidget.Text = output.Stdout
		outputWidget.TextFgColor = ui.ColorWhite
		outputWidget.TextBgColor = ui.ColorBlack
	} else if output.Stderr != "" {
		outputWidget.Text = output.Stderr
		outputWidget.TextFgColor = ui.ColorRed
		outputWidget.TextBgColor = ui.ColorWhite
	} else if output.Error != "" {
		outputWidget.Text = output.Error
		outputWidget.TextFgColor = ui.ColorRed
		outputWidget.TextBgColor = ui.ColorWhite
	} else {
		outputWidget.Text = ""
	}
	wf.refresh()
}

// pragma mark - Formatter Interface

// Add refreshes contents on screen
func (wf *WindowFormatter) Add(output *Output) {
	wf.progress += wf.step
	gauge := wf.widgets["progress"].(*ui.Gauge)
	progressInt := int(wf.progress)
	if progressInt > 100 {
		progressInt = 100
	}
	gauge.Percent = progressInt
	index := output.Index
	machineWidget := wf.machineWidets[index].(*ui.Par)
	if output.ExitCode == 0 {
		machineWidget.TextFgColor = ui.ColorGreen
		machineWidget.TextBgColor = ui.ColorBlack
	} else {
		machineWidget.TextFgColor = ui.ColorRed
		machineWidget.TextBgColor = ui.ColorBlack
	}
	wf.machineOutput[index] = output
	wf.selectItem(wf.selectedIndex)
}

// Print waits a while and then close UI
func (wf *WindowFormatter) Print() {
	time.Sleep(5 * time.Second)
	wf.terminate()
}
