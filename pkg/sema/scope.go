package sema

import (
	"github.com/zeamty/z-lang/pkg/parser"
)

type ScopeKind int

const (
	ScopePackage ScopeKind = iota
	ScopeFunc
	ScopeBlock
)

type Scope struct {
	Kind    ScopeKind
	Parent  *Scope
	Symbols map[string]*Symbol
}

type SymbolKind int

const (
	SymVar SymbolKind = iota
	SymConst
	SymFunc
	SymType
	SymParam
	SymLabel
)

type Symbol struct {
	Kind SymbolKind
	Name string
	Decl parser.Node
	Type parser.Type
	Used bool
	Line int
}

func NewScope(kind ScopeKind, parent *Scope) *Scope {
	return &Scope{
		Kind:    kind,
		Parent:  parent,
		Symbols: make(map[string]*Symbol),
	}
}

func (s *Scope) Define(name string, kind SymbolKind, decl parser.Node, typ parser.Type, line int) *Symbol {
	sym := &Symbol{Kind: kind, Name: name, Decl: decl, Type: typ, Line: line}
	s.Symbols[name] = sym
	return sym
}

func (s *Scope) Lookup(name string) *Symbol {
	if sym, ok := s.Symbols[name]; ok {
		return sym
	}
	if s.Parent != nil {
		return s.Parent.Lookup(name)
	}
	return nil
}