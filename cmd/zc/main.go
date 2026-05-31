package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeamty/z-lang/pkg/codegen"
	"github.com/zeamty/z-lang/pkg/lexer"
	"github.com/zeamty/z-lang/pkg/parser"
)

func main() {
	var (
		output   string
		emitLLVM bool
		help     bool
	)

	flag.StringVar(&output, "o", "", "output file")
	flag.BoolVar(&emitLLVM, "emit-llvm", false, "emit LLVM IR instead of object file")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help || flag.NArg() == 0 {
		fmt.Println("Z Language Compiler")
		fmt.Println("Usage: zc [options] <file.z>")
		flag.PrintDefaults()
		os.Exit(0)
	}

	input := flag.Arg(0)
	if !strings.HasSuffix(input, ".z") {
		fmt.Fprintf(os.Stderr, "error: input file must have .z extension\n")
		os.Exit(1)
	}

	// Read source
	src, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Compile
	result, err := compile(input, string(src), emitLLVM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Determine output file
	if output == "" {
		if emitLLVM {
			output = strings.Replace(input, ".z", ".ll", 1)
		} else {
			output = strings.Replace(input, ".z", ".o", 1)
		}
	}

	// Write output
	err = os.WriteFile(output, []byte(result), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("compiled %s -> %s\n", input, output)
}

func compile(filename, src string, emitLLVM bool) (string, error) {
	// Lexer
	l := lexer.New(src)

	// Parser
	p := parser.New(l)
	file := p.ParseFile()

	if p.HasErrors() {
		return "", fmt.Errorf("parse errors:\n%s", strings.Join(p.Errors(), "\n"))
	}

	// Debug output
	if os.Getenv("Z_DEBUG") != "" {
		debugAST(file)
	}

	// Codegen
	g := codegen.New()
	ir := g.Generate(file)

	if emitLLVM {
		return ir, nil
	}

	// Use llc to compile LLVM IR to object file
	// This is a placeholder - actual compilation would use llc
	return ir, nil
}

func debugAST(file *parser.File) {
	fmt.Println("=== AST Debug ===")
	fmt.Printf("Package: %s\n", file.Pkg.Name.Name)
	for _, imp := range file.Imports {
		fmt.Printf("Import: %s\n", imp.Path.Value)
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *parser.FuncDecl:
			fmt.Printf("Func: %s\n", d.Name.Name)
		case *parser.VarDecl:
			fmt.Printf("Var: %s\n", d.Name.Name)
		case *parser.TypeDecl:
			fmt.Printf("Type: %s\n", d.Name.Name)
		}
	}
	fmt.Println("=================")
}

func compileToObject(ir, output string) error {
	// Write temporary .ll file
	tmpLL := filepath.Join(os.TempDir(), "z_temp.ll")
	err := os.WriteFile(tmpLL, []byte(ir), 0644)
	if err != nil {
		return err
	}

	// Run llc
	// TODO: implement actual compilation pipeline
	return nil
}