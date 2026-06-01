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
	emitter   *Emitter
	tr        *TypeResolver
	functions map[string]*parser.FuncDecl
	vars      map[string]*parser.VarDecl
	structs   map[string]*types.Struct
	file      *parser.File

	// Function-local state
	currentFn     *parser.FuncDecl
	deferList     []*parser.CallExpr
	loopLabels    map[string]string
	loopEnds      map[string]string
	loopContinues map[string]string
}

func New() *Generator {
	return &Generator{
		functions: make(map[string]*parser.FuncDecl),
		vars:      make(map[string]*parser.VarDecl),
		structs:   make(map[string]*types.Struct),
	}
}

func (g *Generator) Generate(file *parser.File) string {
	g.file = file
	g.emitter = newEmitter()

	// Collect declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *parser.FuncDecl:
			g.functions[d.Name.Name] = d
		case *parser.VarDecl:
			g.vars[d.Name.Name] = d
		case *parser.TypeDecl:
			if st, ok := d.Type.(*parser.StructType); ok {
				s := &types.Struct{Name: d.Name.Name}
				for _, f := range st.Fields {
					s.Fields = append(s.Fields, &types.Var{
						Name: f.Name.Name,
						Type: g.tr.resolveType(f.Type),
					})
				}
				g.structs[d.Name.Name] = s
			}
		}
	}

	g.tr = newTypeResolver(g.vars, g.functions)
	g.tr.structs = g.structs

	g.emitter.emitRaw("; Z Language Compiled Output")
	g.emitter.emitRaw("; LLVM IR")
	g.emitter.emitRaw("")
	g.emitter.emitRaw("target datalayout = \"e-m:e-p:64:64-i64:64-n8:16:32:64-S128\"")
	g.emitter.emitRaw("target triple = \"x86_64-unknown-none-elf\"")
	g.emitter.emitRaw("")

	// Declare LLVM intrinsics
	g.emitter.emitRaw("declare void @llvm.memcpy.p0i8.p0i8.i64(i8* nocapture, i8* nocapture, i64, i1) nounwind")
	g.emitter.emitRaw("declare void @llvm.memset.p0i8.i64(i8* nocapture, i8, i64, i1) nounwind")
	g.emitter.emitRaw("")

	// String constants
	g.emitter.emitRaw(g.emitter.declareStrings())

	// Struct type definitions
	for name, st := range g.structs {
		fields := ""
		for i, f := range st.Fields {
			if i > 0 {
				fields += ", "
			}
			fields += f.Type.LLVMType()
		}
		if st.Packed {
			g.emitter.emitRawf("%%struct.%s = type <{%s}>", name, fields)
		} else {
			g.emitter.emitRawf("%%struct.%s = type {%s}", name, fields)
		}
	}

	// Global variables
	for name, v := range g.vars {
		g.emitGlobalVar(name, v)
	}

	g.emitter.emitRaw("")

	// Functions
	for _, decl := range file.Decls {
		if fn, ok := decl.(*parser.FuncDecl); ok {
			g.emitFunction(fn)
		}
	}

	return g.emitter.result()
}

func (g *Generator) emitGlobalVar(name string, v *parser.VarDecl) {
	typ := g.tr.resolveType(v.Type)
	align := ""
	if g.hasDirectiveAttr(v.Directive, "align") {
		if n := g.parseAlignAttr(v.Directive); n > 0 {
			align = fmt.Sprintf(", align %d", n)
		}
	}

	initVal := "0"
	if v.Value != nil {
		if lit, ok := v.Value.(*parser.BasicLit); ok {
			initVal = literalValue(lit)
		}
	}

	linkage := "internal"
	for _, d := range g.file.Directives {
		if strings.Contains(d, "linker") {
			linkage = "external"
		}
	}

	g.emitter.emitRawf("@%s = %s global %s %s%s", name, linkage, typ.LLVMType(), initVal, align)
}

func (g *Generator) emitFunction(fn *parser.FuncDecl) {
	g.currentFn = fn
	g.deferList = nil
	g.loopLabels = make(map[string]string)
	g.loopEnds = make(map[string]string)
	g.loopContinues = make(map[string]string)

	linkage := "define internal"
	callingConv := ""
	hasInterrupt := false

	for _, d := range fn.Directives {
		if strings.Contains(d, "export") {
			linkage = "define"
		}
		if strings.Contains(d, "interrupt") {
			callingConv = " x86_intrcc"
			hasInterrupt = true
		}
		if strings.Contains(d, "inline") {
			linkage += " alwaysinline"
		}
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

	g.emitter.emitRawf("%s%s %s @%s(%s) {", linkage, callingConv, retType, fn.Name.Name, params)
	g.emitter.emitRaw("entry:")

	// Alloca named return values
	for _, r := range fn.Type.Results {
		if r.Name != nil && r.Name.Name != "" {
			rt := g.tr.resolveType(r.Type)
			g.emitter.emitf("%%ret.%s = alloca %s", r.Name.Name, rt.LLVMType())
			g.emitter.emitf("store %s 0, %s* %%ret.%s", rt.LLVMType(), rt.LLVMType(), r.Name.Name)
		}
	}

	if fn.Body != nil {
		for _, stmt := range fn.Body.Stmts {
			g.emitStmt(stmt)
		}
	}

	// Emit deferred calls at end
	g.emitDeferredCalls()

	if retType == "void" {
		g.emitter.emitRaw("  ret void")
	} else if len(fn.Type.Results) > 0 && fn.Type.Results[0].Name != nil {
		r := fn.Type.Results[0]
		rt := g.tr.resolveType(r.Type)
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s, %s* %%ret.%s", tmp, rt.LLVMType(), rt.LLVMType(), r.Name.Name)
		g.emitter.emitf("ret %s %s", rt.LLVMType(), tmp)
	}

	g.emitter.emitRaw("}")
	g.emitter.emitRaw("")
	_ = hasInterrupt
}

func (g *Generator) emitExpr(expr parser.Expr) string {
	switch e := expr.(type) {
	case *parser.Ident:
		typ := g.tr.exprType(expr)
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s, %s* %%v.%s", tmp, typ.LLVMType(), typ.LLVMType(), e.Name)
		return tmp

	case *parser.BasicLit:
		return literalValue(e)

	case *parser.BinaryExpr:
		x := g.emitExpr(e.X)
		y := g.emitExpr(e.Y)
		typ := g.tr.exprType(e.X)
		tmp := g.emitter.newTmp()
		op := g.binOp(e.Op, typ)
		g.emitter.emitf("%s = %s %s %s, %s", tmp, op, typ.LLVMType(), x, y)
		return tmp

	case *parser.CallExpr:
		return g.emitCall(e)

	case *parser.UnaryExpr:
		x := g.emitExpr(e.X)
		typ := g.tr.exprType(e.X)
		tmp := g.emitter.newTmp()
		switch e.Op {
		case lexer.T_MINUS:
			g.emitter.emitf("%s = sub %s 0, %s", tmp, typ.LLVMType(), x)
		case lexer.T_NOT:
			g.emitter.emitf("%s = xor i1 %s, 1", tmp, x)
		}
		return tmp

	case *parser.SelectorExpr:
		return g.emitExpr(e.X) + "." + e.Sel.Name

	case *parser.IndexExpr:
		base := g.emitExpr(e.X)
		idx := g.emitExpr(e.Index)
		typ := g.tr.exprType(e.X)
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 %s", tmp, typ.LLVMType(), typ.LLVMType(), base, idx)
		return tmp

	case *parser.CompositeLit:
		return "undef"

	case *parser.ParenExpr:
		return g.emitExpr(e.X)

	default:
		return "0"
	}
}

func (g *Generator) emitCall(call *parser.CallExpr) string {
	// Handle asm package intrinsics
	if sel, ok := call.Fun.(*parser.SelectorExpr); ok {
		if pkg, ok := sel.X.(*parser.Ident); ok {
			if pkg.Name == "asm" {
				return g.emitAsmCall(sel.Sel.Name, call.Args)
			}
			if pkg.Name == "mem" {
				return g.emitMemCall(sel.Sel.Name, call.Args)
			}
			if pkg.Name == "atomic" {
				return g.emitAtomicCall(sel.Sel.Name, call.Args)
			}
			if pkg.Name == "unsafe" {
				return g.emitUnsafeCall(sel.Sel.Name, call.Args)
			}
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

	retType := "void"
	if len(fn.Type.Results) > 0 {
		retType = g.tr.resolveType(fn.Type.Results[0].Type).LLVMType()
	}

	tmp := g.emitter.newTmp()
	if retType == "void" {
		g.emitter.emitf("call void @%s(%s)", funName, args)
		return ""
	}
	g.emitter.emitf("%s = call %s @%s(%s)", tmp, retType, funName, args)
	return tmp
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
	case lexer.T_PERCENT:
		return "srem"
	case lexer.T_SHL:
		return "shl"
	case lexer.T_SHR:
		return "ashr"
	case lexer.T_EQ:
		if isFloat {
			return "fcmp oeq"
		}
		return "icmp eq"
	case lexer.T_NE:
		if isFloat {
			return "fcmp one"
		}
		return "icmp ne"
	case lexer.T_LT:
		if isFloat {
			return "fcmp olt"
		}
		return "icmp slt"
	case lexer.T_LE:
		if isFloat {
			return "fcmp ole"
		}
		return "icmp sle"
	case lexer.T_GT:
		if isFloat {
			return "fcmp ogt"
		}
		return "icmp sgt"
	case lexer.T_GE:
		if isFloat {
			return "fcmp oge"
		}
		return "icmp sge"
	case lexer.T_AND:
		return "and"
	case lexer.T_OR:
		return "or"
	case lexer.T_XOR:
		return "xor"
	case lexer.T_AND_NOT:
		return "and"
	}
	return "add"
}

func (g *Generator) emitStmt(stmt parser.Stmt) {
	switch s := stmt.(type) {
	case *parser.DeclStmt:
		if vd, ok := s.Decl.(*parser.VarDecl); ok {
			typ := g.tr.resolveType(vd.Type)
			volatileStr := ""
			if g.hasDirectiveAttr(vd.Directive, "volatile") {
				volatileStr = " volatile"
			}
			g.emitter.emitf("%%v.%s = alloca %s", vd.Name.Name, typ.LLVMType())
			if vd.Value != nil {
				val := g.emitExpr(vd.Value)
				g.emitter.emitf("store%s %s %s, %s* %%v.%s",
					volatileStr, typ.LLVMType(), val, typ.LLVMType(), vd.Name.Name)
			}
		}

	case *parser.AssignStmt:
		for i, lhs := range s.Lhs {
			if ident, ok := lhs.(*parser.Ident); ok {
				rhs := g.emitExpr(s.Rhs[i])
				typ := g.tr.exprType(s.Rhs[i])

				if s.Tok == lexer.T_COLON_ASSIGN {
					g.emitter.emitf("%%v.%s = alloca %s", ident.Name, typ.LLVMType())
				}

				volatileStr := ""
				if v, ok := g.vars[ident.Name]; ok {
					if g.hasDirectiveAttr(v.Directive, "volatile") {
						volatileStr = " volatile"
					}
				}
				g.emitter.emitf("store%s %s %s, %s* %%v.%s",
					volatileStr, typ.LLVMType(), rhs, typ.LLVMType(), ident.Name)
			}
		}

	case *parser.ReturnStmt:
		g.emitDeferredCalls()
		if len(s.Results) > 0 {
			val := g.emitExpr(s.Results[0])
			typ := g.tr.exprType(s.Results[0])
			g.emitter.emitf("ret %s %s", typ.LLVMType(), val)
		} else {
			g.emitter.emitRaw("  ret void")
		}

	case *parser.ExprStmt:
		g.emitExpr(s.X)

	case *parser.BlockStmt:
		for _, inner := range s.Stmts {
			g.emitStmt(inner)
		}

	case *parser.IfStmt:
		g.emitIfStmt(s)

	case *parser.ForStmt:
		g.emitForStmt(s)

	case *parser.ForRangeStmt:
		g.emitForRangeStmt(s)

	case *parser.SwitchStmt:
		g.emitSwitchStmt(s)

	case *parser.DeferStmt:
		if s.Call != nil {
			g.deferList = append(g.deferList, s.Call)
		}

	case *parser.BreakStmt:
		var target string
		if s.Label != nil {
			target = g.loopEnds[s.Label.Name]
		} else {
			for _, v := range g.loopEnds {
				target = v
				break
			}
		}
		if target != "" {
			g.emitter.emitf("br label %%%s", target)
		}

	case *parser.ContinueStmt:
		var target string
		if s.Label != nil {
			target = g.loopContinues[s.Label.Name]
		} else {
			for _, v := range g.loopContinues {
				target = v
				break
			}
		}
		if target != "" {
			g.emitter.emitf("br label %%%s", target)
		}

	case *parser.GotoStmt:
		label := fmt.Sprintf("goto.%s", s.Label.Name)
		g.emitter.emitf("br label %%%s", label)

	// case *parser.LabeledStmt: // Not yet available in parser
	}
}

func (g *Generator) emitIfStmt(s *parser.IfStmt) {
	if s.Init != nil {
		g.emitStmt(s.Init)
	}
	cond := g.emitExpr(s.Cond)
	lTrue := g.emitter.newLabel()
	lFalse := g.emitter.newLabel()
	lEnd := g.emitter.newLabel()

	g.emitter.emitf("br i1 %s, label %%%s, label %%%s", cond, lTrue, lFalse)
	g.emitter.emitRawf("%s:", lTrue)
	g.emitStmt(s.Body)
	g.emitter.emitf("br label %%%s", lEnd)
	g.emitter.emitRawf("%s:", lFalse)
	if s.Else != nil {
		g.emitStmt(s.Else)
	}
	g.emitter.emitf("br label %%%s", lEnd)
	g.emitter.emitRawf("%s:", lEnd)
}

func (g *Generator) emitForStmt(s *parser.ForStmt) {
	lCond := g.emitter.newLabel()
	lBody := g.emitter.newLabel()
	lPost := g.emitter.newLabel()
	lEnd := g.emitter.newLabel()

	// Store loop labels for break/continue
	g.loopEnds[""] = lEnd
	g.loopContinues[""] = lPost

	if s.Init != nil {
		g.emitStmt(s.Init)
	}
	g.emitter.emitf("br label %%%s", lCond)
	g.emitter.emitRawf("%s:", lCond)
	if s.Cond != nil {
		cond := g.emitExpr(s.Cond)
		g.emitter.emitf("br i1 %s, label %%%s, label %%%s", cond, lBody, lEnd)
	} else {
		g.emitter.emitf("br label %%%s", lBody)
	}
	g.emitter.emitRawf("%s:", lBody)
	g.emitStmt(s.Body)
	g.emitter.emitf("br label %%%s", lPost)
	g.emitter.emitRawf("%s:", lPost)
	if s.Post != nil {
		g.emitStmt(s.Post)
	}
	g.emitter.emitf("br label %%%s", lCond)
	g.emitter.emitRawf("%s:", lEnd)

	// Clear loop labels
	delete(g.loopEnds, "")
	delete(g.loopContinues, "")
}

func (g *Generator) emitForRangeStmt(s *parser.ForRangeStmt) {
	lBody := g.emitter.newLabel()
	lEnd := g.emitter.newLabel()
	g.loopEnds[""] = lEnd
	g.loopContinues[""] = lBody

	g.emitExpr(s.X)
	g.emitter.emitf("br label %%%s", lBody)
	g.emitter.emitRawf("%s:", lBody)
	g.emitStmt(s.Body)
	g.emitter.emitf("br label %%%s", lEnd)
	g.emitter.emitRawf("%s:", lEnd)

	delete(g.loopEnds, "")
	delete(g.loopContinues, "")
}

func (g *Generator) emitSwitchStmt(s *parser.SwitchStmt) {
	lEnd := g.emitter.newLabel()

	if s.Tag != nil {
		tag := g.emitExpr(s.Tag)
		tagType := g.tr.exprType(s.Tag)
		lDefault := g.emitter.newLabel()

		// Collect cases
		var caseLabels []string
		var caseValues []string
		for _, stmt := range s.Body.Stmts {
			if cs, ok := stmt.(*parser.CaseStmt); ok {
				lCase := g.emitter.newLabel()
				caseLabels = append(caseLabels, lCase)
				for _, val := range cs.List {
					caseValues = append(caseValues, g.emitExpr(val))
				}
			}
		}

			ci := 0
		if len(caseLabels) > 0 {
			g.emitter.emitf("switch %s %s, label %%%s [", tagType.LLVMType(), tag, lDefault)
			for i, stmt := range s.Body.Stmts {
				if cs, ok := stmt.(*parser.CaseStmt); ok {
					for range cs.List {
						g.emitter.emitRawf("  %s %s, label %%%s", tagType.LLVMType(), caseValues[ci], caseLabels[i])
						ci++
					}
				}
			}
			g.emitter.emitRaw("]")
		}

		// Emit case bodies
		ci = 0
		for _, stmt := range s.Body.Stmts {
			if cs, ok := stmt.(*parser.CaseStmt); ok {
				g.emitter.emitRawf("%s:", caseLabels[ci])
				if cs.Body != nil {
					g.emitStmt(cs.Body)
				}
				g.emitter.emitf("br label %%%s", lEnd)
				ci++
			}
		}
		g.emitter.emitRawf("%s:", lDefault)
		g.emitter.emitf("br label %%%s", lEnd)
	} else {
		// Tagless switch: if-else chain
		nextLabel := ""
		for i, stmt := range s.Body.Stmts {
			if cs, ok := stmt.(*parser.CaseStmt); ok {
				lBody := g.emitter.newLabel()
				if nextLabel != "" {
					g.emitter.emitRawf("%s:", nextLabel)
				}
				nextLabel = g.emitter.newLabel()
				if i < len(s.Body.Stmts)-1 && len(cs.List) > 0 {
					cond := g.emitExpr(cs.List[0])
					g.emitter.emitf("br i1 %s, label %%%s, label %%%s", cond, lBody, nextLabel)
				} else {
					g.emitter.emitf("br label %%%s", lBody)
				}
				g.emitter.emitRawf("%s:", lBody)
				if cs.Body != nil {
					g.emitStmt(cs.Body)
				}
				g.emitter.emitf("br label %%%s", lEnd)
			}
		}
		if nextLabel != "" {
			g.emitter.emitRawf("%s:", nextLabel)
			g.emitter.emitf("br label %%%s", lEnd)
		}
	}

	g.emitter.emitRawf("%s:", lEnd)
}

func (g *Generator) emitDeferredCalls() {
	for i := len(g.deferList) - 1; i >= 0; i-- {
		g.emitCall(g.deferList[i])
	}
	g.deferList = nil
}

func (g *Generator) hasDirectiveAttr(directive, attr string) bool {
	return strings.Contains(directive, attr)
}

func (g *Generator) parseAlignAttr(directive string) int {
	for _, part := range strings.Fields(directive) {
		part = strings.Trim(part, "()")
		var val int
		n, _ := fmt.Sscanf(part, "align(%d)", &val)
		if n == 1 {
			return val
		}
	}
	return 0
}