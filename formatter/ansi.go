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
	// *abstractFormatter
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
		headerSpace:  fill - len(fmt.Sprintf(formatStr, 0, 0)),
	}
	// ansiFormatter.abstractFormatter = newAbstractFormatter()
	return ansiFormatter
}

func (af *AnsiFormatter) generateHeader(hostname string) (header string) {
	// fill := 72
	// if len(hostname) > 72-len(af.digitFormat) {
	// 	fill = len(hostname) + 2
	// }
	// digits := len(strconv.FormatInt(af.length(), 10))
	// digits := len(strconv.FormatInt(int64(len(*HostList)), 10))
	// formatStr := fmt.Sprintf("%%%dd / %%%dd : ", af.digits, af.digits)
	// header += fmt.Sprintf(formatStr, af.index, af.count)
	header += fmt.Sprintf(af.digitFormat, af.index, af.count)
	// fillLeft := fill - len(header)
	fillLeft := af.headerSpace
	headerText := info.User + "@" + hostname
	if fillLeft+6 <= len(headerText) {
		fillLeft = len(headerText) + 6
	}
	symCount := (fillLeft - len(headerText) - 2) / 2
	var left, right string
	for i := 0; i < symCount; i++ {
		left += "="
		right += "="
	}
	header += left + " " + headerText + " " + right
	// if len(header) > fill-1 {
	// 	header = header[0 : fill-1]
	// }
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
