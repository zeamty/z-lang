package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/zeamty/z-lang/pkg/codegen"
	"github.com/zeamty/z-lang/pkg/lexer"
	"github.com/zeamty/z-lang/pkg/parser"
	"github.com/zeamty/z-lang/pkg/sema"
)

func main() {
	var (
		output    string
		emitLLVM  bool
		help      bool
		buildTags string
	)

	flag.StringVar(&output, "o", "", "output file")
	flag.BoolVar(&emitLLVM, "emit-llvm", false, "emit LLVM IR instead of object file")
	flag.StringVar(&buildTags, "tags", "", "comma-separated build tags (e.g. amd64,linux)")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help || flag.NArg() == 0 {
		fmt.Println("Z Language Compiler")
		fmt.Println("Usage: zc [options] <file.z> [file2.z ...]")
		flag.PrintDefaults()
		os.Exit(0)
	}

	inputs := flag.Args()
	activeTags := parseBuildTags(buildTags)

	var allFiles []*parser.File
	for _, input := range inputs {
		if !strings.HasSuffix(input, ".z") {
			fmt.Fprintf(os.Stderr, "warning: skipping non-.z file: %s\n", input)
			continue
		}

		src, err := os.ReadFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		if !checkBuildTags(string(src), activeTags) {
			continue
		}

		l := lexer.New(string(src))
		p := parser.New(l)
		file := p.ParseFile()

		if p.HasErrors() {
			fmt.Fprintf(os.Stderr, "parse errors in %s:\n%s\n", input,
				strings.Join(p.Errors(), "\n"))
			os.Exit(1)
		}
		allFiles = append(allFiles, file)
	}

	if len(allFiles) == 0 {
		fmt.Fprintf(os.Stderr, "error: no valid input files\n")
		os.Exit(1)
	}

	merged := mergeFiles(allFiles)

	// Semantic analysis
	analyzer := sema.NewAnalyzer()
	errs := analyzer.Analyze(merged)
	if errs.HasErrors() {
		fmt.Fprintf(os.Stderr, "semantic errors:\n%s\n", errs.String())
		os.Exit(1)
	}

	// Codegen
	g := codegen.New()
	ir := g.Generate(merged)

	// Determine output
	if output == "" {
		base := strings.TrimSuffix(inputs[0], ".z")
		if emitLLVM {
			output = base + ".ll"
		} else {
			output = base + ".o"
		}
	}

	err := os.WriteFile(output, []byte(ir), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("compiled %d files -> %s\n", len(allFiles), output)
}

func parseBuildTags(tagStr string) map[string]bool {
	tags := make(map[string]bool)
	if tagStr == "" {
		return tags
	}
	for _, t := range strings.Split(tagStr, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags[t] = true
		}
	}
	return tags
}

func checkBuildTags(src string, activeTags map[string]bool) bool {
	if len(activeTags) == 0 {
		return true
	}
	lines := strings.Split(src, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//z:build") {
			tagSpec := strings.TrimPrefix(line, "//z:build")
			tagSpec = strings.TrimSpace(tagSpec)

			if strings.HasPrefix(tagSpec, "!") {
				negTag := strings.TrimPrefix(tagSpec, "!")
				if activeTags[negTag] {
					return false
				}
			} else {
				if !activeTags[tagSpec] {
					return false
				}
			}
		}
	}
	return true
}

func mergeFiles(files []*parser.File) *parser.File {
	if len(files) == 1 {
		return files[0]
	}

	merged := &parser.File{}
	for _, f := range files {
		if merged.Pkg == nil {
			merged.Pkg = f.Pkg
		}
		merged.Imports = append(merged.Imports, f.Imports...)
		merged.Decls = append(merged.Decls, f.Decls...)
		merged.Directives = append(merged.Directives, f.Directives...)
	}
	return merged
}