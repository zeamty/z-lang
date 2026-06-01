package sema

import (
	"fmt"
	"strings"
)

type ErrorLevel int

const (
	Error ErrorLevel = iota
	Warning
)

type Diagnostic struct {
	Level   ErrorLevel
	Message string
	Line    int
	Column  int
}

func (d Diagnostic) String() string {
	level := "error"
	if d.Level == Warning {
		level = "warning"
	}
	return fmt.Sprintf("line %d: %s: %s", d.Line, level, d.Message)
}

type ErrorCollector struct {
	errors   []Diagnostic
	hasError bool
}

func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{}
}

func (ec *ErrorCollector) Add(level ErrorLevel, line, col int, msg string) {
	ec.errors = append(ec.errors, Diagnostic{
		Level:   level,
		Message: msg,
		Line:    line,
		Column:  col,
	})
	if level == Error {
		ec.hasError = true
	}
}

func (ec *ErrorCollector) Errorf(line, col int, format string, args ...interface{}) {
	ec.Add(Error, line, col, fmt.Sprintf(format, args...))
}

func (ec *ErrorCollector) HasErrors() bool {
	return ec.hasError
}

func (ec *ErrorCollector) Errors() []string {
	var msgs []string
	for _, e := range ec.errors {
		msgs = append(msgs, e.String())
	}
	return msgs
}

func (ec *ErrorCollector) String() string {
	return strings.Join(ec.Errors(), "\n")
}