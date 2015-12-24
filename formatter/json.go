package formatter

import (
	"encoding/json"
	"fmt"
	// "gsck/config"
)

type jsonData struct {
	List    []Output `json:"list"`
	Summary struct {
		Success int64 `json:"success"`
		Failed  int64 `json:"failed"`
		Error   int64 `json:"error"`
	} `json:"summary"`
}

// JSONFormatter prints Outputs in JSON format.
type JSONFormatter struct {
	data *jsonData
}

// NewJSONFormatter is JSONFormatter's constructor
func NewJSONFormatter() *JSONFormatter {
	data := new(jsonData)
	data.List = make([]Output, 0, 10)
	jf := &JSONFormatter{
		data: data,
	}
	return jf
}

// pragma mark - Formatter Interface

// Add just collects all outputs, no prints.
func (jf *JSONFormatter) Add(output Output) {
	jf.data.List = append(jf.data.List, output)
	if "" != output.Error {
		jf.data.Summary.Error++
	} else if 0 != output.ExitCode {
		jf.data.Summary.Failed++
	} else {
		jf.data.Summary.Success++
	}
}

// Print prints all outputs that collected by Add
func (jf *JSONFormatter) Print() {
	var enc []byte
	var err error
	// if config.GetBool("json.pretty") {
	// 	enc, err = json.MarshalIndent(jf.data, "", "    ")
	// } else {
	// 	enc, err = json.Marshal(jf.data)
	// }
	enc, err = json.MarshalIndent(jf.data, "", "    ")
	if nil != err {
		fmt.Println(err)
		return
	}
	fmt.Println(string(enc))
}
