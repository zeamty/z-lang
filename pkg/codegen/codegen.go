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

	// Pre-scan for string literals before declaring them
	g.collectStrings(file)

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
	g.tr.currentFn = fn
	g.tr.locals = make(map[string]types.Type)
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
		if g.isParam(e.Name) {
			return "%" + e.Name
		}
		typ := g.tr.exprType(expr)
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s, %s* %%v.%s", tmp, typ.LLVMType(), typ.LLVMType(), e.Name)
		return tmp

	case *parser.BasicLit:
		if e.Type == lexer.T_STRING {
			strName := g.emitter.newString(e.Value)
			strLen := len(strings.Trim(e.Value, "\""))
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = getelementptr [%d x i8], [%d x i8]* %s, i64 0, i64 0",
				tmp, strLen+1, strLen+1, strName)
			return tmp
		}
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
		// For local arrays, use the alloca pointer directly
		var base string
		if ident, ok := e.X.(*parser.Ident); ok {
			base = "%v." + ident.Name
		} else {
			base = g.emitExpr(e.X)
		}
		idx := g.emitExpr(e.Index)
		arrType := g.tr.exprType(e.X)
		eltType := arrType
		if arr, ok := arrType.(*types.Array); ok {
			eltType = arr.Elt
		}
		ptrTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 0, i64 %s", ptrTmp, arrType.LLVMType(), arrType.LLVMType(), base, idx)
		// Load the element value
		valTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s, %s* %s", valTmp, eltType.LLVMType(), eltType.LLVMType(), ptrTmp)
		return valTmp

	case *parser.CompositeLit:
		// Array or struct literal initialization
		if e.Type != nil {
			typ := g.tr.resolveType(e.Type)
			// Handle array type
			if arr, ok := typ.(*types.Array); ok {
				// Allocate array on stack
				tmp := g.emitter.newTmp()
				g.emitter.emitf("%s = alloca %s", tmp, arr.LLVMType())
				// Initialize each element
				for i, elt := range e.Elts {
					eltTmp := g.emitter.newTmp()
					g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 0, i64 %d", eltTmp, arr.LLVMType(), arr.LLVMType(), tmp, i)
					val := g.emitExpr(elt)
					g.emitter.emitf("store %s %s, %s* %s", arr.Elt.LLVMType(), val, arr.Elt.LLVMType(), eltTmp)
				}
				return tmp
			}
			// Handle struct type
			if st, ok := typ.(*types.Struct); ok {
				// Allocate struct on stack
				tmp := g.emitter.newTmp()
				g.emitter.emitf("%s = alloca %s", tmp, st.LLVMType())
				// Initialize each field
				for i, elt := range e.Elts {
					fieldTmp := g.emitter.newTmp()
					g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 0, i32 %d", fieldTmp, st.LLVMType(), st.LLVMType(), tmp, i)
					val := g.emitExpr(elt)
					if i < len(st.Fields) {
						g.emitter.emitf("store %s %s, %s* %s", st.Fields[i].Type.LLVMType(), val, st.Fields[i].Type.LLVMType(), fieldTmp)
					}
				}
				return tmp
			}
		}
		// Fallback: allocate unnamed struct
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = alloca i64", tmp)
		if len(e.Elts) > 0 {
			val := g.emitExpr(e.Elts[0])
			g.emitter.emitf("store i64 %s, i64* %s", val, tmp)
		}
		return tmp

	case *parser.ParenExpr:
		return g.emitExpr(e.X)

	case *parser.CastExpr:
		val := g.emitExpr(e.X)
		srcType := g.tr.exprType(e.X)
		dstType := g.tr.resolveType(e.Type)
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = bitcast %s %s to %s", tmp, srcType.LLVMType(), val, dstType.LLVMType())
		return tmp

	case *parser.StarExpr:
		ptr := g.emitExpr(e.X)
		typ := g.tr.exprType(e)
		tmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s, %s* %s", tmp, typ.LLVMType(), typ.LLVMType(), ptr)
		return tmp
	case *parser.AddrExpr:
		// Address-of operation: &x returns pointer to x
		switch x := e.X.(type) {
		case *parser.Ident:
			// For local variables, the alloca result is already a pointer
			return "%v." + x.Name
		case *parser.StarExpr:
			// &*p just returns p (the pointer itself)
			return g.emitExpr(x.X)
		default:
			// For other expressions, need to store to temp and return address
			val := g.emitExpr(x)
			typ := g.tr.exprType(x)
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = alloca %s", tmp, typ.LLVMType())
			g.emitter.emitf("store %s %s, %s* %s", typ.LLVMType(), val, typ.LLVMType(), tmp)
			return tmp
		}

	case *parser.SliceExpr:
		// Slice expression: arr[low:high] returns a slice
		// For local arrays, use the alloca pointer directly
		var base string
		if ident, ok := e.X.(*parser.Ident); ok {
			base = "%v." + ident.Name
		} else {
			base = g.emitExpr(e.X)
		}
		baseType := g.tr.exprType(e.X)
		var lowVal, highVal int64
		var lowReg string = "0"
		if e.Low != nil {
			lowReg = g.emitExpr(e.Low)
			if lit, ok := e.Low.(*parser.BasicLit); ok {
				lowVal = types.ParseInt(lit.Value)
			} else {
				lowVal = 0
			}
		} else {
			lowVal = 0
		}
		if e.High != nil {
			_ = g.emitExpr(e.High)
			if lit, ok := e.High.(*parser.BasicLit); ok {
				highVal = types.ParseInt(lit.Value)
			} else {
				highVal = 0
			}
		} else {
			// Default high is array length
			if arrTyp, ok := baseType.(*types.Array); ok {
				highVal = int64(arrTyp.Len)
			} else {
				highVal = 0
			}
		}
		// Compute slice length
		sliceLen := highVal - lowVal
		// Compute pointer to first element
		eltType := baseType
		if arrTyp, ok := baseType.(*types.Array); ok {
			eltType = arrTyp.Elt
		}
		ptrTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 %s", ptrTmp, baseType.LLVMType(), baseType.LLVMType(), base, lowReg)
		// Allocate slice struct {ptr, len}
		sliceTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = alloca {%s*, i64}", sliceTmp, eltType.LLVMType())
		// Store pointer
		ptrFieldTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr {%s*, i64}, {%s*, i64}* %s, i32 0, i32 0", ptrFieldTmp, eltType.LLVMType(), eltType.LLVMType(), sliceTmp)
		g.emitter.emitf("store %s* %s, %s** %s", eltType.LLVMType(), ptrTmp, eltType.LLVMType(), ptrFieldTmp)
		// Store length
		lenFieldTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr {%s*, i64}, {%s*, i64}* %s, i32 0, i32 1", lenFieldTmp, eltType.LLVMType(), eltType.LLVMType(), sliceTmp)
		g.emitter.emitf("store i64 %d, i64* %s", sliceLen, lenFieldTmp)
		// Return the slice struct value
		retTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load {%s*, i64}, {%s*, i64}* %s", retTmp, eltType.LLVMType(), eltType.LLVMType(), sliceTmp)
		return retTmp

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
		// Truncate/extend to match parameter type if needed
		if i < len(fn.Type.Params) {
			paramType := g.tr.resolveType(fn.Type.Params[i].Type)
			if typ.LLVMType() != paramType.LLVMType() {
				tmp := g.emitter.newTmp()
				if paramType.Size() < typ.Size() {
					g.emitter.emitf("%s = trunc %s %s to %s", tmp, typ.LLVMType(), val, paramType.LLVMType())
				} else {
					g.emitter.emitf("%s = sext %s %s to %s", tmp, typ.LLVMType(), val, paramType.LLVMType())
				}
				val = tmp
			}
			typ = paramType
		}
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
			g.tr.locals[vd.Name.Name] = typ
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
					g.tr.locals[ident.Name] = typ
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
			} else if se, ok := lhs.(*parser.StarExpr); ok {
				rhs := g.emitExpr(s.Rhs[i])
				ptr := g.emitExpr(se.X)
				ptrType := g.tr.exprType(se.X)
				baseType := ptrType
				if p, ok := ptrType.(*types.Pointer); ok {
					baseType = p.Base
				}
				rhsType := g.tr.exprType(s.Rhs[i])
				if rhsType.LLVMType() != baseType.LLVMType() {
					tmp := g.emitter.newTmp()
					g.emitter.emitf("%s = trunc %s %s to %s", tmp, rhsType.LLVMType(), rhs, baseType.LLVMType())
					rhs = tmp
				}
				g.emitter.emitf("store %s %s, %s* %s",
					baseType.LLVMType(), rhs, baseType.LLVMType(), ptr)
			} else if ie, ok := lhs.(*parser.IndexExpr); ok {
				// arr[i] = value: compute pointer and store
				rhs := g.emitExpr(s.Rhs[i])
				// Get base pointer
				var base string
				if ident, ok := ie.X.(*parser.Ident); ok {
					base = "%v." + ident.Name
				} else {
					base = g.emitExpr(ie.X)
				}
				idx := g.emitExpr(ie.Index)
				arrType := g.tr.exprType(ie.X)
				eltType := arrType
				if arr, ok := arrType.(*types.Array); ok {
					eltType = arr.Elt
				}
				ptrTmp := g.emitter.newTmp()
				g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 0, i64 %s", ptrTmp, arrType.LLVMType(), arrType.LLVMType(), base, idx)
				rhsType := g.tr.exprType(s.Rhs[i])
				if rhsType.LLVMType() != eltType.LLVMType() {
					tmp := g.emitter.newTmp()
					g.emitter.emitf("%s = trunc %s %s to %s", tmp, rhsType.LLVMType(), rhs, eltType.LLVMType())
					rhs = tmp
				}
				g.emitter.emitf("store %s %s, %s* %s", eltType.LLVMType(), rhs, eltType.LLVMType(), ptrTmp)
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
	// For range loop: for key, value := range expr { body }
	// 1. Get the expression (array/slice pointer)
	// 2. Get length
	// 3. Loop with counter, bind key/value each iteration

	lCond := g.emitter.newLabel()
	lBody := g.emitter.newLabel()
	lInc := g.emitter.newLabel()
	lEnd := g.emitter.newLabel()
	g.loopEnds[""] = lEnd
	g.loopContinues[""] = lInc

	// Get base pointer - for local arrays, use the alloca address directly
	basePtr := ""
	baseType := g.tr.exprType(s.X)
	if ident, ok := s.X.(*parser.Ident); ok {
		// For local variables, the alloca is already a pointer to the array
		basePtr = "%v." + ident.Name
	} else {
		basePtr = g.emitExpr(s.X)
	}

	var eltType types.Type
	var arrLen int
	var lenPtr string = ""

	// Determine element type and length
	if arrTyp, ok := baseType.(*types.Array); ok {
		eltType = arrTyp.Elt
		arrLen = arrTyp.Len
	} else if sl, ok := baseType.(*types.Slice); ok {
		eltType = sl.Elt
		// For slices, we need to load the length from slice struct
		// slice struct is {ptr, len}, at index 1
		lenPtrTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr {%s*, i64}, {%s*, i64}* %s, i64 0, i32 1", lenPtrTmp, eltType.LLVMType(), eltType.LLVMType(), basePtr)
		lenValTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load i64, i64* %s", lenValTmp, lenPtrTmp)
		lenPtr = lenValTmp
		// Also get the actual data pointer
		dataPtrTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = getelementptr {%s*, i64}, {%s*, i64}* %s, i64 0, i32 0", dataPtrTmp, eltType.LLVMType(), eltType.LLVMType(), basePtr)
		dataValTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s*, %s** %s", dataValTmp, eltType.LLVMType(), eltType.LLVMType(), dataPtrTmp)
		basePtr = dataValTmp
		arrLen = 0 // dynamic length, use lenPtr
	} else {
		// Default: treat as single element
		eltType = baseType
		arrLen = 1
	}

	// Allocate counter (index) variable
	idxTmp := g.emitter.newTmp()
	g.emitter.emitf("%s = alloca i64", idxTmp)
	g.emitter.emitf("store i64 0, i64* %s", idxTmp)

	// Allocate key variable (if needed)
	if s.Key != nil {
		g.tr.locals[s.Key.Name] = types.BasicTypes["int"]
		keyTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = alloca i64", keyTmp)
	}

	// Allocate value variable (if needed)
	if s.Value != nil {
		g.tr.locals[s.Value.Name] = eltType
		valTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = alloca %s", valTmp, eltType.LLVMType())
	}

	// Jump to condition
	g.emitter.emitf("br label %%%s", lCond)
	g.emitter.emitRawf("%s:", lCond)

	// Load counter and check bounds
	idxVal := g.emitter.newTmp()
	g.emitter.emitf("%s = load i64, i64* %s", idxVal, idxTmp)

	// Compare with length
	condTmp := g.emitter.newTmp()
	if lenPtr != "" {
		// Slice: compare with dynamic length
		g.emitter.emitf("%s = icmp ult i64 %s, %s", condTmp, idxVal, lenPtr)
	} else {
		// Array: compare with constant length
		g.emitter.emitf("%s = icmp ult i64 %s, %d", condTmp, idxVal, arrLen)
	}
	g.emitter.emitf("br i1 %s, label %%%s, label %%%s", condTmp, lBody, lEnd)

	// Body
	g.emitter.emitRawf("%s:", lBody)

	// Bind key (index)
	if s.Key != nil {
		g.emitter.emitf("store i64 %s, i64* %%v.%s", idxVal, s.Key.Name)
	}

	// Bind value (element at index)
	if s.Value != nil {
		eltPtrTmp := g.emitter.newTmp()
		if lenPtr == "" {
			// Array: GEP with array type
			g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 0, i64 %s", eltPtrTmp, baseType.LLVMType(), baseType.LLVMType(), basePtr, idxVal)
		} else {
			// Slice: GEP with element type (pointer to first element)
			g.emitter.emitf("%s = getelementptr %s, %s* %s, i64 %s", eltPtrTmp, eltType.LLVMType(), eltType.LLVMType(), basePtr, idxVal)
		}
		eltValTmp := g.emitter.newTmp()
		g.emitter.emitf("%s = load %s, %s* %s", eltValTmp, eltType.LLVMType(), eltType.LLVMType(), eltPtrTmp)
		g.emitter.emitf("store %s %s, %s* %%v.%s", eltType.LLVMType(), eltValTmp, eltType.LLVMType(), s.Value.Name)
	}

	// Execute body
	g.emitStmt(s.Body)

	g.emitter.emitf("br label %%%s", lInc)
	g.emitter.emitRawf("%s:", lInc)

	// Increment counter
	idxVal2 := g.emitter.newTmp()
	g.emitter.emitf("%s = load i64, i64* %s", idxVal2, idxTmp)
	incTmp := g.emitter.newTmp()
	g.emitter.emitf("%s = add i64 %s, 1", incTmp, idxVal2)
	g.emitter.emitf("store i64 %s, i64* %s", incTmp, idxTmp)

	g.emitter.emitf("br label %%%s", lCond)
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

func (g *Generator) isParam(name string) bool {
	if g.currentFn == nil {
		return false
	}
	for _, p := range g.currentFn.Type.Params {
		if p.Name.Name == name {
			return true
		}
	}
	return false
}

func (g *Generator) collectStrings(file *parser.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *parser.FuncDecl:
			g.collectStmtStrings(d.Body)
		case *parser.VarDecl:
			if d.Value != nil {
				g.collectExprStrings(d.Value)
			}
		}
	}
}

func (g *Generator) collectStmtStrings(stmt parser.Stmt) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *parser.ExprStmt:
		g.collectExprStrings(s.X)
	case *parser.AssignStmt:
		for _, rhs := range s.Rhs {
			g.collectExprStrings(rhs)
		}
		if s.Lhs != nil {
			for _, lhs := range s.Lhs {
				g.collectExprStrings(lhs)
			}
		}
	case *parser.DeclStmt:
		if vd, ok := s.Decl.(*parser.VarDecl); ok && vd.Value != nil {
			g.collectExprStrings(vd.Value)
		}
	case *parser.ReturnStmt:
		for _, r := range s.Results {
			g.collectExprStrings(r)
		}
	case *parser.IfStmt:
		g.collectExprStrings(s.Cond)
		g.collectStmtStrings(s.Body)
		g.collectStmtStrings(s.Else)
	case *parser.ForStmt:
		if s.Init != nil {
			g.collectStmtStrings(s.Init)
		}
		if s.Cond != nil {
			g.collectExprStrings(s.Cond)
		}
		if s.Post != nil {
			g.collectStmtStrings(s.Post)
		}
		g.collectStmtStrings(s.Body)
	case *parser.BlockStmt:
		for _, ss := range s.Stmts {
			g.collectStmtStrings(ss)
		}
	case *parser.SwitchStmt:
		if s.Tag != nil {
			g.collectExprStrings(s.Tag)
		}
		g.collectStmtStrings(s.Body)
	case *parser.BreakStmt, *parser.ContinueStmt, *parser.GotoStmt, *parser.EmptyStmt, *parser.IncDecStmt:
		// no expressions to collect
	}
}

func (g *Generator) collectExprStrings(expr parser.Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *parser.BasicLit:
		if e.Type == lexer.T_STRING {
			g.emitter.newString(e.Value)
		}
	case *parser.BinaryExpr:
		g.collectExprStrings(e.X)
		g.collectExprStrings(e.Y)
	case *parser.UnaryExpr:
		g.collectExprStrings(e.X)
	case *parser.CallExpr:
		g.collectExprStrings(e.Fun)
		for _, arg := range e.Args {
			g.collectExprStrings(arg)
		}
	case *parser.CastExpr:
		g.collectExprStrings(e.X)
	case *parser.StarExpr:
		g.collectExprStrings(e.X)
	case *parser.AddrExpr:
		g.collectExprStrings(e.X)
	case *parser.ParenExpr:
		g.collectExprStrings(e.X)
	case *parser.IndexExpr:
		g.collectExprStrings(e.X)
		g.collectExprStrings(e.Index)
	case *parser.SelectorExpr:
		g.collectExprStrings(e.X)
	}
}
