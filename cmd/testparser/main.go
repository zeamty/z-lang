package main

import (
	"fmt"

	"github.com/zeamty/z-lang/pkg/lexer"
	"github.com/zeamty/z-lang/pkg/parser"
)

func main() {
	src := `package main

func test() {
    for {
        asm.Instr("hlt")
    }
}`

	l := lexer.New(src)

	fmt.Println("Tokens:")
	count := 0
	for {
		tok := l.NextToken()
		fmt.Printf("Token %d: type=%d lit=%q\n", count, tok.Type, tok.Literal)
		if tok.Type == lexer.T_EOF {
			break
		}
		count++
		if count > 30 {
			break
		}
	}

	fmt.Println("\nParsing...")
	l2 := lexer.New(src)
	p := parser.New(l2)
	_ = p.ParseFile()

	if p.HasErrors() {
		fmt.Println("Errors:", p.Errors())
	} else {
		fmt.Println("Success!")
	}
}