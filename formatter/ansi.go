package formatter

import (
	"fmt"
	"github.com/mgutz/ansi"
	"strconv"
)

var ansiMute bool

// MuteAnsi disables AnsiFormatter
func MuteAnsi() {
	ansiMute = true
}

// AnsiFormatter prints colored Outputs to stdout
type AnsiFormatter struct {
	reset        string
	rootHeader   string
	normalHeader string
	stdout       string
	stderr       string
	error        string
	index        int64
	count        int64
	digits       int
	digitFormat  string
	headerSpace  int
}

var fill = 72

// NewAnsiFormatter sets default colors.
func NewAnsiFormatter() *AnsiFormatter {
	reset := ansi.ColorCode("reset")
	count := int64(len(*HostList))
	digits := len(strconv.FormatInt(count, 10))
	formatStr := fmt.Sprintf("%%%dd / %%%dd : ", digits, digits)
	ansiFormatter := &AnsiFormatter{
		reset:        reset,
		rootHeader:   ansi.ColorCode("red+b"),
		normalHeader: ansi.ColorCode("yellow+b"),
		stdout:       ansi.ColorCode(""),
		stderr:       ansi.ColorCode("red"),
		error:        ansi.ColorCode("red+b:white"),
		count:        count,
		digits:       digits,
		digitFormat:  formatStr,
		headerSpace:  fill - len(fmt.Sprintf(formatStr, count, count)),
	}
	return ansiFormatter
}

func (af *AnsiFormatter) generateHeader(hostname string) (header string) {
	header += fmt.Sprintf(af.digitFormat, af.index, af.count)
	headerText := hostname
	if info.User != "" {
		headerText = info.User + "@" + headerText
	}
	symCount := (af.headerSpace - len(headerText) - 2) / 2
	if symCount <= 3 {
		symCount = 3
	}
	var left, right string
	for i := 0; i < symCount; i++ {
		left += "="
		right += "="
	}
	header += left + " " + headerText + " " + right
	for i := len(header); i < fill; i++ {
		header += "="
	}
	return
}

// pragma mark - Formatter Interface

// Add will print output to screen as AnsiFormatter is a real-time Formatter.
func (af *AnsiFormatter) Add(output Output) {
	af.index = af.index + 1
	header := af.generateHeader(output.Alias)
	headerFmt := af.normalHeader
	if info.User == "root" {
		headerFmt = af.rootHeader
	}
	fmt.Println(headerFmt, header, af.reset)
	if output.Stdout != "" {
		fmt.Printf("%s%s%s\n", af.stdout, output.Stdout, af.reset)
	}
	if output.Stderr != "" {
		fmt.Printf("%s%s%s\n", af.stderr, output.Stderr, af.reset)
	}
	if output.Error != "" {
		fmt.Printf("%s%s%s\n", af.error, output.Error, af.reset)
	}
}

// Print do nothing
func (af *AnsiFormatter) Print() {}
