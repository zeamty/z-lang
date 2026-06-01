package codegen

import (
	"fmt"
	"strings"
)

// Emitter handles LLVM IR text generation
type Emitter struct {
	builder   strings.Builder
	labelCnt  int
	tmpCnt    int
	stringCnt int
	strings   map[string]int
	indent    string
}

func newEmitter() *Emitter {
	return &Emitter{
		strings: make(map[string]int),
		indent:  "  ",
	}
}

func (e *Emitter) emit(s string) {
	e.builder.WriteString(e.indent + s + "\n")
}

func (e *Emitter) emitf(format string, args ...interface{}) {
	e.builder.WriteString(fmt.Sprintf(e.indent+format+"\n", args...))
}

func (e *Emitter) emitRaw(s string) {
	e.builder.WriteString(s + "\n")
}

func (e *Emitter) emitRawf(format string, args ...interface{}) {
	e.builder.WriteString(fmt.Sprintf(format+"\n", args...))
}

func (e *Emitter) newLabel() string {
	e.labelCnt++
	return fmt.Sprintf("L%d", e.labelCnt)
}

func (e *Emitter) newTmp() string {
	e.tmpCnt++
	return fmt.Sprintf("%%t%d", e.tmpCnt)
}

func (e *Emitter) newString(name string) string {
	if id, ok := e.strings[name]; ok {
		return fmt.Sprintf("@.str.%d", id)
	}
	id := e.stringCnt
	e.strings[name] = id
	e.stringCnt++
	return fmt.Sprintf("@.str.%d", id)
}

func (e *Emitter) declareStrings() string {
	var b strings.Builder
	for s, id := range e.strings {
		escaped := strings.ReplaceAll(s, "\\", "\\5C")
		escaped = strings.ReplaceAll(escaped, "\"", "\\22")
		escaped = strings.ReplaceAll(escaped, "\n", "\\0A")
		b.WriteString(fmt.Sprintf("@.str.%d = private unnamed_addr constant [%d x i8] c\"%s\\00\"\n",
			id, len(s)+1, escaped))
	}
	return b.String()
}

func (e *Emitter) result() string {
	return e.builder.String()
}