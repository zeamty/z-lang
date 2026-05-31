package main

import (
	"fmt"

	"github.com/zeamty/z-lang/pkg/lexer"
)

func main() {
	src := `asm.Instr("mov byte [eax], 90")`
	l := lexer.New(src)

	fmt.Println("Tokens:")
	for i := 0; i < 20; i++ {
		tok := l.NextToken()
		fmt.Printf("Token %d: type=%d lit=%q\n", i, tok.Type, tok.Literal)
		if tok.Type == lexer.T_EOF {
			break
		}
	}
}