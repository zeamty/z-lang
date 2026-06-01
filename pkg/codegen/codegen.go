package codegen

import (
	"fmt"
	"strings"

	"github.com/zeamty/z-lang/pkg/lexer"
	"github.com/zeamty/z-lang/pkg/parser"
	"github.com/zeamty/z-lang/pkg/types"
)

// Generator generates LLVM IR from AST
type Generator struct {
	output    strings.Builder
	functions map[string]*parser.FuncDecl
	structs   map[string]*types.Struct
	vars      map[string]*parser.VarDecl
	strings   map[string]int
	stringCnt int
	tmpCnt    int
	labelCnt  int
	tr        *TypeResolver
}

func New() *Generator {
	g := &Generator{
		functions: make(map[string]*parser.FuncDecl),
		structs:   make(map[string]*types.Struct),
		vars:      make(map[string]*parser.VarDecl),
		strings:   make(map[string]int),
	}
	g.tr = newTypeResolver(g.vars, g.functions)
	return g
}

func (g *Generator) Generate(file *parser.File) string {
	g.output.Reset()

	g.emit("; Z Language Compiled Output")
	g.emit("; LLVM IR")
	g.emit("")

	// Collect declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *parser.FuncDecl:
			g.functions[d.Name.Name] = d
		case *parser.VarDecl:
			g.vars[d.Name.Name] = d
		}
	}

	// Global variables
	for name, v := range g.vars {
		g.emitGlobalVar(name, v)
	}

	g.emit("")

	// Functions
	for _, decl := range file.Decls {
		if fn, ok := decl.(*parser.FuncDecl); ok {
			g.emitFunction(fn)
		}
	}

	return g.output.String()
}

func (g *Generator) emit(s string) {
	g.output.WriteString(s + "\n")
}

func (g *Generator) emitf(format string, args ...interface{}) {
	g.emit(fmt.Sprintf(format, args...))
}

func (g *Generator) emitGlobalVar(name string, v *parser.VarDecl) {
	typ := g.tr.resolveType(v.Type)
	align := ""
	if strings.Contains(v.Directive, "align") {
		align = ", align 4096"
	}

	if v.Value != nil {
		if lit, ok := v.Value.(*parser.BasicLit); ok {
			g.emitf("@%s = global %s %s%s", name, typ.LLVMType(), literalValue(lit), align)
		} else {
			g.emitf("@%s = global %s 0%s", name, typ.LLVMType(), align)
		}
	} else {
		g.emitf("@%s = global %s 0%s", name, typ.LLVMType(), align)
	}
}

func (g *Generator) emitFunction(fn *parser.FuncDecl) {
	linkage := ""
	for _, d := range fn.Directives {
		if strings.Contains(d, "export") {
			linkage = "define"
		}
	}
	if linkage == "" {
		linkage = "define internal"
	}

	retType := "void"
	if len(fn.Type.Results) > 0 {
		retType = g.tr.resolveType(fn.Type.Results[0].Type).LLVMType()
	}

	params := ""
	for i, p := range fn.Type.Params {
		if i > 0 {
			params += ", "
		}
		params += g.tr.resolveType(p.Type).LLVMType() + " %" + p.Name.Name
	}

	g.emitf("%s %s @%s(%s) {", linkage, retType, fn.Name.Name, params)
	g.emit("entry:")

	if fn.Body != nil {
		for _, stmt := range fn.Body.Stmts {
			g.emitStmt(stmt)
		}
	}

	g.emit("  ret void")
	g.emit("}")
	g.emit("")
}

func (g *Generator) emitStmt(stmt parser.Stmt) {
	switch s := stmt.(type) {
	case *parser.DeclStmt:
		if vd, ok := s.Decl.(*parser.VarDecl); ok {
			typ := g.tr.resolveType(vd.Type)
			g.emitf("  %%v.%s = alloca %s", vd.Name.Name, typ.LLVMType())
			if vd.Value != nil {
				val := g.emitExpr(vd.Value)
				g.emitf("  store %s %s, %s* %%v.%s", typ.LLVMType(), val, typ.LLVMType(), vd.Name.Name)
			}
		}

	case *parser.AssignStmt:
		for i, lhs := range s.Lhs {
			if ident, ok := lhs.(*parser.Ident); ok {
				rhs := g.emitExpr(s.Rhs[i])
				typ := g.tr.exprType(s.Rhs[i])
				g.emitf("  store %s %s, %s* %%v.%s", typ.LLVMType(), rhs, typ.LLVMType(), ident.Name)
			}
		}

	case *parser.ReturnStmt:
		if len(s.Results) > 0 {
			val := g.emitExpr(s.Results[0])
			typ := g.tr.exprType(s.Results[0])
			g.emitf("  ret %s %s", typ.LLVMType(), val)
		} else {
			g.emit("  ret void")
		}

	case *parser.ExprStmt:
		g.emitExpr(s.X)

	case *parser.BlockStmt:
		for _, inner := range s.Stmts {
			g.emitStmt(inner)
		}

	case *parser.IfStmt:
		cond := g.emitExpr(s.Cond)
		lTrue := g.newLabel()
		lFalse := g.newLabel()
		lEnd := g.newLabel()
		g.emitf("  br i1 %s, label %%%s, label %%%s", cond, lTrue, lFalse)
		g.emitf("%s:", lTrue)
		g.emitStmt(s.Body)
		g.emitf("  br label %%%s", lEnd)
		g.emitf("%s:", lFalse)
		if s.Else != nil {
			g.emitStmt(s.Else)
		}
		g.emitf("  br label %%%s", lEnd)
		g.emitf("%s:", lEnd)

	case *parser.ForStmt:
		lStart := g.newLabel()
		lBody := g.newLabel()
		lEnd := g.newLabel()
		if s.Init != nil {
			g.emitStmt(s.Init)
		}
		g.emitf("  br label %%%s", lStart)
		g.emitf("%s:", lStart)
		if s.Cond != nil {
			cond := g.emitExpr(s.Cond)
			g.emitf("  br i1 %s, label %%%s, label %%%s", cond, lBody, lEnd)
		} else {
			g.emitf("  br label %%%s", lBody)
		}
		g.emitf("%s:", lBody)
		g.emitStmt(s.Body)
		if s.Post != nil {
			g.emitStmt(s.Post)
		}
		g.emitf("  br label %%%s", lStart)
		g.emitf("%s:", lEnd)
	}
}

func (g *Generator) emitExpr(expr parser.Expr) string {
	switch e := expr.(type) {
	case *parser.Ident:
		typ := g.tr.exprType(expr)
		g.tmpCnt++
		g.emitf("  %%t%d = load %s, %s* %%v.%s", g.tmpCnt, typ.LLVMType(), typ.LLVMType(), e.Name)
		return fmt.Sprintf("%%t%d", g.tmpCnt)

	case *parser.BasicLit:
		return literalValue(e)

	case *parser.BinaryExpr:
		x := g.emitExpr(e.X)
		y := g.emitExpr(e.Y)
		typ := g.tr.exprType(e.X)
		g.tmpCnt++
		op := g.binOp(e.Op, typ)
		g.emitf("  %%t%d = %s %s %s, %s", g.tmpCnt, op, typ.LLVMType(), x, y)
		return fmt.Sprintf("%%t%d", g.tmpCnt)

	case *parser.CallExpr:
		return g.emitCall(e)

	case *parser.UnaryExpr:
		x := g.emitExpr(e.X)
		typ := g.tr.exprType(e.X)
		g.tmpCnt++
		if e.Op == lexer.T_MINUS {
			g.emitf("  %%t%d = sub %s 0, %s", g.tmpCnt, typ.LLVMType(), x)
		} else if e.Op == lexer.T_NOT {
			g.emitf("  %%t%d = xor i1 %s, 1", g.tmpCnt, x)
		}
		return fmt.Sprintf("%%t%d", g.tmpCnt)

	default:
		return "0"
	}
}

func (g *Generator) emitCall(call *parser.CallExpr) string {
	// Handle asm package
	if sel, ok := call.Fun.(*parser.SelectorExpr); ok {
		if pkg, ok := sel.X.(*parser.Ident); ok && pkg.Name == "asm" {
			return g.emitAsmCall(sel.Sel.Name, call.Args)
		}
	}

	funName := ""
	if ident, ok := call.Fun.(*parser.Ident); ok {
		funName = ident.Name
	}

	fn := g.functions[funName]
	if fn == nil {
		return "0"
	}

	args := ""
	for i, arg := range call.Args {
		if i > 0 {
			args += ", "
		}
		val := g.emitExpr(arg)
		typ := g.tr.exprType(arg)
		args += typ.LLVMType() + " " + val
	}

	g.tmpCnt++
	retType := "void"
	if len(fn.Type.Results) > 0 {
		retType = g.tr.resolveType(fn.Type.Results[0].Type).LLVMType()
	}
	g.emitf("  %%t%d = call %s @%s(%s)", g.tmpCnt, retType, funName, args)

	if retType == "void" {
		return ""
	}
	return fmt.Sprintf("%%t%d", g.tmpCnt)
}

func (g *Generator) emitAsmCall(name string, args []parser.Expr) string {
	if len(args) > 0 {
		if lit, ok := args[0].(*parser.BasicLit); ok {
			instr := strings.Trim(lit.Value, "\"")
			instr = strings.ReplaceAll(instr, "`", "")
			// In LLVM inline asm:
			// - $ is used for operands, so $$ escapes to literal $
			// - % is used for registers in AT&T syntax, %% escapes to literal %
			// But for AT&T syntax, we keep % for registers
			// So we only escape $ to $$ for immediate values
			instr = strings.ReplaceAll(instr, "$", "$$")
			instr = strings.ReplaceAll(instr, "\n", "\\n")
			g.emitf("  call void asm sideeffect \"%s\", \"\"()", instr)
		}
	}
	return ""
}

func (g *Generator) binOp(op lexer.TokenType, typ types.Type) string {
	isFloat := false
	if basic, ok := typ.(*types.Basic); ok {
		isFloat = basic.Kind == types.Float32 || basic.Kind == types.Float64
	}

	switch op {
	case lexer.T_PLUS:
		if isFloat {
			return "fadd"
		}
		return "add"
	case lexer.T_MINUS:
		if isFloat {
			return "fsub"
		}
		return "sub"
	case lexer.T_STAR:
		if isFloat {
			return "fmul"
		}
		return "mul"
	case lexer.T_SLASH:
		if isFloat {
			return "fdiv"
		}
		return "sdiv"
	case lexer.T_EQ:
		if isFloat {
			return "fcmp oeq"
		}
		return "icmp eq"
	case lexer.T_NE:
		return "icmp ne"
	case lexer.T_LT:
		return "icmp slt"
	case lexer.T_LE:
		return "icmp sle"
	case lexer.T_GT:
		return "icmp sgt"
	case lexer.T_GE:
		return "icmp sge"
	case lexer.T_AND:
		return "and"
	case lexer.T_OR:
		return "or"
	}
	return "add"
}

func (g *Generator) newLabel() string {
	g.labelCnt++
	return fmt.Sprintf("L%d", g.labelCnt)
}