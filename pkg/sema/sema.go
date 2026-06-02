package sema

import (
	"github.com/zeamty/z-lang/pkg/parser"
)

type Analyzer struct {
	errors  *ErrorCollector
	scope   *Scope
	checker *TypeChecker
}

func NewAnalyzer() *Analyzer {
	errors := NewErrorCollector()
	scope := NewScope(ScopePackage, nil)
	return &Analyzer{
		errors:  errors,
		scope:   scope,
		checker: NewTypeChecker(errors, scope),
	}
}

func (a *Analyzer) Analyze(file *parser.File) *ErrorCollector {
	a.checker.CollectDecls(file.Decls)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *parser.FuncDecl:
			a.analyzeFunc(d)
		case *parser.TypeDecl:
			a.analyzeTypeDecl(d)
		}
	}

	return a.errors
}

func (a *Analyzer) analyzeFunc(fn *parser.FuncDecl) {
	funcScope := NewScope(ScopeFunc, a.scope)
	a.scope = funcScope
	a.checker.scope = funcScope

	for _, p := range fn.Type.Params {
		a.scope.Define(p.Name.Name, SymParam, fn, p.Type, 0)
	}

	for _, r := range fn.Type.Results {
		if r.Name != nil && r.Name.Name != "" {
			a.scope.Define(r.Name.Name, SymVar, fn, r.Type, 0)
		}
	}

	if fn.Body != nil {
		for _, stmt := range fn.Body.Stmts {
			a.checker.CheckStmt(stmt)
		}
	}

	a.validateDirectives(fn)
	a.scope = a.scope.Parent
	a.checker.scope = a.scope
}

func (a *Analyzer) analyzeTypeDecl(td *parser.TypeDecl) {
	if _, ok := td.Type.(*parser.StructType); ok {
		// Struct fields are validated during codegen
	}
	if _, ok := td.Type.(*parser.InterfaceType); ok {
		// Interface validation
	}
}

func (a *Analyzer) validateDirectives(fn *parser.FuncDecl) {
	hasInterrupt := false

	for _, d := range fn.Directives {
		if d == "//z:interrupt" {
			hasInterrupt = true
		}
	}

	if hasInterrupt {
		if len(fn.Type.Params) != 1 {
			a.errors.Errorf(0, 0, "interrupt handler '%s' must take exactly 1 parameter", fn.Name.Name)
		}
		if len(fn.Type.Results) > 0 {
			a.errors.Errorf(0, 0, "interrupt handler '%s' must not return values", fn.Name.Name)
		}
	}
}