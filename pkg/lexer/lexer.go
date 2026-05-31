package lexer

import "unicode"

// Lexer scans Z source code and produces tokens
type Lexer struct {
	input    string
	pos      int  // current position
	readPos  int  // next reading position
	ch       byte // current char
	line     int
	column   int
}

// New creates a new Lexer
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads next character
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.column++
}

// peekChar looks ahead without advancing
func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// NextToken returns next token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	// Check for directive first
	if l.ch == '/' && l.peekChar() == '/' {
		l.readChar()
		l.readChar()
		if l.ch == 'z' && l.peekChar() == ':' {
			// This is a directive
			directive := "//z:"
			l.readChar() // skip 'z'
			l.readChar() // skip ':'
			// Read the directive name
			for l.ch != '\n' && l.ch != 0 && l.ch != ' ' && l.ch != '\t' {
				directive += string(l.ch)
				l.readChar()
			}
			// Skip rest of line
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			return Token{Type: T_DIRECTIVE, Literal: directive, Line: l.line, Column: l.column}
		}
		// Regular line comment - skip it
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		return l.NextToken()
	}

	// Check for block comment
	if l.ch == '/' && l.peekChar() == '*' {
		l.readChar()
		l.readChar()
		for {
			if l.ch == '*' && l.peekChar() == '/' {
				l.readChar()
				l.readChar()
				break
			}
			if l.ch == '\n' {
				l.line++
				l.column = 0
			}
			if l.ch == 0 {
				break
			}
			l.readChar()
		}
		return l.NextToken()
	}

	switch l.ch {
	case 0:
		tok = Token{Type: T_EOF, Line: l.line, Column: l.column}

	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: T_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_ASSIGN, Literal: "=", Line: l.line, Column: l.column}
		}

	case '+':
		if l.peekChar() == '+' {
			l.readChar()
			tok = Token{Type: T_INC, Literal: "++", Line: l.line, Column: l.column - 2}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_PLUS_ASSIGN, Literal: "+=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_PLUS, Literal: "+", Line: l.line, Column: l.column}
		}

	case '-':
		if l.peekChar() == '-' {
			l.readChar()
			tok = Token{Type: T_DEC, Literal: "--", Line: l.line, Column: l.column - 2}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_MINUS_ASSIGN, Literal: "-=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_MINUS, Literal: "-", Line: l.line, Column: l.column}
		}

	case '*':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_STAR_ASSIGN, Literal: "*=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_STAR, Literal: "*", Line: l.line, Column: l.column}
		}

	case '/':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_SLASH_ASSIGN, Literal: "/=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_SLASH, Literal: "/", Line: l.line, Column: l.column}
		}

	case '%':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_PERCENT_ASSIGN, Literal: "%=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_PERCENT, Literal: "%", Line: l.line, Column: l.column}
		}

	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_NE, Literal: "!=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_NOT, Literal: "!", Line: l.line, Column: l.column}
		}

	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_LE, Literal: "<=", Line: l.line, Column: l.column - 2}
		} else if l.peekChar() == '<' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = Token{Type: T_SHL_ASSIGN, Literal: "<<=", Line: l.line, Column: l.column - 3}
			} else {
				tok = Token{Type: T_SHL, Literal: "<<", Line: l.line, Column: l.column - 2}
			}
		} else if l.peekChar() == '-' {
			l.readChar()
			tok = Token{Type: T_ARROW, Literal: "<-", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_LT, Literal: "<", Line: l.line, Column: l.column}
		}

	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_GE, Literal: ">=", Line: l.line, Column: l.column - 2}
		} else if l.peekChar() == '>' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = Token{Type: T_SHR_ASSIGN, Literal: ">>=", Line: l.line, Column: l.column - 3}
			} else {
				tok = Token{Type: T_SHR, Literal: ">>", Line: l.line, Column: l.column - 2}
			}
		} else {
			tok = Token{Type: T_GT, Literal: ">", Line: l.line, Column: l.column}
		}

	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = Token{Type: T_AND, Literal: "&&", Line: l.line, Column: l.column - 2}
		} else if l.peekChar() == '^' {
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = Token{Type: T_AND_NOT_ASSIGN, Literal: "&^=", Line: l.line, Column: l.column - 3}
			} else {
				tok = Token{Type: T_AND_NOT, Literal: "&^", Line: l.line, Column: l.column - 2}
			}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_AND_ASSIGN, Literal: "&=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_AND, Literal: "&", Line: l.line, Column: l.column}
		}

	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = Token{Type: T_OR, Literal: "||", Line: l.line, Column: l.column - 2}
		} else if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_OR_ASSIGN, Literal: "|=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_OR, Literal: "|", Line: l.line, Column: l.column}
		}

	case '^':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_XOR_ASSIGN, Literal: "^=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_XOR, Literal: "^", Line: l.line, Column: l.column}
		}

	case '(':
		tok = Token{Type: T_LPAREN, Literal: "(", Line: l.line, Column: l.column}
	case ')':
		tok = Token{Type: T_RPAREN, Literal: ")", Line: l.line, Column: l.column}
	case '[':
		tok = Token{Type: T_LBRACK, Literal: "[", Line: l.line, Column: l.column}
	case ']':
		tok = Token{Type: T_RBRACK, Literal: "]", Line: l.line, Column: l.column}
	case '{':
		tok = Token{Type: T_LBRACE, Literal: "{", Line: l.line, Column: l.column}
	case '}':
		tok = Token{Type: T_RBRACE, Literal: "}", Line: l.line, Column: l.column}
	case ',':
		tok = Token{Type: T_COMMA, Literal: ",", Line: l.line, Column: l.column}
	case ';':
		tok = Token{Type: T_SEMICOLON, Literal: ";", Line: l.line, Column: l.column}
	case ':':
		if l.peekChar() == '=' {
			l.readChar()
			tok = Token{Type: T_COLON_ASSIGN, Literal: ":=", Line: l.line, Column: l.column - 2}
		} else {
			tok = Token{Type: T_COLON, Literal: ":", Line: l.line, Column: l.column}
		}
	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			if l.peekChar() == '.' {
				l.readChar()
				tok = Token{Type: T_ELLIPSIS, Literal: "...", Line: l.line, Column: l.column - 3}
			}
		} else {
			tok = Token{Type: T_DOT, Literal: ".", Line: l.line, Column: l.column}
		}

	case '"':
		tok = Token{Type: T_STRING, Literal: l.readString('"'), Line: l.line, Column: l.column}
		return tok

	case '`':
		tok = Token{Type: T_RAWSTRING, Literal: l.readRawString(), Line: l.line, Column: l.column}
		return tok

	case '\'':
		tok = Token{Type: T_CHAR, Literal: l.readCharLiteral(), Line: l.line, Column: l.column}
		return tok

	default:
		if isLetter(l.ch) {
			ident := l.readIdent()
			tok = Token{Type: LookupIdent(ident), Literal: ident, Line: l.line, Column: l.column - len(ident)}
			return tok
		} else if isDigit(l.ch) {
			num := l.readNumber()
			tok = Token{Type: T_INT, Literal: num, Line: l.line, Column: l.column - len(num)}
			return tok
		} else {
			tok = Token{Type: T_ILLEGAL, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	if l.ch == '/' {
		if l.peekChar() == '/' {
			// Line comment
			l.readChar()
			l.readChar()
			// Check for directive
			if l.ch == 'z' && l.peekChar() == ':' {
				l.readIdent() // read the directive
			}
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			l.skipWhitespace()
			l.skipComment()
		} else if l.peekChar() == '*' {
			// Block comment
			l.readChar()
			l.readChar()
			for {
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar()
					l.readChar()
					break
				}
				if l.ch == '\n' {
					l.line++
					l.column = 0
				}
				if l.ch == 0 {
					break
				}
				l.readChar()
			}
			l.skipWhitespace()
			l.skipComment()
		}
	}
}

func (l *Lexer) readIdent() string {
	pos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) readNumber() string {
	pos := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	// Handle hex, octal, binary
	if l.ch == 'x' || l.ch == 'X' {
		l.readChar()
		for isHexDigit(l.ch) {
			l.readChar()
		}
	} else if l.ch == 'o' || l.ch == 'O' {
		l.readChar()
		for isOctalDigit(l.ch) {
			l.readChar()
		}
	} else if l.ch == 'b' || l.ch == 'B' {
		l.readChar()
		for isBinaryDigit(l.ch) {
			l.readChar()
		}
	}
	// Handle underscores
	for l.ch == '_' {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	// Handle float
	if l.ch == '.' {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	// Handle exponent
	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) readString(quote byte) string {
	start := l.pos + 1 // skip opening quote
	l.readChar() // move past opening quote
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	str := l.input[start:l.pos] // content without quotes
	l.readChar() // skip closing quote
	return str
}

func (l *Lexer) readRawString() string {
	pos := l.pos
	l.readChar() // skip opening quote
	for l.ch != '`' && l.ch != 0 {
		if l.ch == '\n' {
			l.line++
		}
		l.readChar()
	}
	str := l.input[pos:l.pos]
	l.readChar() // skip closing quote
	return str
}

func (l *Lexer) readCharLiteral() string {
	pos := l.pos
	l.readChar() // skip opening quote
	if l.ch == '\\' {
		l.readChar()
	}
	l.readChar() // read char
	l.readChar() // skip closing quote
	return l.input[pos:l.pos]
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || 'a' <= ch && ch <= 'f' || 'A' <= ch && ch <= 'F'
}

func isOctalDigit(ch byte) bool {
	return '0' <= ch && ch <= '7'
}

func isBinaryDigit(ch byte) bool {
	return ch == '0' || ch == '1'
}