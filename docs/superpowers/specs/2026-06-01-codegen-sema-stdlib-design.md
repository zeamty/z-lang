# Z Language Compiler - Codegen, Sema, Stdlib Design

**Date:** 2026-06-01
**Status:** Approved

## Overview

Complete P0-P3 missing features for the Z language compiler in 3 module batches:
1. **Codegen** — all LLVM IR generation features (asm constraints, interrupt wrappers, switch/defer/goto, packed/volatile/align, iota, named returns)
2. **Sema** — semantic analysis (type checking, scope resolution, directive validation)
3. **Stdlib** — standard library packages (asm, unsafe, atomic, mem, errors)

## Architecture Principle

Zero CGo, pure Go, text-based LLVM IR generation. No external LLVM bindings.

## Module 1: Codegen Refactoring

### File Split
```
pkg/codegen/
├── codegen.go      # Generator struct + Generate() entry point
├── emit.go         # IR emission utilities (typed helpers)
├── asm.go          # Inline assembly with operand constraints
├── stmt.go         # switch, defer, goto, labeled break/continue
├── expr.go         # expression codegen (extracted from current)
├── types.go        # LLVM type mapping (extracted from current)
└── directives.go   # interrupt, packed, volatile, align, inline, noalloc
```

### Feature Details

**asm constraints (P0):**
- Parse `asm.In(constraint, &var)`, `asm.Out(constraint, &var)`, `asm.InOut(constraint, &inVar, &outVar)`, `asm.Clobbers(regs...)` from AST call args
- Map register constraint chars to LLVM constraint strings: `"a"` → `"{ax}"`, `"=a"` → `"={ax}"`, `"r"` → `"r"`
- Generate: `call <ret> asm sideeffect "<template>", "<constraints>"(<args>)`
- `$` in AT&T syntax must be escaped to `$$` in LLVM

**interrupt handlers (P1):**
- Detect `//z:interrupt` directive on functions
- Generate x86-64 interrupt calling convention: `x86_intrcc`
- Function prologue: save all GPRs to stack
- Function epilogue: restore all GPRs, `iretq`

**switch (P1):**
- Integer switches → LLVM `switch` instruction with default label
- Tagless switches → chain of if-else blocks

**defer (P1):**
- Collect deferred calls in function scope
- At each return point + function end, emit deferred calls in LIFO order
- Simple approach: track defer list, emit at known exit points

**goto + labeled break/continue (P1):**
- Labels map to LLVM basic blocks
- `goto <label>` → `br label %<label>`
- `break <label>` → `br label %<label>.end`
- `continue <label>` → `br label %<label>.continue`

**packed struct (P2):**
- Detect `//z:packed` directive on struct type
- Emit LLVM packed struct: `<{ i8, i16 }>` instead of `{ i8, i16 }`

**volatile (P2):**
- Detect `//z:volatile` directive on variable
- Emit `load volatile` / `store volatile` for accesses

**align (P2):**
- Parse `//z:align(N)` directive value
- Emit `align N` on global variable or alloca

**inline (P2):**
- Detect `//z:inline`
- Emit `alwaysinline` attribute on function

**noalloc (P2):**
- Detect `//z:noalloc`
- Emit `noalloc` attribute on function (custom attribute for validation)

**iota (P2):**
- In parser: track iota value in const block, increment per const
- In codegen: iota resolved to integer constant at compile time

**named return values (P2):**
- In parser: track named results in FuncType
- In codegen: alloca named return values in function entry, return loads them

**multi-file compilation (P3):**
- Compiler takes multiple `.z` files
- Parse each independently, merge ASTs by package
- Emit single LLVM IR module

**build tags (P3):**
- Parse `//z:build <tag>` and `//z:build !tag`
- Filter files by active build tags before parsing

## Module 2: Semantic Analysis

```
pkg/sema/
├── sema.go         # Analyzer entry point, walks AST
├── scope.go        # Scope tree (package → file → function → block)
├── types.go        # Type checking expressions and statements
├── resolve.go      # Name resolution
├── directives.go   # Directive validation (noalloc, interrupt, etc.)
├── errors.go       # Diagnostic error messages
```

### Features
- **Scope resolution:** nested scopes (block, function, file, package)
- **Type checking:** binary ops, assignments, function calls, return types
- **Directive validation:** interrupt funcs must have correct signature, noalloc funcs must not allocate
- **Interface satisfaction:** check struct methods match interface method sets
- **Export rules:** capitalized = exported, detect unexported foreign refs

## Module 3: Standard Library

```
stdlib/
├── asm/asm.z       # asm.Instr, asm.Block, asm.In, asm.Out, asm.InOut, asm.Clobbers
├── unsafe/unsafe.z # Pointer, Sizeof, Offsetof, Alignof, Slice
├── atomic/atomic.z # Load*, Store*, Add*, Swap*, CAS*
├── mem/mem.z       # Memcpy, Memset, Memmove
├── errors/errors.z # error interface, New()
└── bits/bits.z     # bit manipulation helpers
```

### Strategy
- asm package functions are compiler intrinsics — codegen directly handles calls into asm.*
- unsafe.Sizeof/Offsetof/Alignof are compiler intrinsics — computed at compile time
- atomic operations map to LLVM atomicrmw or cmpxchg instructions
- mem functions map to LLVM llvm.memcpy/llvm.memset intrinsics
- errors package is pure Z code (no runtime needed)

## Implementation Order

1. Codegen file split + emit.go foundation
2. asm constraints (P0)
3. switch/defer/goto/labels codegen (P1)
4. interrupt wrappers (P1)
5. packed/volatile/align codegen (P2)
6. iota + named returns (P2)
7. Multi-file + build tags (P3)
8. Semantic analysis module (P1-P2)
9. Standard library packages (P2)