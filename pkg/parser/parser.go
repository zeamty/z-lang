package parser

import (
	"fmt"

	"github.com/zeamty/z-lang/pkg/lexer"
)

// Parser parses Z source into AST
type Parser struct {
	lex   *lexer.Lexer
	tok   lexer.Token
	peek  lexer.Token

	directives []string // accumulated directives
	errors     []string
	iotaValue  int
}

// New creates a new Parser
func New(l *lexer.Lexer) *Parser {
	p := &Parser{lex: l}
	p.nextTok()
	p.nextTok()
	return p
}

func (p *Parser) nextTok() {
	p.tok = p.peek
	p.peek = p.lex.NextToken()
}

func (p *Parser) expect(t lexer.TokenType) lexer.Token {
	if p.tok.Type != t {
		p.error(fmt.Sprintf("expected %v, got %v", t, p.tok.Type))
		return p.tok
	}
	tok := p.tok
	p.nextTok()
	return tok
}

func (p *Parser) error(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d: %s", p.tok.Line, msg))
}

func (p *Parser) HasErrors() bool {
	return len(p.errors) > 0
}

func (p *Parser) Errors() []string {
	return p.errors
}

// ParseFile parses entire file
func (p *Parser) ParseFile() *File {
	file := &File{}

	// Collect directives at top
	p.collectDirectives()

	// Package clause
	file.Pkg = p.parsePackage()

	// Imports
	for p.tok.Type == lexer.T_IMPORT {
		imp := p.parseImport()
		switch d := imp.(type) {
		case *ImportDecl:
			file.Imports = append(file.Imports, d)
		case *ImportBlock:
			file.Imports = append(file.Imports, d.Imports...)
		}
	}

	// Top-level declarations
	for p.tok.Type != lexer.T_EOF {
		file.Decls = append(file.Decls, p.parseDecl())
	}

	return file
}

func (p *Parser) collectDirectives() {
	for {
		if p.tok.Type == lexer.T_DIRECTIVE || (p.tok.Type == lexer.T_COMMENT && len(p.tok.Literal) > 2 && p.tok.Literal[:3] == "//z") {
			p.directives = append(p.directives, p.tok.Literal)
			p.nextTok()
		} else {
			break
		}
	}
}

func (p *Parser) parsePackage() *PackageDecl {
	p.expect(lexer.T_PACKAGE)
	name := p.parseIdent()
	return &PackageDecl{Name: name}
}

func (p *Parser) parseImport() Decl {
	// Check for alias
	if p.tok.Type == lexer.T_IDENT && p.peek.Type == lexer.T_STRING {
		alias := p.parseIdent()
		decl := &ImportDecl{Name: alias}
		pathTok := p.expect(lexer.T_STRING)
		decl.Path = &BasicLit{Type: lexer.T_STRING, Value: pathTok.Literal}
		return decl
	}

	if p.tok.Type == lexer.T_STRING {
		decl := &ImportDecl{}
		decl.Path = &BasicLit{Type: lexer.T_STRING, Value: p.tok.Literal}
		p.nextTok()
		return decl
	}

	if p.tok.Type == lexer.T_LPAREN {
		p.nextTok()
		block := &ImportBlock{}
		for p.tok.Type != lexer.T_RPAREN {
			imp := &ImportDecl{}
			if p.tok.Type == lexer.T_IDENT {
				imp.Name = p.parseIdent()
			}
			pathTok := p.expect(lexer.T_STRING)
			imp.Path = &BasicLit{Type: lexer.T_STRING, Value: pathTok.Literal}
			block.Imports = append(block.Imports, imp)
		}
		p.expect(lexer.T_RPAREN)
		return block
	}

	return nil
}

func (p *Parser) parseDecl() Decl {
	// Collect any directives before this declaration
	p.collectDirectives()
	dirs := p.directives
	p.directives = nil

	switch p.tok.Type {
	case lexer.T_CONST:
		return p.parseConstDecl()
	case lexer.T_VAR:
		return p.parseVarDecl(dirs)
	case lexer.T_FUNC:
		return p.parseFuncDecl(dirs)
	case lexer.T_TYPE:
		return p.parseTypeDecl()
	default:
		p.error(fmt.Sprintf("unexpected token %v", p.tok.Type))
		p.nextTok()
		return nil
	}
}

func (p *Parser) parseConstDecl() Decl {
	p.expect(lexer.T_CONST)

	if p.tok.Type == lexer.T_LPAREN {
		p.nextTok()
		decl := &ConstBlock{}
		iotaVal := 0
		for p.tok.Type != lexer.T_RPAREN && p.tok.Type != lexer.T_EOF {
			p.iotaValue = iotaVal
			name := p.parseIdent()

			var typ Type
			if p.tok.Type != lexer.T_ASSIGN {
				typ = p.parseType()
			}

			p.expect(lexer.T_ASSIGN)
			val := p.parseExpr()
			decl.Consts = append(decl.Consts, &ConstDecl{Name: name, Type: typ, Value: val})
			iotaVal++
		}
		p.expect(lexer.T_RPAREN)
		return decl
	}

	name := p.parseIdent()
	var typ Type
	if p.tok.Type != lexer.T_ASSIGN {
		typ = p.parseType()
	}
	p.expect(lexer.T_ASSIGN)
	val := p.parseExpr()
	return &ConstDecl{Name: name, Type: typ, Value: val}
}

func (p *Parser) parseVarDecl(dirs []string) Decl {
	p.expect(lexer.T_VAR)

	if p.tok.Type == lexer.T_LPAREN {
		p.nextTok()
		decl := &VarBlock{}
		for p.tok.Type != lexer.T_RPAREN && p.tok.Type != lexer.T_EOF {
			p.collectDirectives()
			d := p.directives
			p.directives = nil
			name := p.parseIdent()

			var typ Type
			var val Expr
			if p.tok.Type == lexer.T_ASSIGN {
				p.nextTok()
				val = p.parseExpr()
			} else {
				typ = p.parseType()
				if p.tok.Type == lexer.T_ASSIGN {
					p.nextTok()
					val = p.parseExpr()
				}
			}

			directive := ""
			if len(d) > 0 {
				directive = d[0]
			}
			decl.Vars = append(decl.Vars, &VarDecl{Name: name, Type: typ, Value: val, Directive: directive})
		}
		p.expect(lexer.T_RPAREN)
		return decl
	}

	name := p.parseIdent()
	var typ Type
	var val Expr

	if p.tok.Type == lexer.T_ASSIGN {
		p.nextTok()
		val = p.parseExpr()
	} else {
		typ = p.parseType()
		if p.tok.Type == lexer.T_ASSIGN {
			p.nextTok()
			val = p.parseExpr()
		}
	}

	directive := ""
	if len(dirs) > 0 {
		directive = dirs[0]
	}

	return &VarDecl{Name: name, Type: typ, Value: val, Directive: directive}
}

func (p *Parser) parseFuncDecl(dirs []string) *FuncDecl {
	p.expect(lexer.T_FUNC)

	decl := &FuncDecl{Directives: dirs}

	// Check for receiver
	if p.tok.Type == lexer.T_LPAREN {
		p.nextTok()
		decl.Recv = &Field{
			Name: p.parseIdent(),
			Type: p.parseType(),
		}
		p.expect(lexer.T_RPAREN)
	}

	decl.Name = p.parseIdent()
	decl.Type = p.parseFuncType()

	if p.tok.Type == lexer.T_LBRACE {
		decl.Body = p.parseBlock()
	}

	return decl
}

func (p *Parser) parseTypeDecl() *TypeDecl {
	p.expect(lexer.T_TYPE)
	name := p.parseIdent()
	typ := p.parseType()
	return &TypeDecl{Name: name, Type: typ}
}

func (p *Parser) parseFuncType() *FuncType {
	ft := &FuncType{}

	p.expect(lexer.T_LPAREN)
	ft.Params = p.parseParamList()
	p.expect(lexer.T_RPAREN)

	// Results
	if p.tok.Type == lexer.T_LPAREN {
		p.nextTok()
		for p.tok.Type != lexer.T_RPAREN && p.tok.Type != lexer.T_EOF {
			field := &Field{}
			if p.tok.Type == lexer.T_IDENT {
				first := p.parseIdent()
				if p.tok.Type == lexer.T_COMMA || p.tok.Type == lexer.T_RPAREN {
					field.Type = first
				} else if p.tok.Type == lexer.T_IDENT {
					field.Name = first
					field.Type = p.parseType()
				} else {
					field.Type = first
				}
			} else {
				field.Type = p.parseType()
			}
			ft.Results = append(ft.Results, field)
			if p.tok.Type == lexer.T_COMMA {
				p.nextTok()
			}
		}
		p.expect(lexer.T_RPAREN)
	} else if p.tok.Type != lexer.T_LBRACE && p.tok.Type != lexer.T_SEMICOLON {
		ft.Results = []*Field{{Type: p.parseType()}}
	}

	return ft
}

func (p *Parser) parseParamList() []*Field {
	var fields []*Field

	for p.tok.Type != lexer.T_RPAREN && p.tok.Type != lexer.T_EOF {
		name := p.parseIdent()
		typ := p.parseType()
		fields = append(fields, &Field{Name: name, Type: typ})

		if p.tok.Type == lexer.T_COMMA {
			p.nextTok()
		} else {
			break
		}
	}

	return fields
}

func (p *Parser) parseType() Type {
	switch p.tok.Type {
	case lexer.T_IDENT:
		name := p.parseIdent()
		return name

	case lexer.T_LBRACK:
		p.nextTok()
		if p.tok.Type == lexer.T_RBRACK {
			// Slice type
			p.nextTok()
			return &SliceType{Elt: p.parseType()}
		}
		// Array type
		len := p.parseExpr()
		p.expect(lexer.T_RBRACK)
		return &ArrayType{Len: len, Elt: p.parseType()}

	case lexer.T_STAR:
		p.nextTok()
		return &PointerType{Base: p.parseType()}

	case lexer.T_STRUCT:
		return p.parseStructType()

	case lexer.T_INTERFACE:
		return p.parseInterfaceType()

	case lexer.T_FUNC:
		p.nextTok()
		return p.parseFuncType()

	default:
		p.error(fmt.Sprintf("unexpected type token %v", p.tok.Type))
		return &Ident{Name: "unknown"}
	}
}

func (p *Parser) parseStructType() *StructType {
	p.expect(lexer.T_STRUCT)
	p.expect(lexer.T_LBRACE)

	st := &StructType{}
	for p.tok.Type != lexer.T_RBRACE && p.tok.Type != lexer.T_EOF {
		dirs := p.directives
		p.directives = nil
		p.collectDirectives()

		field := &Field{}
		if len(dirs) > 0 {
			field.Directive = dirs[0]
		}

		field.Name = p.parseIdent()
		field.Type = p.parseType()
		st.Fields = append(st.Fields, field)
	}
	p.expect(lexer.T_RBRACE)

	return st
}

func (p *Parser) parseInterfaceType() *InterfaceType {
	p.expect(lexer.T_INTERFACE)
	p.expect(lexer.T_LBRACE)

	it := &InterfaceType{}
	for p.tok.Type != lexer.T_RBRACE && p.tok.Type != lexer.T_EOF {
		name := p.parseIdent()
		ft := p.parseFuncType()
		it.Methods = append(it.Methods, &Field{Name: name, Type: ft})
	}
	p.expect(lexer.T_RBRACE)

	return it
}

func (p *Parser) parseBlock() *BlockStmt {
	p.expect(lexer.T_LBRACE)
	block := &BlockStmt{}

	for p.tok.Type != lexer.T_RBRACE && p.tok.Type != lexer.T_EOF {
		stmt := p.parseStmt()
		if stmt != nil {
			block.Stmts = append(block.Stmts, stmt)
		}
		// Skip semicolons and handle end of statements
		if p.tok.Type == lexer.T_SEMICOLON {
			p.nextTok()
		}
	}
	p.expect(lexer.T_RBRACE)

	return block
}

func (p *Parser) parseStmt() Stmt {
	switch p.tok.Type {
	case lexer.T_RETURN:
		return p.parseReturnStmt()
	case lexer.T_IF:
		return p.parseIfStmt()
	case lexer.T_FOR:
		return p.parseForStmt()
	case lexer.T_SWITCH:
		return p.parseSwitchStmt()
	case lexer.T_BREAK:
		p.nextTok()
		stmt := &BreakStmt{}
		if p.tok.Type == lexer.T_IDENT {
			stmt.Label = p.parseIdent()
		}
		return stmt
	case lexer.T_CONTINUE:
		p.nextTok()
		stmt := &ContinueStmt{}
		if p.tok.Type == lexer.T_IDENT {
			stmt.Label = p.parseIdent()
		}
		return stmt
	case lexer.T_GOTO:
		p.nextTok()
		return &GotoStmt{Label: p.parseIdent()}
	case lexer.T_DEFER:
		return p.parseDeferStmt()
	case lexer.T_LBRACE:
		return p.parseBlock()
	case lexer.T_VAR:
		return &DeclStmt{Decl: p.parseVarDecl(nil)}
	case lexer.T_CONST:
		return &DeclStmt{Decl: p.parseConstDecl()}

	default:
		// Check for label: IDENT ':'
		if p.tok.Type == lexer.T_IDENT && p.peek.Type == lexer.T_COLON {
			label := p.parseIdent()
			p.expect(lexer.T_COLON)
			return &LabeledStmt{Label: label, Stmt: p.parseStmt()}
		}

		// Expression or assignment
		expr := p.parseExpr()
		if p.tok.Type == lexer.T_ASSIGN || p.tok.Type == lexer.T_COLON_ASSIGN ||
			p.tok.Type == lexer.T_PLUS_ASSIGN || p.tok.Type == lexer.T_MINUS_ASSIGN ||
			p.tok.Type == lexer.T_STAR_ASSIGN || p.tok.Type == lexer.T_SLASH_ASSIGN ||
			p.tok.Type == lexer.T_AND_ASSIGN || p.tok.Type == lexer.T_OR_ASSIGN {
			op := p.tok.Type
			p.nextTok()
			rhs := []Expr{p.parseExpr()}
			for p.tok.Type == lexer.T_COMMA {
				p.nextTok()
				rhs = append(rhs, p.parseExpr())
			}
			return &AssignStmt{Lhs: []Expr{expr}, Rhs: rhs, Tok: op}
		}
		if p.tok.Type == lexer.T_INC || p.tok.Type == lexer.T_DEC {
			op := p.tok.Type
			p.nextTok()
			return &IncDecStmt{X: expr, Op: op}
		}
		return &ExprStmt{X: expr}
	}
}

func (p *Parser) parseReturnStmt() *ReturnStmt {
	p.expect(lexer.T_RETURN)
	stmt := &ReturnStmt{}

	if p.tok.Type != lexer.T_SEMICOLON && p.tok.Type != lexer.T_RBRACE {
		stmt.Results = append(stmt.Results, p.parseExpr())
		for p.tok.Type == lexer.T_COMMA {
			p.nextTok()
			stmt.Results = append(stmt.Results, p.parseExpr())
		}
	}
	return stmt
}

func (p *Parser) parseIfStmt() *IfStmt {
	p.expect(lexer.T_IF)
	stmt := &IfStmt{}

	// Optional init
	if p.tok.Type == lexer.T_IDENT && p.peek.Type == lexer.T_ASSIGN {
		stmt.Init = &AssignStmt{
			Lhs: []Expr{p.parseIdent()},
			Tok: lexer.T_ASSIGN,
		}
		p.expect(lexer.T_ASSIGN)
		stmt.Init.(*AssignStmt).Rhs = []Expr{p.parseExpr()}
		p.expect(lexer.T_SEMICOLON)
	}

	stmt.Cond = p.parseExpr()
	stmt.Body = p.parseBlock()

	if p.tok.Type == lexer.T_ELSE {
		p.nextTok()
		if p.tok.Type == lexer.T_IF {
			stmt.Else = p.parseIfStmt()
		} else {
			stmt.Else = p.parseBlock()
		}
	}

	return stmt
}

func (p *Parser) parseForStmt() Stmt {
	p.expect(lexer.T_FOR)

	// Check for range
	if p.tok.Type == lexer.T_IDENT && p.peek.Type == lexer.T_COLON_ASSIGN {
		key := p.parseIdent()
		p.nextTok() // skip :=
		var val *Ident
		if p.tok.Type == lexer.T_IDENT {
			val = p.parseIdent()
		}
		p.expect(lexer.T_RANGE)
		x := p.parseExpr()
		body := p.parseBlock()
		return &ForRangeStmt{Key: key, Value: val, X: x, Body: body}
	}

	// Check for "for { }" infinite loop
	if p.tok.Type == lexer.T_LBRACE {
		body := p.parseBlock()
		return &ForStmt{Body: body}
	}

	stmt := &ForStmt{}

	// Parse first part - could be init or condition
	firstExpr := p.parseExpr()

	// Check for "for cond { }" while-style (no semicolons)
	if p.tok.Type == lexer.T_LBRACE {
		stmt.Cond = firstExpr
		stmt.Body = p.parseBlock()
		return stmt
	}

	// C-style: for init; cond; post { }
	stmt.Init = &ExprStmt{X: firstExpr}
	p.expect(lexer.T_SEMICOLON)

	// Cond
	if p.tok.Type != lexer.T_SEMICOLON {
		stmt.Cond = p.parseExpr()
	}
	p.expect(lexer.T_SEMICOLON)

	// Post
	if p.tok.Type != lexer.T_LBRACE {
		stmt.Post = &ExprStmt{X: p.parseExpr()}
	}

	stmt.Body = p.parseBlock()
	return stmt
}

func (p *Parser) parseSwitchStmt() *SwitchStmt {
	p.expect(lexer.T_SWITCH)
	stmt := &SwitchStmt{}

	if p.tok.Type != lexer.T_LBRACE {
		stmt.Tag = p.parseExpr()
	}

	stmt.Body = p.parseBlock()
	return stmt
}

func (p *Parser) parseDeferStmt() *DeferStmt {
	p.expect(lexer.T_DEFER)
	return &DeferStmt{Call: p.parseCallExpr()}
}

func (p *Parser) parseExpr() Expr {
	return p.parseBinaryExpr(0)
}

func (p *Parser) parseBinaryExpr(minPrec int) Expr {
	x := p.parseUnaryExpr()

	loopCount := 0
	for {
		loopCount++
		if loopCount > 100 {
			p.error("parseBinaryExpr infinite loop")
			return x
		}

		op := p.tok.Type
		prec := precedence(op)
		if prec <= minPrec {
			break
		}
		p.nextTok()
		y := p.parseBinaryExpr(prec + 1)
		x = &BinaryExpr{X: x, Op: op, Y: y}
	}

	return x
}

func (p *Parser) parseUnaryExpr() Expr {
	switch p.tok.Type {
	case lexer.T_PLUS, lexer.T_MINUS, lexer.T_NOT, lexer.T_STAR, lexer.T_AND:
		op := p.tok.Type
		p.nextTok()
		return &UnaryExpr{X: p.parseUnaryExpr(), Op: op}
	case lexer.T_LPAREN:
		p.nextTok()
		x := p.parseExpr()
		p.expect(lexer.T_RPAREN)
		return &ParenExpr{X: x}
	case lexer.T_EOF:
		p.error("unexpected EOF in expression")
		return &Ident{Name: "error"}
	default:
		return p.parsePrimaryExpr()
	}
}

func (p *Parser) parsePrimaryExpr() Expr {
	x := p.parseOperand()

	for {
		switch p.tok.Type {
		case lexer.T_DOT:
			p.nextTok()
			x = &SelectorExpr{X: x, Sel: p.parseIdent()}
		case lexer.T_LBRACK:
			p.nextTok()
			idx := p.parseExpr()
			p.expect(lexer.T_RBRACK)
			x = &IndexExpr{X: x, Index: idx}
		case lexer.T_LPAREN:
			x = p.parseCallExprWithFun(x)
		default:
			return x
		}
	}
}

func (p *Parser) parseOperand() Expr {
	switch p.tok.Type {
	case lexer.T_TRUE:
		lit := &BasicLit{Type: lexer.T_TRUE, Value: "true"}
		p.nextTok()
		return lit
	case lexer.T_FALSE:
		lit := &BasicLit{Type: lexer.T_FALSE, Value: "false"}
		p.nextTok()
		return lit
	case lexer.T_NIL:
		lit := &BasicLit{Type: lexer.T_NIL, Value: "nil"}
		p.nextTok()
		return lit
	case lexer.T_IOTA:
		lit := &BasicLit{Type: lexer.T_INT, Value: fmt.Sprintf("%d", p.iotaValue)}
		p.nextTok()
		return lit
	case lexer.T_IDENT:
		return p.parseIdent()
	case lexer.T_INT:
		lit := &BasicLit{Type: lexer.T_INT, Value: p.tok.Literal}
		p.nextTok()
		return lit
	case lexer.T_FLOAT:
		lit := &BasicLit{Type: lexer.T_FLOAT, Value: p.tok.Literal}
		p.nextTok()
		return lit
	case lexer.T_STRING:
		lit := &BasicLit{Type: lexer.T_STRING, Value: strconvQuote(p.tok.Literal)}
		p.nextTok()
		return lit
	case lexer.T_RAWSTRING:
		lit := &BasicLit{Type: lexer.T_RAWSTRING, Value: p.tok.Literal}
		p.nextTok()
		return lit
	case lexer.T_CHAR:
		lit := &BasicLit{Type: lexer.T_CHAR, Value: p.tok.Literal}
		p.nextTok()
		return lit
	case lexer.T_LBRACE:
		return p.parseCompositeLit(nil)
	default:
		p.error(fmt.Sprintf("unexpected operand %v", p.tok.Type))
		p.nextTok()
		return &Ident{Name: "error"}
	}
}

func (p *Parser) parseCallExpr() *CallExpr {
	fun := p.parsePrimaryExpr()
	return p.parseCallExprWithFun(fun)
}

func (p *Parser) parseCallExprWithFun(fun Expr) *CallExpr {
	p.expect(lexer.T_LPAREN)
	call := &CallExpr{Fun: fun}

	for p.tok.Type != lexer.T_RPAREN && p.tok.Type != lexer.T_EOF {
		call.Args = append(call.Args, p.parseExpr())
		if p.tok.Type == lexer.T_COMMA {
			p.nextTok()
		} else {
			break
		}
	}
	p.expect(lexer.T_RPAREN)

	return call
}

func (p *Parser) parseCompositeLit(typ Type) *CompositeLit {
	p.expect(lexer.T_LBRACE)
	lit := &CompositeLit{Type: typ}

	for p.tok.Type != lexer.T_RBRACE && p.tok.Type != lexer.T_EOF {
		lit.Elts = append(lit.Elts, p.parseExpr())
		if p.tok.Type == lexer.T_COMMA {
			p.nextTok()
		} else {
			break
		}
	}
	p.expect(lexer.T_RBRACE)

	return lit
}

func (p *Parser) parseIdent() *Ident {
	tok := p.expect(lexer.T_IDENT)
	return &Ident{Name: tok.Literal}
}

func precedence(op lexer.TokenType) int {
	switch op {
	case lexer.T_OR:
		return 1
	case lexer.T_AND:
		return 2
	case lexer.T_EQ, lexer.T_NE, lexer.T_LT, lexer.T_LE, lexer.T_GT, lexer.T_GE:
		return 3
	case lexer.T_PLUS, lexer.T_MINUS, lexer.T_SHL, lexer.T_SHR:
		return 4
	case lexer.T_STAR, lexer.T_SLASH, lexer.T_PERCENT, lexer.T_XOR, lexer.T_AND_NOT:
		return 5
	default:
		return 0
	}
}

func strconvQuote(s string) string {
	return "\"" + s + "\""
}