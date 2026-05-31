# Z Language

Z (zlang) is a systems programming language designed for operating system development. It uses Go-like syntax while providing low-level control necessary for bare-metal programming.

## Features

- **Go-like Syntax** - Familiar to Go developers
- **No Runtime Dependency** - No garbage collector, scheduler, or hidden runtime
- **Memory Safety by Default** - Safe defaults with explicit escape hatches
- **Deterministic Behavior** - No hidden allocations, no unexpected control flow
- **Bare-metal Ready** - First-class support for interrupts, memory-mapped I/O, and inline assembly

## Quick Start

### Build the Compiler

```bash
make build
```

### Compile a Z Program

```bash
./bin/zc -emit-llvm examples/hello.z
```

This generates `hello.ll` (LLVM IR). Use `llc` to compile to object file:

```bash
llc -filetype=obj hello.ll -o hello.o
```

### Run OS Demo

```bash
make os-demo
make run-os
```

## Project Structure

```
z-lang/
├── cmd/zc/          # Compiler CLI
├── pkg/
│   ├── lexer/       # Lexer (tokenization)
│   ├── parser/      # Parser (AST generation)
│   ├── types/       # Type system
│   ├── sema/        # Semantic analysis (TODO)
│   └── codegen/     # LLVM IR generation
├── stdlib/          # Standard library (asm, unsafe, etc.)
├── examples/        # Example programs
│   ├── hello.z      # Basic hello world
│   ├── arithmetic.z # Arithmetic operations
│   ├── control_flow.z # If/for examples
│   ├── struct_demo.z # Struct usage
│   ├── asm_demo.z   # Inline assembly
│   └── os/          # Bootable OS demo
│       ├── boot.asm # Multiboot entry
│       ├── main.z   # Kernel main
│       └── linker.ld # Linker script
├── LANGUAGE_SPEC.md # Language specification
├── IMPLEMENTATION_PLAN.md # Implementation roadmap
└── Makefile         # Build system
```

## Language Syntax

### Package Declaration

```z
package main
```

### Functions

```z
func add(a, b int32) int32 {
    return a + b
}

//z:export
func KernelMain() {
    // Exported function
}
```

### Variables

```z
var x int32 = 42
var y int32      // zero value: 0
```

### Control Flow

```z
// If-else
if x > 10 {
    // ...
} else {
    // ...
}

// Infinite loop
for {
    break
}

// While-style
for x < 10 {
    x = x + 1
}

// C-style
for i := 0; i < 10; i = i + 1 {
    // ...
}
```

### Inline Assembly (AT&T syntax)

```z
import "asm"

asm.Instr("movl $$42, %eax")
asm.Block(`
    nop
    hlt
`)
```

### Structs

```z
type Point struct {
    X, Y int32
}

//z:packed
type Header struct {
    Flags uint8
    Size  uint16
}
```

## Compiler Directives

| Directive | Purpose |
|-----------|---------|
| `//z:export` | Export symbol for linker |
| `//z:inline` | Force inline function |
| `//z:noalloc` | Prohibit allocations |
| `//z:interrupt` | Mark as interrupt handler |
| `//z:packed` | Pack struct (no padding) |
| `//z:align(N)` | Align variable/type |
| `//z:volatile` | Mark as volatile |

## Requirements

- Go 1.20+ (for building compiler)
- LLVM/llc (for compiling IR)
- ld.lld (for linking)
- nasm (for OS demo boot code)
- QEMU (for running OS demo)

## Current Status

**MVP Complete**:
- Lexer: 90%
- Parser: 85%
- Codegen: 70%
- OS Demo: Working

**TODO**:
- Semantic analysis (type checking)
- Better error messages
- Optimization passes
- Standard library implementation
- Full interrupt support

## License

MIT License - See LICENSE file.

## Documentation

Full language specification: [LANGUAGE_SPEC.md](LANGUAGE_SPEC.md)

Implementation roadmap: [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)