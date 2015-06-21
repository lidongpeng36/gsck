package sig

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	enable = true
	code := m.Run()
	os.Exit(code)
}

func funcGen(buf *bytes.Buffer, name string, err error) func() error {
	return func() error {
		buf.WriteString(fmt.Sprintf("I'm function %s\n", name))
		return err
	}
}

func TestPriorityQueue(t *testing.T) {
	buf := new(bytes.Buffer)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("%d", i)
		f := funcGen(buf, name, nil)
		RegisterSignalHandler(name, f, -i)
	}
	// fmt.Println(buf.String())
	expected := `I'm function 2
I'm function 1
I'm function 0
`
	shpq.Run()
	if buf.String() != expected {
		t.Fatalf("Expectd: %s. Actual: %s", expected, buf.String())
	}
}

func TestPriorityQueueError(t *testing.T) {
	buf := new(bytes.Buffer)
	f1 := funcGen(buf, "error", fmt.Errorf("ErrorByDesign"))
	f2 := funcGen(buf, "normal", nil)
	RegisterSignalHandler("error", f1, 0)
	RegisterSignalHandler("normal", f2, 1)
	expected := `Signal Handler error Failed: ErrorByDesign
I'm function error`
	shpq.Run()
	if buf.String() != expected {
		t.Fatalf("Expectd: %s. Actual: %s", expected, buf.String())
	}
}
