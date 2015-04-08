package hostlist

import (
	"os/exec"
	"strings"
)

// FromCmd is an abstract struct
// Other Hostlist that get hostlist from Command may want to inherit from it.
type FromCmd struct {
	Args          []string
	LineProcessor func(string) []string
}

// Get is part of Hostlist Interface.
// It executes cmd from @Args, and translate each line to hostname by @LineProcessor
func (hc *FromCmd) Get() (list []string, err error) {
	var out []byte
	args := hc.Args[1:]
	out, err = exec.Command(hc.Args[0], args...).Output()
	if err != nil {
		return
	}
	lines := strings.Split(string(out), "\n")
	if hc.LineProcessor != nil {
		list = make([]string, 0, len(lines))
		for _, line := range lines {
			if line != "" {
				for _, host := range hc.LineProcessor(line) {
					list = append(list, host)
				}
			}
		}
	} else {
		list = lines
	}
	return
}

// ShouldBreak makes default disicion: fallthrough
func (hc *FromCmd) ShouldBreak() bool {
	return false
}
