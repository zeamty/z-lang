package lexer

// Token types for Z language
type TokenType int

const (
	// Special tokens
	T_ILLEGAL TokenType = iota
	T_EOF
	T_COMMENT

	// Identifiers and literals
	T_IDENT    // main
	T_INT      // 12345
	T_FLOAT    // 123.45
	T_STRING   // "abc"
	T_RAWSTRING // `abc`
	T_CHAR     // 'a'

	// Operators
	T_PLUS     // +
	T_MINUS    // -
	T_STAR     // *
	T_SLASH    // /
	T_PERCENT  // %

	T_AND      // &
	T_OR       // |
	T_XOR      // ^
	T_SHL      // <<
	T_SHR      // >>
	T_AND_NOT  // &^

	T_ASSIGN   // =
	T_PLUS_ASSIGN  // +=
	T_MINUS_ASSIGN // -=
	T_STAR_ASSIGN  // *=
	T_SLASH_ASSIGN // /=
	T_PERCENT_ASSIGN // %=
	T_AND_ASSIGN     // &=
	T_OR_ASSIGN      // |=
	T_XOR_ASSIGN     // ^=
	T_SHL_ASSIGN     // <<=
	T_SHR_ASSIGN     // >>=
	T_AND_NOT_ASSIGN // &^=

	T_EQ       // ==
	T_NE       // !=
	T_LT       // <
	T_LE       // <=
	T_GT       // >
	T_GE       // >=

	T_NOT      // !

	T_ARROW    // <-

	T_INC      // ++
	T_DEC      // --

	// Delimiters
	T_LPAREN   // (
	T_RPAREN   // )
	T_LBRACK   // [
	T_RBRACK   // ]
	T_LBRACE   // {
	T_RBRACE   // }
	T_COMMA    // ,
	T_SEMICOLON // ;
	T_COLON    // :
	T_COLON_ASSIGN // :=
	T_DOT      // .
	T_ELLIPSIS // ...

	// Keywords
	T_PACKAGE
	T_IMPORT
	T_CONST
	T_VAR
	T_FUNC
	T_RETURN
	T_IF
	T_ELSE
	T_FOR
	T_RANGE
	T_SWITCH
	T_CASE
	T_DEFAULT
	T_BREAK
	T_CONTINUE
	T_GOTO
	T_FALLTHROUGH
	T_DEFER

	T_TYPE
	T_STRUCT
	T_INTERFACE

	T_UNSAFE
	T_VOLATILE

	// Directive prefix
	T_DIRECTIVE // //z:
)

// Keywords map
var Keywords = map[string]TokenType{
	"package":     T_PACKAGE,
	"import":      T_IMPORT,
	"const":       T_CONST,
	"var":         T_VAR,
	"func":        T_FUNC,
	"return":      T_RETURN,
	"if":          T_IF,
	"else":        T_ELSE,
	"for":         T_FOR,
	"range":       T_RANGE,
	"switch":      T_SWITCH,
	"case":        T_CASE,
	"default":     T_DEFAULT,
	"break":       T_BREAK,
	"continue":    T_CONTINUE,
	"goto":        T_GOTO,
	"fallthrough": T_FALLTHROUGH,
	"defer":       T_DEFER,
	"type":        T_TYPE,
	"struct":      T_STRUCT,
	"interface":   T_INTERFACE,
	"unsafe":      T_UNSAFE,
	"volatile":    T_VOLATILE,
}

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) String() string {
	return t.Literal
}

// LookupIdent checks if identifier is a keyword
func LookupIdent(ident string) TokenType {
	if tok, ok := Keywords[ident]; ok {
		return tok
	}
	return T_IDENT
}