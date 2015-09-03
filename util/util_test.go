package util

import (
	"testing"
)

var beforeArray = []string{
	"",
	"pwd",
}

var cmdArray = []string{
	"",
	"hostname",
}

var afterArray = []string{
	"",
	"uptime",
}

var expectedArray = []string{
	"",
	"uptime",
	"hostname",
	"hostname && uptime",
	"pwd",
	"pwd && uptime",
	"pwd && hostname",
	"pwd && hostname && uptime",
}

func TestWrapCmd(t *testing.T) {
	for i, before := range beforeArray {
		for j, cmd := range cmdArray {
			for k, after := range afterArray {
				wrapped := WrapCmd(cmd, before, after)
				expected := expectedArray[i<<2+j<<1+k]
				if wrapped != expected {
					t.Fatalf("before:%s, cmd: %s, after: %s, Expectd: %s. Actual: %s", before, cmd, after, expected, wrapped)
				}
			}
		}
	}
}
