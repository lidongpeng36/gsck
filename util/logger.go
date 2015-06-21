package util

import (
	"bytes"
	"log"
	"os"
	"path"
)

// Log Level
const (
	TRACE = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

var loggerPrefix = []string{
	"[TRACE] ",
	"[DEBUG] ",
	"[INFO] ",
	"[WARN] ",
	"[ERROR] ",
	"[FATAL] ",
}

// Logger is a simple logger
type Logger struct {
	level int
	*log.Logger
}

// NewLogger returns a logger
// @filepath log's full path, mkdirp if dirname does not exists. Use stdout if is empty.
func NewLogger(filepath string) (logger *Logger) {
	var output *os.File
	output = os.Stdout
	if filepath != "" {
		dirname := path.Dir(filepath)
		if err := os.MkdirAll(dirname, 0755); err != nil {
			return
		}
		f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			output = f
		}
	}
	_logger := log.New(output, "", log.LstdFlags)
	logger = &Logger{DEBUG, _logger}
	return logger
}

// SetLevel is a level setter
func (logger *Logger) SetLevel(level int) {
	if level > 5 {
		level = 5
	}
	if level < 0 {
		level = 0
	}
	logger.level = level
}

func (logger *Logger) printLog(level int, format string, v ...interface{}) {
	if level < logger.level {
		return
	}
	var buffer bytes.Buffer
	_, _ = buffer.WriteString(loggerPrefix[level])
	_, _ = buffer.WriteString(format)
	_, _ = buffer.WriteString("\n")
	logger.Printf(buffer.String(), v...)
}

// Log : arbitrary log
func (logger *Logger) Log(prefix string, format string, v ...interface{}) {
	var buffer bytes.Buffer
	_, _ = buffer.WriteString("[" + prefix + "] ")
	_, _ = buffer.WriteString(format)
	_, _ = buffer.WriteString("\n")
	logger.Printf(buffer.String(), v...)
}

// Trace : logger.Trace
func (logger *Logger) Trace(format string, v ...interface{}) {
	logger.printLog(TRACE, format, v...)
}

// Debug : logger.Debug
func (logger *Logger) Debug(format string, v ...interface{}) {
	logger.printLog(DEBUG, format, v...)
}

// Info : logger.Info
func (logger *Logger) Info(format string, v ...interface{}) {
	logger.printLog(INFO, format, v...)
}

// Warn : logger.Warn
func (logger *Logger) Warn(format string, v ...interface{}) {
	logger.printLog(WARN, format, v...)
}

// Fatal : logger.Fatal
func (logger *Logger) Fatal(format string, v ...interface{}) {
	logger.printLog(FATAL, format, v...)
}
