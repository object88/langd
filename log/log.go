package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Level is the logging level: None, Error, Warn, Info, Verbose, or Debug
type Level int

const (
	// None means that the log should never write
	None Level = iota

	// Error means that only errors will be written
	Error

	// Warn means that errors and warnings will be written
	Warn

	// Info logging writes info, warning, and error
	Info

	// Verbose logs everything bug debug-level messages
	Verbose

	// Debug logs every message
	Debug

	stdOutLogname = "__stdout"
	stdErrLogname = "__stderr"
)

var m sync.RWMutex
var ls = map[string]*Log{}

// Log is a fairly basic logger
type Log struct {
	w   io.Writer
	lvl Level
}

// GetLog will return a log for the given name, creating
// one with the provided writer as needed
func GetLog(name string, w io.Writer) *Log {
	return getLog(name, w)
}

// Stderr gets the log for os.Stderr
func Stderr() *Log {
	return getLog(stdErrLogname, os.Stderr)
}

// Stdout gets the log for os.Stdout
func Stdout() *Log {
	return getLog(stdOutLogname, os.Stdout)
}

// Debugf will write if the log level is at least Debug.
// If the pointer receiver is nil, the log for `os.Stdout` will be used.
func (l *Log) Debugf(msg string, v ...interface{}) {
	if l == nil {
		l = Stdout()
	}

	l.writeIf(Debug, msg, v...)
}

// Errorf will write if the log level is at least Error.
// If the pointer receiver is nil, the log for `os.Stdout` will be used.
func (l *Log) Errorf(msg string, v ...interface{}) {
	if l == nil {
		l = Stdout()
	}

	l.writeIf(Error, msg, v...)
}

// Infof will write if the log level is at least Info.
// If the pointer receiver is nil, the log for `os.Stdout` will be used.
func (l *Log) Infof(msg string, v ...interface{}) {
	if l == nil {
		l = Stdout()
	}

	l.writeIf(Info, msg, v...)
}

// Printf will always log the given message, regardless of log level set.
// If the pointer receiver is nil, the log for `os.Stdout` will be used.
func (l *Log) Printf(msg string, v ...interface{}) {
	if l == nil {
		l = Stdout()
	}

	l.write(msg, v...)
}

// SetLevel will adjust the logger's level.  If the pointer receiver is nil,
// the log for `os.Stdout` will be used.
func (l *Log) SetLevel(lvl Level) {
	if l == nil {
		l = Stdout()
	}

	l.lvl = lvl
}

// Verbosef will write if the log level is at least Verbose.
// If the pointer receiver is nil, the log for `os.Stdout` will be used.
func (l *Log) Verbosef(msg string, v ...interface{}) {
	if l == nil {
		l = Stdout()
	}

	l.writeIf(Verbose, msg, v...)
}

// Warnf will write if the log level is at least Warn.
// If the pointer receiver is nil, the log for `os.Stdout` will be used.
func (l *Log) Warnf(msg string, v ...interface{}) {
	if l == nil {
		l = Stdout()
	}

	l.writeIf(Warn, msg, v...)
}

func getLog(name string, w io.Writer) *Log {
	m.RLock()

	if l, ok := ls[name]; ok {
		m.RUnlock()
		return l
	}

	m.RUnlock()

	m.Lock()

	if l, ok := ls[name]; ok {
		m.Unlock()
		return l
	}

	l := &Log{w, Error}
	ls[name] = l

	m.Unlock()
	return l
}

func (l *Log) write(msg string, v ...interface{}) {
	if v == nil {
		l.w.Write([]byte(msg))
	} else {
		m := fmt.Sprintf(msg, v...)
		l.w.Write([]byte(m))
	}
}

func (l *Log) writeIf(lvl Level, msg string, v ...interface{}) {
	if l.lvl < lvl {
		return
	}

	l.write(msg, v...)
}
