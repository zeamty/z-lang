package codegen

import (
	"strings"

	"github.com/zeamty/z-lang/pkg/parser"
)

func (g *Generator) emitAsmCall(name string, args []parser.Expr) string {
	template := ""
	var operands []asmOperand
	var inVals []string
	var inTypes []string

	if len(args) == 0 {
		return ""
	}
	if lit, ok := args[0].(*parser.BasicLit); ok {
		template = strings.Trim(lit.Value, "\"`")
	}

	for i := 1; i < len(args); i++ {
		if call, ok := args[i].(*parser.CallExpr); ok {
			sel, ok := call.Fun.(*parser.SelectorExpr)
			if !ok {
				continue
			}
			pkg, ok := sel.X.(*parser.Ident)
			if !ok || pkg.Name != "asm" {
				continue
			}

			switch sel.Sel.Name {
			case "In":
				op := g.parseAsmIn(call)
				operands = append(operands, op)
				val := g.emitExpr(op.varExpr)
				inVals = append(inVals, val)
				inTypes = append(inTypes, g.tr.exprType(op.varExpr).LLVMType())
			case "Out":
				op := g.parseAsmOut(call)
				operands = append(operands, op)
			case "InOut":
				op := g.parseAsmInOut(call)
				operands = append(operands, op)
			case "Clobbers":
				op := g.parseAsmClobbers(call)
				operands = append(operands, op)
			}
		}
	}

	constraints := ""
	clobbers := ""
	hasOutput := false
	outputType := "void"

	for _, op := range operands {
		switch op.kind {
		case asmIn:
			constraints += op.llvmConstraint() + ","
		case asmOut:
			hasOutput = true
			constraints += op.llvmConstraint() + ","
			outputType = g.tr.exprType(op.varExpr).LLVMType()
		case asmInOut:
			hasOutput = true
			constraints += op.llvmConstraint() + ","
		case asmClobber:
			clobbers += strings.Join(op.clobbers, ",") + ","
		}
	}
	constraints = strings.TrimSuffix(constraints, ",")
	clobbers = strings.TrimSuffix(clobbers, ",")

	llvmAsm := escapeAsmTemplate(template)

	fullConstraints := constraints
	if clobbers != "" {
		fullConstraints += ",~{" + clobbers + "}"
	}

	callArgs := ""
	for i, val := range inVals {
		if i > 0 {
			callArgs += ", "
		}
		callArgs += inTypes[i] + " " + val
	}

	if hasOutput {
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = call %s asm sideeffect \"%s\", \"%s\"(%s)",
			tmp, outputType, llvmAsm, fullConstraints, callArgs)
		return tmp
	}

	g.emitter.emitf("call void asm sideeffect \"%s\", \"%s\"(%s)",
		llvmAsm, fullConstraints, callArgs)
	return ""
}

type asmOperandKind int

const (
	asmIn asmOperandKind = iota
	asmOut
	asmInOut
	asmClobber
)

type asmOperand struct {
	kind       asmOperandKind
	constraint string
	varName    string
	varExpr    parser.Expr
	clobbers   []string
}

func (op asmOperand) llvmConstraint() string {
	switch op.kind {
	case asmIn:
		return llvmInConstraint(op.constraint)
	case asmOut:
		return llvmOutConstraint(op.constraint)
	case asmInOut:
		return llvmInOutConstraint(op.constraint)
	}
	return ""
}

var constraintMap = map[string]string{
	"a": "{ax}",
	"b": "{bx}",
	"c": "{cx}",
	"d": "{dx}",
	"S": "{si}",
	"D": "{di}",
	"r": "r",
	"m": "*m",
	"i": "i",
	"x": "{xmm0}",
}

func llvmInConstraint(c string) string {
	if mapped, ok := constraintMap[c]; ok {
		return mapped
	}
	return c
}

func llvmOutConstraint(c string) string {
	if len(c) > 1 && c[0] == '=' {
		inner := c[1:]
		if mapped, ok := constraintMap[inner]; ok {
			return "=" + mapped
		}
	}
	if len(c) > 1 && c[0] == '+' {
		inner := c[1:]
		if mapped, ok := constraintMap[inner]; ok {
			return "+" + mapped
		}
	}
	return c
}

func llvmInOutConstraint(c string) string {
	return llvmInConstraint(c)
}

func escapeAsmTemplate(tmpl string) string {
	tmpl = strings.ReplaceAll(tmpl, "$", "$$")
	tmpl = strings.ReplaceAll(tmpl, "\n", "\\0A")
	tmpl = strings.ReplaceAll(tmpl, "\t", "\\09")
	return tmpl
}

func (g *Generator) parseAsmIn(call *parser.CallExpr) asmOperand {
	var op asmOperand
	op.kind = asmIn
	if len(call.Args) >= 2 {
		if lit, ok := call.Args[0].(*parser.BasicLit); ok {
			op.constraint = strings.Trim(lit.Value, "\"")
		}
		op.varExpr = call.Args[1]
		if ident, ok := call.Args[1].(*parser.Ident); ok {
			op.varName = ident.Name
		}
	}
	return op
}

func (g *Generator) parseAsmOut(call *parser.CallExpr) asmOperand {
	var op asmOperand
	op.kind = asmOut
	if len(call.Args) >= 2 {
		if lit, ok := call.Args[0].(*parser.BasicLit); ok {
			op.constraint = strings.Trim(lit.Value, "\"")
		}
		op.varExpr = call.Args[1]
		if ident, ok := call.Args[1].(*parser.Ident); ok {
			op.varName = ident.Name
		}
	}
	return op
}

func (g *Generator) parseAsmInOut(call *parser.CallExpr) asmOperand {
	var op asmOperand
	op.kind = asmInOut
	if len(call.Args) >= 2 {
		if lit, ok := call.Args[0].(*parser.BasicLit); ok {
			op.constraint = strings.Trim(lit.Value, "\"")
		}
		op.varExpr = call.Args[1]
		if ident, ok := call.Args[1].(*parser.Ident); ok {
			op.varName = ident.Name
		}
	}
	return op
}

func (g *Generator) parseAsmClobbers(call *parser.CallExpr) asmOperand {
	var op asmOperand
	op.kind = asmClobber
	for _, arg := range call.Args {
		if lit, ok := arg.(*parser.BasicLit); ok {
			op.clobbers = append(op.clobbers, strings.Trim(lit.Value, "\""))
		}
	}
	return op
}