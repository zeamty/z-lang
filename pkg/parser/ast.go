package parser

import "github.com/zeamty/z-lang/pkg/lexer"

// Node represents an AST node
type Node interface {
	node()
}

// Expression nodes
type Expr interface {
	Node
	exprNode()
}

// Statement nodes
type Stmt interface {
	Node
	stmtNode()
}

// Declaration nodes
type Decl interface {
	Node
	declNode()
}

// === Base implementations ===

func (File) node()        {}
func (PackageDecl) node() {}
func (ImportDecl) node()  {}
func (ImportBlock) node() {}
func (ConstDecl) node()   {}
func (ConstBlock) node()  {}
func (VarDecl) node()     {}
func (VarBlock) node()    {}
func (FuncDecl) node()    {}
func (TypeDecl) node()    {}

func (PackageDecl) declNode() {}
func (ImportDecl) declNode()  {}
func (ImportBlock) declNode() {}
func (ConstDecl) declNode()   {}
func (ConstBlock) declNode()  {}
func (VarBlock) declNode()    {}
func (VarDecl) declNode()     {}
func (FuncDecl) declNode()    {}
func (TypeDecl) declNode()    {}

// Statement node() implementations
func (BlockStmt) node()      {}
func (ReturnStmt) node()     {}
func (IfStmt) node()         {}
func (ForStmt) node()        {}
func (ForRangeStmt) node()   {}
func (SwitchStmt) node()     {}
func (CaseStmt) node()       {}
func (BreakStmt) node()      {}
func (ContinueStmt) node()   {}
func (GotoStmt) node()       {}
func (DeferStmt) node()      {}
func (AssignStmt) node()     {}
func (ExprStmt) node()       {}
func (DeclStmt) node()       {}
func (EmptyStmt) node()      {}
func (IncDecStmt) node()     {}
func (LabeledStmt) node()    {}

func (BlockStmt) stmtNode()      {}
func (ReturnStmt) stmtNode()     {}
func (IfStmt) stmtNode()         {}
func (ForStmt) stmtNode()        {}
func (ForRangeStmt) stmtNode()   {}
func (SwitchStmt) stmtNode()     {}
func (CaseStmt) stmtNode()       {}
func (BreakStmt) stmtNode()      {}
func (ContinueStmt) stmtNode()   {}
func (GotoStmt) stmtNode()       {}
func (DeferStmt) stmtNode()      {}
func (AssignStmt) stmtNode()     {}
func (ExprStmt) stmtNode()       {}
func (DeclStmt) stmtNode()       {}
func (EmptyStmt) stmtNode()      {}
func (IncDecStmt) stmtNode()     {}
func (LabeledStmt) stmtNode()    {}

// Expression node() implementations
func (Ident) node()          {}
func (BasicLit) node()       {}
func (BinaryExpr) node()     {}
func (UnaryExpr) node()      {}
func (ParenExpr) node()      {}
func (CallExpr) node()       {}
func (SelectorExpr) node()   {}
func (IndexExpr) node()      {}
func (SliceExpr) node()      {}
func (CompositeLit) node()   {}
func (StarExpr) node()       {}
func (AddrExpr) node()       {}
func (CastExpr) node()       {}
func (FuncLit) node()        {}

func (Ident) exprNode()          {}
func (BasicLit) exprNode()       {}
func (BinaryExpr) exprNode()     {}
func (UnaryExpr) exprNode()      {}
func (ParenExpr) exprNode()      {}
func (CallExpr) exprNode()       {}
func (SelectorExpr) exprNode()   {}
func (IndexExpr) exprNode()      {}
func (SliceExpr) exprNode()      {}
func (CompositeLit) exprNode()   {}
func (StarExpr) exprNode()       {}
func (AddrExpr) exprNode()       {}
func (CastExpr) exprNode()       {}
func (FuncLit) exprNode()        {}

// === File ===

type File struct {
	Pkg      *PackageDecl
	Imports  []*ImportDecl
	Decls    []Decl
	Directives []string // //z: directives
}

// === Package ===

type PackageDecl struct {
	Name *Ident
}

// === Import ===

type ImportDecl struct {
	Name   *Ident    // alias, optional
	Path   *BasicLit // string literal
}

type ImportBlock struct {
	Imports []*ImportDecl
}

// === Type declarations ===

type TypeDecl struct {
	Name   *Ident
	Type   Type
}

// === Type expressions ===

type Type interface {
	Node
	typeNode()
}

// Type node() implementations
func (ArrayType) node()    {}
func (SliceType) node()    {}
func (PointerType) node()  {}
func (StructType) node()   {}
func (FuncType) node()     {}
func (InterfaceType) node() {}

func (Ident) typeNode()        {}
func (ArrayType) typeNode()   {}
func (SliceType) typeNode()   {}
func (PointerType) typeNode() {}
func (StructType) typeNode()  {}
func (FuncType) typeNode()    {}
func (InterfaceType) typeNode() {}

type ArrayType struct {
	Len  Expr
	Elt  Type
}

type SliceType struct {
	Elt Type
}

type PointerType struct {
	Base Type
}

type StructType struct {
	Fields []*Field
}

type Field struct {
	Name     *Ident
	Type     Type
	Directive string // //z:align etc.
}

type FuncType struct {
	Params  []*Field
	Results []*Field
}

type InterfaceType struct {
	Methods []*Field
}

// === Const ===

type ConstDecl struct {
	Name  *Ident
	Type  Type
	Value Expr
}

type ConstBlock struct {
	Consts []*ConstDecl
}

// === Var ===

type VarDecl struct {
	Name      *Ident
	Type      Type
	Value     Expr
	Directive string // //z:align, //z:volatile
}

type VarBlock struct {
	Vars []*VarDecl
}

// === Func ===

type FuncDecl struct {
	Name       *Ident
	Recv       *Field   // receiver, optional
	Type       *FuncType
	Body       *BlockStmt
	Directives []string // //z:export, //z:interrupt, etc.
}

// === Statements ===

type BlockStmt struct {
	Stmts []Stmt
}

type ReturnStmt struct {
	Results []Expr
}

type IfStmt struct {
	Init Stmt
	Cond Expr
	Body *BlockStmt
	Else Stmt
}

type ForStmt struct {
	Init Stmt
	Cond Expr
	Post Stmt
	Body *BlockStmt
}

type ForRangeStmt struct {
	Key   *Ident
	Value *Ident
	X     Expr
	Body  *BlockStmt
}

type SwitchStmt struct {
	Init Stmt
	Tag  Expr
	Body *BlockStmt
}

type CaseStmt struct {
	List []Expr
	Body *BlockStmt
}

type BreakStmt struct {
	Label *Ident
}

type ContinueStmt struct {
	Label *Ident
}

type GotoStmt struct {
	Label *Ident
}

type DeferStmt struct {
	Call *CallExpr
}

type AssignStmt struct {
	Lhs    []Expr
	Rhs    []Expr
	Tok    lexer.TokenType // =, :=, +=, etc.
}

type ExprStmt struct {
	X Expr
}

type DeclStmt struct {
	Decl Decl
}

type EmptyStmt struct{}

type IncDecStmt struct {
	X  Expr
	Op lexer.TokenType
}

type LabeledStmt struct {
	Label *Ident
	Stmt  Stmt
}

// === Expressions ===

type Ident struct {
	Name string
}

type BasicLit struct {
	Type  lexer.TokenType
	Value string
}

type BinaryExpr struct {
	X  Expr
	Op lexer.TokenType
	Y  Expr
}

type UnaryExpr struct {
	X  Expr
	Op lexer.TokenType
}

type ParenExpr struct {
	X Expr
}

type CallExpr struct {
	Fun  Expr
	Args []Expr
}

type SelectorExpr struct {
	X   Expr
	Sel *Ident
}

type IndexExpr struct {
	X     Expr
	Index Expr
}

type SliceExpr struct {
	X    Expr
	Low  Expr
	High Expr
}

type CompositeLit struct {
	Type Type
	Elts []Expr
}

type StarExpr struct {
	X Expr
}

type AddrExpr struct {
	X Expr
}

type CastExpr struct {
	X    Expr
	Type Type
}

type FuncLit struct {
	Type *FuncType
	Body *BlockStmt
}