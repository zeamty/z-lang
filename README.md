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
│   ├── sema/        # Semantic analysis
│   └── codegen/     # LLVM IR generation
├── stdlib/          # Standard library
│   ├── asm/         # Inline assembly intrinsics
│   ├── unsafe/      # Unsafe operations
│   ├── atomic/      # Atomic operations
│   ├── mem/         # Memory intrinsics
│   └── errors/      # Error interface
├── examples/        # Example programs
│   ├── hello.z      # Basic hello world
│   ├── arithmetic.z # Arithmetic operations
│   ├── control_flow.z # If/for examples
│   ├── struct_demo.z # Struct usage
│   ├── asm_demo.z   # Inline assembly
│   └── os/          # Bootable OS demo
│       ├── boot.asm # Multiboot entry
│       ├── main.z   # Kernel main (VGA Hello Zlang!)
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
x := 42          // short declaration
```

### Constants

```z
const MAX = 100
const (
    A = iota  // 0
    B         // 1
    C         // 2
)
```

### Pointers & Casts

```z
var p *byte
p = (*byte)(unsafe.Pointer(0xB8000))
*p = 72  // write to memory

// Address-of operator
var x int
var ptr *int
ptr = &x  // take address of x
```

### Arrays & Slices

```z
var arr [5]int
arr[0] = 10  // array indexing

// Slice expression
var sl []int
sl = arr[1:4]  // slice from index 1 to 4

// Range loop
for i, v := range arr {
    // i is index, v is element value
}
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

// Switch
switch x {
case 1:
    // ...
case 2:
    // ...
default:
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

**P0-P3 Features Complete**:
- Lexer: keywords, operators, literals, directives, identifiers
- Parser: full expression/statement grammar, :=, var/const blocks, iota, named returns, labels, import blocks, type casts, pointer dereference, address-of (`&x`), slice expressions (`arr[low:high]`)
- Semantic Analysis: scope resolution, type checking, directive validation, builtin type recognition
- Codegen: Emitter + TypeResolver, functions, control flow (if/for/switch/defer/goto), structs, pointers, type casts, inline assembly, builtin intrinsics
- **New**: `AddrExpr` (`&x`), `SliceExpr` (`arr[low:high]`), `ForRangeStmt` (`for i, v := range arr`), array indexing LHS (`arr[i] = value`), CompositeLit
- **Latest**: struct field access (`p.X`), keyed struct literals (`Point{X: 1, Y: 2}`), labels and `goto`
- Standard Library: asm, unsafe, atomic, mem, errors packages
- CLI: multi-file input, build tags (`//z:build`), `-emit-llvm` flag
- OS Demo: VGA text output via pure Z code (no `asm.Instr`)

## License

MIT License - See LICENSE file.

## Documentation

Full language specification: [LANGUAGE_SPEC.md](LANGUAGE_SPEC.md)

Implementation roadmap: [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)