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
	*abstractFormatter
}

// NewAnsiFormatter sets default colors.
func NewAnsiFormatter() *AnsiFormatter {
	reset := ansi.ColorCode("reset")
	ansiFormatter := &AnsiFormatter{
		reset:        reset,
		rootHeader:   ansi.ColorCode("red+b"),
		normalHeader: ansi.ColorCode("yellow+b"),
		stdout:       ansi.ColorCode(""),
		stderr:       ansi.ColorCode("red"),
		error:        ansi.ColorCode("red+b:white"),
	}
	ansiFormatter.abstractFormatter = newAbstractFormatter()
	return ansiFormatter
}

func (af *AnsiFormatter) generateHeader(hostname string) (header string) {
	fill := 60
	digits := len(strconv.FormatInt(af.length(), 10))
	formatStr := fmt.Sprintf("%%%dd / %%%dd : ", digits, digits)
	header += fmt.Sprintf(formatStr, af.index, af.count)
	fill -= len(header)
	symCount := (fill - len(hostname) - 2) / 2
	var left, right string
	for i := 0; i < symCount; i++ {
		left += "="
		right += "="
	}
	header += left + " " + hostname + " " + right
	if len(header) > 59 {
		header = header[0:59]
	}
	return
}

// pragma mark - Formatter Interface

// Add will print output to screen as AnsiFormatter is a real-time Formatter.
func (af *AnsiFormatter) Add(output *Output) {
	af.index = af.index + 1
	header := af.generateHeader(output.Hostname)
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
