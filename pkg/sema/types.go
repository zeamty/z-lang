package sema

import (
	"github.com/zeamty/z-lang/pkg/lexer"
	"github.com/zeamty/z-lang/pkg/parser"
)

type TypeChecker struct {
	errors    *ErrorCollector
	scope     *Scope
	functions map[string]*parser.FuncDecl
}

func NewTypeChecker(errors *ErrorCollector, scope *Scope) *TypeChecker {
	return &TypeChecker{
		errors:    errors,
		scope:     scope,
		functions: make(map[string]*parser.FuncDecl),
	}
}

func (tc *TypeChecker) CollectDecls(decls []parser.Decl) {
	for _, d := range decls {
		switch d := d.(type) {
		case *parser.FuncDecl:
			tc.functions[d.Name.Name] = d
			tc.scope.Define(d.Name.Name, SymFunc, d, d.Type, 0)
		case *parser.TypeDecl:
			tc.scope.Define(d.Name.Name, SymType, d, d.Type, 0)
		case *parser.VarDecl:
			tc.scope.Define(d.Name.Name, SymVar, d, d.Type, 0)
		case *parser.ConstDecl:
			tc.scope.Define(d.Name.Name, SymConst, d, d.Type, 0)
		case *parser.VarBlock:
			for _, v := range d.Vars {
				tc.scope.Define(v.Name.Name, SymVar, v, v.Type, 0)
			}
		case *parser.ConstBlock:
			for _, c := range d.Consts {
				tc.scope.Define(c.Name.Name, SymConst, c, c.Type, 0)
			}
		}
	}
}

func (tc *TypeChecker) CheckExpr(expr parser.Expr) parser.Type {
	switch e := expr.(type) {
	case *parser.Ident:
		// Check built-in types first
		if isBuiltinType(e.Name) {
			return &parser.Ident{Name: e.Name}
		}
		sym := tc.scope.Lookup(e.Name)
		if sym == nil {
			tc.errors.Errorf(0, 0, "undefined identifier '%s'", e.Name)
			return nil
		}
		return sym.Type

	case *parser.BasicLit:
		switch e.Type {
		case lexer.T_INT:
			return &parser.Ident{Name: "int"}
		case lexer.T_FLOAT:
			return &parser.Ident{Name: "float64"}
		case lexer.T_STRING:
			return &parser.Ident{Name: "string"}
		case lexer.T_TRUE, lexer.T_FALSE:
			return &parser.Ident{Name: "bool"}
		case lexer.T_NIL:
			return &parser.Ident{Name: "nil"}
		}

	case *parser.BinaryExpr:
		lType := tc.CheckExpr(e.X)
		rType := tc.CheckExpr(e.Y)
		if lType != nil && rType != nil {
			return lType
		}

	case *parser.UnaryExpr:
		return tc.CheckExpr(e.X)

	case *parser.CallExpr:
		if ident, ok := e.Fun.(*parser.Ident); ok {
			if fn, ok := tc.functions[ident.Name]; ok {
				if len(fn.Type.Results) > 0 {
					return fn.Type.Results[0].Type
				}
			}
		}

	case *parser.SelectorExpr:
		return tc.CheckExpr(e.X)

	case *parser.ParenExpr:
		return tc.CheckExpr(e.X)

	case *parser.StarExpr:
		base := tc.CheckExpr(e.X)
		if base != nil {
			if pt, ok := base.(*parser.PointerType); ok {
				return pt.Base
			}
		}
		return nil

	case *parser.AddrExpr:
		// &x returns a pointer to x's type
		base := tc.CheckExpr(e.X)
		if base != nil {
			return &parser.PointerType{Base: base}
		}
		return nil

	case *parser.SliceExpr:
		// arr[low:high] returns a slice of the array's element type
		base := tc.CheckExpr(e.X)
		if arr, ok := base.(*parser.ArrayType); ok {
			return &parser.SliceType{Elt: arr.Elt}
		}
		// If base is already a slice, return slice type
		if sl, ok := base.(*parser.SliceType); ok {
			return sl
		}
		return nil

	case *parser.KeyedElement:
		// Keyed element in struct literal: Key: Value
		return tc.CheckExpr(e.Value)

	case *parser.CastExpr:
		return e.Type
	}

	return nil
}

func (tc *TypeChecker) CheckStmt(stmt parser.Stmt) {
	switch s := stmt.(type) {
	case *parser.AssignStmt:
		for _, rhs := range s.Rhs {
			tc.CheckExpr(rhs)
		}
	case *parser.ReturnStmt:
		for _, r := range s.Results {
			tc.CheckExpr(r)
		}
	case *parser.ExprStmt:
		tc.CheckExpr(s.X)
	case *parser.IncDecStmt:
		tc.CheckExpr(s.X)
	case *parser.IfStmt:
		tc.CheckExpr(s.Cond)
		tc.pushScope()
		tc.CheckStmt(s.Body)
		tc.popScope()
		if s.Else != nil {
			tc.pushScope()
			tc.CheckStmt(s.Else)
			tc.popScope()
		}
	case *parser.ForStmt:
		tc.pushScope()
		tc.CheckStmt(s.Body)
		tc.popScope()
	case *parser.ForRangeStmt:
		tc.pushScope()
		tc.CheckStmt(s.Body)
		tc.popScope()
	case *parser.BlockStmt:
		tc.pushScope()
		for _, inner := range s.Stmts {
			tc.CheckStmt(inner)
		}
		tc.popScope()
	case *parser.SwitchStmt:
		tc.CheckStmt(s.Body)
	case *parser.LabeledStmt:
		tc.CheckStmt(s.Stmt)
	case *parser.DeclStmt:
		if vd, ok := s.Decl.(*parser.VarDecl); ok {
			tc.scope.Define(vd.Name.Name, SymVar, vd, vd.Type, 0)
			if vd.Value != nil {
				tc.CheckExpr(vd.Value)
			}
		}
	}
}

func (tc *TypeChecker) pushScope() {
	tc.scope = NewScope(ScopeBlock, tc.scope)
}

func (tc *TypeChecker) popScope() {
	if tc.scope.Parent != nil {
		tc.scope = tc.scope.Parent
	}
}

func isBuiltinType(name string) bool {
	switch name {
	case "bool", "int8", "int16", "int32", "int64", "int",
		"uint8", "uint16", "uint32", "uint64", "uint", "uintptr",
		"float32", "float64", "string", "rune", "byte":
		return true
	}
	return false
}
