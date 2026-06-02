# Z Language Compiler & OS Demo Implementation Plan

## Overview

Using Go + LLVM to implement a compiler for Z language, then write a QEMU-bootable OS demo.

---

## Phase 1: Compiler Infrastructure (Week 1-2)

### 1.1 Project Structure
```
z-lang/
├── cmd/
│   └── zc/             # Compiler CLI
├── pkg/
│   ├── lexer/          # Lexer implementation
│   ├── parser/         # Parser & AST
│   ├── types/          # Type system
│   ├── sema/           # Semantic analysis
│   ├── ir/             # IR generation
│   └── codegen/        # LLVM IR generation
├── runtime/
│   └── zrt/            # Minimal runtime
└── stdlib/
    ├── asm/            # Inline assembly support
    ├── unsafe/         # Unsafe operations
    └── ...
```

### 1.2 Lexer (Go)
- Token types: keywords, operators, literals, identifiers
- Handle Z-specific tokens: `//z:` directives
- Support hex (`0x`), octal (`0o`), binary (`0b`) literals
- Raw strings and rune literals

### 1.3 Parser & AST (Go)
- Recursive descent parser
- AST nodes for:
  - Package declarations
  - Import statements
  - Type declarations (struct, interface, type aliases)
  - Function declarations with `//z:interrupt`, `//z:export`, `//z:inline`
  - Variable/constant declarations
  - Control flow (if, for, switch, goto, defer)
  - Inline assembly blocks (`asm.Instr`, `asm.Block`)

### 1.4 Type System (Go)
- Basic types: int8-64, uint8-64, float32-64, bool, string
- Pointer types and `unsafe.Pointer`
- Array, slice, struct types
- Interface types (monomorphization only)
- Type checking with no implicit conversions

### 1.5 Semantic Analysis (Go)
- Scope resolution
- Type checking
- Directive validation (`//z:noalloc`, `//z:interrupt`)
- Interface satisfaction check
- Escape analysis for allocations

---

## Phase 2: LLVM IR Generation (Week 3-4)

### 2.1 LLVM Go Bindings
Use `tinygo-org/go-llvm` or generate LLVM IR textually:
```go
import "tinygo.org/x/go-llvm"
```

### 2.2 Type Mapping
| Z Type | LLVM Type |
|--------|-----------|
| int8 | i8 |
| int16 | i16 |
| int32 | i32 |
| int64 | i64 |
| int | i64 (64-bit) |
| float32 | float |
| float64 | double |
| bool | i1 |
| *T | T* |
| unsafe.Pointer | i8* |

### 2.3 Code Generation
- Functions → LLVM functions
- Structs → LLVM struct types
- Arrays → LLVM array types
- Slices → `{ptr, len}` struct
- Control flow → LLVM basic blocks
- Inline assembly → LLVM inline asm

### 2.4 Directive Handling
- `//z:export` → LLVM linkage `external`
- `//z:inline` → LLVM `alwaysinline`
- `//z:noalloc` → Compile-time check, no heap alloc
- `//z:interrupt` → Generate interrupt wrapper (save/restore registers)
- `//z:packed` → LLVM packed struct
- `//z:align(N)` → LLVM alignment attribute
- `//z:volatile` → LLVM volatile load/store

### 2.5 Inline Assembly
```llvm
; Z: asm.Instr("cli")
call void asm sideeffect "cli", ""()

; Z: asm.Instr("mov %cr2, %rax", asm.Out("=a", &val))
%val = call i64 asm "mov %cr2, %rax", "=r"()
```

---

## Phase 3: Runtime & Linker (Week 5)

### 3.1 Minimal Runtime
```
runtime/zrt/
├── start.s         # Entry point (_start)
├── setjmp.s        # setjmp/longjmp (optional)
└── compiler-rt/    # LLVM compiler-rt intrinsics
```

### 3.2 Linker Script
```ld
ENTRY(_start)
SECTIONS {
    . = 1M;
    .text : { *(.text) }
    .rodata : { *(.rodata) }
    .data : { *(.data) }
    .bss : { *(.bss) }
}
```

### 3.3 Build Pipeline
```
.z → Lexer → Parser → AST → Semantics → LLVM IR → LLVM MC → .o → Linker → ELF
```

---

## Phase 4: Standard Library (Week 6)

### 4.1 Core Packages
```
stdlib/
├── asm/
│   └── asm.z          # asm.Instr, asm.Block, asm.In, etc.
├── unsafe/
│   └── unsafe.z       # Pointer, Sizeof, Offsetof, Slice
├── atomic/
│   └── atomic.z       # Load*, Store*, Add*, Swap*, CAS*
├── mem/
│   └── mem.z          # Memcpy, Memset, Memmove
├── errors/
│   └── errors.z       # error interface, New
└── fmt/
    └── fmt.z          # minimal console output
```

### 4.2 asm Package Implementation
```go
// This is compiler intrinsic - codegen handles it
package asm

type Operand struct{}

func In(constraint string, variable any) Operand { ... }
func Out(constraint string, variable any) Operand { ... }
func InOut(constraint string, in, out any) Operand { ... }
func Clobbers(regs ...string) Operand { ... }
func Instr(template string, ops ...Operand) { ... }
func Block(template string, ops ...Operand) { ... }
```

---

## Phase 5: OS Demo (Week 7-8)

### 5.1 OS Structure
```
os-demo/
├── boot/
│   └── multiboot.asm    # Multiboot header (GRUB boot)
├── kernel/
│   ├── main.z           # KernelMain entry
│   ├── arch/
│   │   └── x86_64/
│   │       ├── gdt.z    # GDT setup
│   │       ├── idt.z    # IDT & interrupt handlers
│   │       ├── cpu.z    # CPU utilities
│   │       └── vga.z    # VGA text mode
│   ├── mm/
│   │   ├── paging.z     # Page table management
│   │   └── heap.z       # Simple allocator
│   └── drivers/
│       ├── keyboard.z   # PS/2 keyboard
│       └── serial.z     # Serial port
├── linker.ld
└── Makefile
```

### 5.2 Minimal Kernel (main.z)
```go
package main

import (
    "asm"
    "kernel/arch/x86_64"
)

var ticks uint64

//z:interrupt
func TimerHandler(frame *x86_64.InterruptFrame) {
    ticks++
    x86_64.SendEOI()
}

//z:export
func KernelMain() {
    x86_64.InitVGA()
    x86_64.Print("Z OS v0.1\n")

    x86_64.InitGDT()
    x86_64.InitIDT()
    x86_64.SetInterruptHandler(32, TimerHandler)
    x86_64.EnableInterrupts()

    x86_64.Print("Kernel ready!\n")

    for {
        asm.Instr("hlt")
    }
}
```

### 5.3 GDT Setup (gdt.z)
```go
package x86_64

//z:packed
type GDTEntry struct {
    LimitLow  uint16
    BaseLow   uint16
    BaseMid   uint8
    Access    uint8
    LimitHigh uint8
    BaseHigh  uint8
}

type GDT [5]GDTEntry

//z:align(16)
var gdt GDT

func InitGDT() {
    // Set up null, kernel code, kernel data, user code, user data segments
    setGDTEntry(0, 0, 0, 0, 0)                    // null
    setGDTEntry(1, 0, 0xFFFFF, 0x9A, 0xA)         // kernel code
    setGDTEntry(2, 0, 0xFFFFF, 0x92, 0xC)         // kernel data

    asm.Instr("lgdt gdt_ptr")
}
```

### 5.4 IDT & Interrupts (idt.z)
```go
package x86_64

type InterruptFrame struct {
    IP, CS, Flags, SP, SS uintptr
    AX, BX, CX, DX        uintptr
    SI, DI, BP            uintptr
    R8, R9, R10, R11      uintptr
    R12, R13, R14, R15    uintptr
}

//z:packed
type IDTEntry struct {
    OffsetLow   uint16
    Selector    uint16
    IST         uint8
    Type        uint8
    OffsetMid   uint16
    OffsetHigh  uint32
    Reserved    uint32
}

type Handler func(frame *InterruptFrame)

var handlers [256]Handler
var idt [256]IDTEntry

func SetInterruptHandler(n int, h Handler) {
    handlers[n] = h
    // Set IDT entry
}

//z:interrupt
func DefaultHandler(frame *InterruptFrame) {
    Print("Unhandled interrupt\n")
    Halt()
}
```

### 5.5 VGA Driver (vga.z)
```go
package x86_64

const VGA_BASE = 0xB8000

var cursorX, cursorY int

func Print(msg string) {
    vga := (*[80*25*2]byte)(unsafe.Pointer(VGA_BASE))
    for _, c := range msg {
        if c == '\n' {
            cursorX = 0
            cursorY++
            continue
        }
        offset := (cursorY*80 + cursorX) * 2
        vga[offset] = byte(c)
        vga[offset+1] = 0x0F // white on black
        cursorX++
    }
}

func ClearScreen() {
    vga := (*[80*25*2]byte)(unsafe.Pointer(VGA_BASE))
    for i := range vga {
        vga[i] = 0
    }
    cursorX, cursorY = 0, 0
}
```

### 5.6 Paging (paging.z)
```go
package mm

//z:align(4096)
var pml4 [512]uint64

func InitPaging() {
    // Identity map first 4MB
    for i := 0; i < 512; i++ {
        pml4[i] = 0
    }

    // Enable paging
    asm.Instr("mov %rax, cr3", asm.In("a", &pml4))
    asm.Instr(`
        mov %cr0, %rax
        or $0x80000000, %rax
        mov %rax, %cr0
    `)
}
```

---

## Phase 6: Build System & QEMU (Week 8)

### 6.1 Makefile
```makefile
ZC = ./bin/zc
LLC = llc
CC = gcc
LD = ld

KERNEL_SRCS = $(wildcard kernel/*.z kernel/**/*.z)
KERNEL_OBJS = $(KERNEL_SRCS:.z=.o)

all: os.iso

%.o: %.z
	$(ZC) -emit-llvm -o $@.ll $<
	$(LLC) -filetype=obj -o $@ $@.ll

kernel.elf: $(KERNEL_OBJS) linker.ld boot/multiboot.o
	$(LD) -T linker.ld -o $@ $^

os.iso: kernel.elf
	mkdir -p isodir/boot/grub
	cp kernel.elf isodir/boot/kernel.elf
	echo 'menuentry "Z OS" { multiboot /boot/kernel.elf }' > isodir/boot/grub/grub.cfg
	grub-mkrescue -o $@ isodir

run: os.iso
	qemu-system-x86_64 -cdrom os.iso -serial stdio

debug: os.iso
	qemu-system-x86_64 -cdrom os.iso -serial stdio -s -S &
	gdb -ex "target remote :1234"

clean:
	rm -rf *.o *.ll kernel.elf os.iso isodir
```

### 6.2 QEMU Commands
```bash
# Run OS
qemu-system-x86_64 -cdrom os.iso

# Debug with GDB
qemu-system-x86_64 -cdrom os.iso -s -S
gdb -ex "target remote :1234"

# With serial output
qemu-system-x86_64 -cdrom os.iso -serial stdio

# Exit on shutdown
qemu-system-x86_64 -cdrom os.iso -no-reboot
```

---

## Implementation Milestones

| Milestone | Deliverable | Status |
|----------|-------------|--------|
| M1 | Lexer + Parser (AST output) | ✅ Done |
| M2 | Semantic analysis (type checking) | ✅ Done |
| M3 | LLVM IR generation (basic functions) | ✅ Done |
| M4 | Full codegen + runtime | ✅ Done |
| M5 | Standard library (asm, unsafe, atomic, mem, errors) | ✅ Done |
| M6 | OS demo boots to VGA output | ✅ Done (pure Z code) |
| M7 | OS demo with interrupts | 🔜 Next |

## Current Implementation Status (2026-06-02)

### Lexer: Complete
- All keywords, operators, literals (hex, octal, binary, raw strings, runes)
- `//z:` directives recognition
- Token types: `T_TRUE`, `T_FALSE`, `T_NIL`, `T_IOTA`, `T_UNSAFE`

### Parser: Complete
- Recursive descent parser with full expression grammar
- AST nodes: all declarations, statements, expressions
- `:=` short declaration, `var`/`const` blocks, `iota`
- Named return values, labels, import blocks
- Type cast expressions: `(*byte)(unsafe.Pointer(addr))`
- Pointer dereference: `*ptr = value`
- `StarExpr`/`AddrExpr`/`CastExpr` AST nodes

### Semantic Analysis: Complete
- Scope resolution (package, function, block scopes)
- Type checking with builtin type recognition
- Directive validation (`//z:interrupt`)
- `DeclStmt`, `CastExpr`, `StarExpr`, `SelectorExpr` checking

### Codegen: Complete
- Emitter + TypeResolver pattern
- LLVM IR text generation (no CGo dependency)
- Functions, structs, globals, control flow
- `switch`/`defer`/`goto`/labeled `break`/`continue`
- Inline assembly with AT&T constraint mapping
- Builtin intrinsics: `mem` (memcpy/memset), `atomic` (load/store/add/swap/cas), `unsafe` (Pointer/Sizeof/Slice)
- Type casts (`bitcast`), pointer dereference (`load`/`store`)
- Parameter type matching with `trunc`/`sext`
- Local variable type tracking

### Standard Library: Complete
- `asm` — `Instr`, `Block`, `In`, `Out`, `InOut`, `Clobbers`
- `unsafe` — `Pointer`, `Sizeof`, `Offsetof`, `Alignof`, `Slice`
- `atomic` — `LoadInt32`, `StoreInt32`, `AddInt32`, `SwapInt32`, `CAS`
- `mem` — `Memcpy`, `Memset`, `Memmove`
- `errors` — `error` interface, `New`

### CLI: Complete
- Multi-file input with merging
- Build tags: `//z:build tag` and `//z:build !tag`
- `-emit-llvm` flag, `-o` output flag, `-tags` flag

### OS Demo: Complete
- `vga_write_char` — extracted common function for VGA character output
- `KernelMain` — outputs "Hello Zlang!" to VGA text mode at 0xB8000
- Pure Z code: no `asm.Instr`, uses `unsafe.Pointer` + type casts + pointer dereference

---

## Technical Decisions

### LLVM Backend Choice
- **Option A**: Generate LLVM IR text, use `llc` to compile
- **Option B**: Use Go LLVM bindings (`tinygo-org/go-llvm`)
- **Recommendation**: Option A (simpler, easier debugging)

### Boot Method
- **Multiboot1**: GRUB legacy, simpler
- **Multiboot2**: GRUB2, more features
- **UEFI**: Modern but complex
- **Recommendation**: Multiboot1 for simplicity

### Target Format
- **ELF64**: Native Linux format, QEMU compatible
- **PE**: Windows format
- **Recommendation**: ELF64 for x86_64

---

## Success Criteria

1. **Compiler**: Compile valid Z code to working ELF binary
2. **Runtime**: No segfaults, proper stack setup
3. **OS Demo**:
   - Boots in QEMU
   - Displays "Z OS v0.1" on VGA console
   - Handles timer interrupt (increments counter)
   - Responds to keyboard input