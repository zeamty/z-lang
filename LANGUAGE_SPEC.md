# Z Language Specification

**Version 0.1 Draft**
**May 2026**

---

## 1. Introduction

Z (zlang) is a systems programming language designed for operating system development. It uses Go-like syntax while providing low-level control necessary for bare-metal programming.

### 1.1 Design Goals

1. **Go-like Syntax** - Familiar to Go developers
2. **No Runtime Dependency** - No garbage collector, scheduler, or hidden runtime
3. **Memory Safety by Default** - Safe defaults with explicit escape hatches
4. **Deterministic Behavior** - No hidden allocations, no unexpected control flow
5. **Bare-metal Ready** - First-class support for interrupts, memory-mapped I/O, and inline assembly

### 1.2 File Extension

Z source files use `.z` extension:

```
main.z
paging.z
keyboard.z
```

### 1.3 What's Removed from Go

| Feature | Reason |
|---------|--------|
| Garbage Collector | OS kernels need deterministic memory management |
| Goroutines | Kernels provide scheduling, not use it |
| Channels | High-level concurrency primitive |
| Reflection | Requires runtime type information |
| recover | No runtime panic unwinding |
| Complex Standard Library | Replaced with kernel-specific primitives |
| Map type | Requires runtime hashing and allocation |

### 1.4 What's Added

| Feature | Purpose |
|---------|---------|
| Inline Assembly | `asm` package for CPU-specific instructions |
| Memory Layout Control | `//z:packed`, `//z:align` directives |
| Interrupt Handlers | `//z:interrupt` directive |
| Volatile Operations | Memory-mapped I/O |
| Noalloc Functions | `//z:noalloc` for compile-time allocation checking |
| Explicit Pointer Types | `unsafe.Pointer` with pointer arithmetic |

---

## 2. Source Code Structure

### 2.1 Package Clause

Every Z source file begins with a package clause:

```
package main

package mm

package drivers
```

### 2.2 Import Declarations

```
import "fmt"

import (
    "kernel/mm"
    "kernel/cpu"
)

import cpu "kernel/arch/cpu"  // aliased import
```

---

## 3. Lexical Elements

### 3.1 Keywords

```
break    const    continue  default  defer
else     for      func      goto     if
import   interface package   range    return
struct   switch   type      var      fallthrough

unsafe   volatile
```

### 3.2 Operators and Punctuation

```
+    &    +=   &=   &&   ==   !=   (    )
-    |    -=   |=   ||   <    <=   [    ]
*    ^    *=   ^=   <-   >    >=   {    }
/    <<   /=   <<=  ++   =    :=   ,    ;
%    >>   %=   >>=  --   !    ...  .    :

&^   &^=
```

### 3.3 Integer Literals

```
42
0x2A        // hexadecimal
0o52        // octal
0b101010    // binary
1_000_000   // underscores for readability
```

### 3.4 Floating-point Literals

```
3.14
1e10
6.022e23
```

### 3.5 Rune Literals

```
'a'
'\n'
'\x41'
'世'
'世'
'\U0001F600'
```

### 3.6 String Literals

```
"hello, world"        // interpreted string
`raw string
can span multiple lines` // raw string
```

---

## 4. Types

### 4.1 Basic Types

```
bool

// Signed integers
int8    int16    int32    int64    int
// Unsigned integers
uint8   uint16   uint32   uint64   uint    uintptr

// Aliases
byte = uint8
rune = int32

// Floating point (optional in kernel)
float32    float64
```

Note: `int` and `uint` are 64-bit on 64-bit platforms, 32-bit on 32-bit platforms. Use explicit sizes (`int32`, `int64`) when needed.

### 4.2 Pointer Types

```
*T              // regular pointer
unsafe.Pointer  // unsafe pointer (can be cast freely)
```

Pointer arithmetic on `unsafe.Pointer`:

```
var p unsafe.Pointer
p = unsafe.Pointer(uintptr(p) + 16)  // pointer arithmetic
```

### 4.3 Struct Types

```
// Regular struct
type Point struct {
    X, Y int32
}

// Packed struct (use directive)
//z:packed
type Header struct {
    Flags uint8
    Size  uint16
    Type  uint8
}

// Aligned fields
type PageTable struct {
    Entries [512]uint64 `align:"4096"`
}
```

### 4.4 Array Types

```
[4]int32
[1024]byte
[len]byte  // len must be compile-time constant
```

### 4.5 Slice Types

```
[]int32

// Slice literal
[]int32{1, 2, 3}

// Slice from array
arr := [4]int32{1, 2, 3, 4}
s := arr[:]      // full slice
s = arr[1:3]     // partial slice
```

Slice header:

```
type slice struct {
    ptr unsafe.Pointer
    len uintptr
}
```

---

## 5. Constants

### 5.1 Typed Constants

```
const PageSize uintptr = 4096

const (
    MaxCPUs   = 256
    MaxFiles  = 1024
)
```

### 5.2 Untyped Constants

```
const (
    a = 42              // untyped int
    b = 3.14            // untyped float
    c = "hello"         // untyped string
)
```

### 5.3 Iota

```
const (
    StatusOK = iota
    StatusError
    StatusPending
)

const (
    InterruptTimer    = iota
    InterruptKeyboard
    InterruptDisk
)

const (
    bit0 = 1 << iota
    bit1
    bit2
    bit3
)
```

---

## 6. Variables

### 6.1 Variable Declarations

```
var counter int32
var buffer [1024]byte

var (
    x int32
    y float64
)
```

### 6.2 Short Variable Declarations

```
x := 42
ptr := &x
msg := "hello"
```

### 6.3 Zero Values

```
var i int         // 0
var f float64     // 0.0
var b bool        // false
var s string      // ""
var p *int        // nil
var arr [4]int    // [0, 0, 0, 0]
```

---

## 7. Functions

### 7.1 Function Declarations

```
func add(a, b int32) int32 {
    return a + b
}

func log(msg string) {
    // no return value
}
```

### 7.2 Named Return Values

```
func divmod(a, b int32) (quotient, remainder int32) {
    quotient = a / b
    remainder = a % b
    return
}
```

### 7.3 Multiple Return Values

```
func parse(s string) (int32, error) {
    // ...
    return value, nil
}
```

### 7.4 Function Types

```
type Handler func(frame *InterruptFrame)

var handlers [256]Handler
```

---

## 8. Compiler Directives

Z uses `//z:` prefix for compiler directives:

### 8.1 Export

```
//z:export
func KernelMain(info *MultibootInfo) {
    // ...
}
```

### 8.2 Inline

```
//z:inline
func square(x int32) int32 {
    return x * x
}
```

### 8.3 Noalloc

```
//z:noalloc
func handler(frame *InterruptFrame) {
    // Compiler error if allocation attempted
}
```

### 8.4 Interrupt

```
//z:interrupt
func timerHandler(frame *InterruptFrame) {
    ticks++
}
```

### 8.5 Packed

```
//z:packed
type Descriptor struct {
    LimitLow  uint16
    BaseLow   uint16
    BaseMid   uint8
    Access    uint8
    LimitHigh uint8
    BaseHigh  uint8
}
```

### 8.6 Align

```
//z:align(4096)
var pageTable [512]uint64

//z:align(16)
type KernelStack [16384]byte
```

### 8.7 Volatile

```
//z:volatile
var mmioReg uint32
```

### 8.8 Linker Symbol

```
//z:linker
var (
    KernelStart uintptr
    KernelEnd   uintptr
)
```

### 8.9 Build Tags

```
//z:build amd64

//z:build !debug
```

---

## 9. Methods

### 9.1 Method Declarations

```
type Counter struct {
    value int32
}

// Value receiver
func (c Counter) Get() int32 {
    return c.value
}

// Pointer receiver
func (c *Counter) Increment() {
    c.value++
}
```

### 9.2 Constructor Pattern

```
func NewCounter() *Counter {
    return &Counter{value: 0}
}
```

---

## 10. Interfaces

### 10.1 Interface Types

```
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type ReadWriter interface {
    Reader
    Writer
}
```

### 10.2 Interface Satisfaction

Interfaces are satisfied implicitly (same as Go):

```
type File struct{}

func (f *File) Read(p []byte) (int, error) {
    return 0, nil
}

func (f *File) Write(p []byte) (int, error) {
    return 0, nil
}

// File implicitly implements ReadWriter
var rw ReadWriter = &File{}
```

### 10.3 Restrictions

No type assertions or type switches:

```
// NOT ALLOWED in Z
if r, ok := v.(Reader); ok { }

// NOT ALLOWED in Z
switch v := i.(type) { }
```

Interfaces are used only for compile-time polymorphism via monomorphization.

---

## 11. Control Flow

### 11.1 If Statements

```
if x > 0 {
    // ...
} else if x < 0 {
    // ...
} else {
    // ...
}

// With initialization
if err := doSomething(); err != nil {
    return err
}
```

### 11.2 For Statements

```
// Traditional for
for i := 0; i < 10; i++ {
    // ...
}

// While-like
for x < 10 {
    x++
}

// Infinite loop
for {
    if done {
        break
    }
}

// Range over array/slice
for i, v := range arr {
    // i = index, v = value
}

for i := range arr {
    // i = index only
}
```

### 11.3 Switch Statements

```
switch x {
case 1:
    // ...
case 2, 3:
    // ...
default:
    // ...
}

// Tagless switch
switch {
case x < 0:
    // ...
case x == 0:
    // ...
default:
    // ...
}
```

### 11.4 Break, Continue, Goto

```
outer:
for i := 0; i < 10; i++ {
    for j := 0; j < 10; j++ {
        if i == j {
            break outer
        }
    }
}

goto error
error:
    // error handling
```

### 11.5 Defer

Z's defer is compile-time (generates code at scope exit):

```
func example() error {
    f, err := OpenFile("test.txt")
    if err != nil {
        return err
    }
    defer CloseFile(f)

    // ... use f ...

    return nil
} // CloseFile(f) called here
```

---

## 12. Error Handling

### 12.1 Error Type

```
type error interface {
    Error() string
}
```

### 12.2 Error Values

```
var (
    ErrNotFound   = errors.New("not found")
    ErrPermission = errors.New("permission denied")
)

func doSomething() error {
    if !ok {
        return ErrNotFound
    }
    return nil
}
```

### 12.3 No Panic/Recover

Z removes panic/recover. Use explicit error returns or kernel panic function:

```
func Panic(msg string) {
    Print("KERNEL PANIC: ")
    Println(msg)
    Halt()
}
```

---

## 13. Inline Assembly

Z provides inline assembly through the `asm` standard library package. The `asm` package offers functions for embedding CPU-specific instructions with optional operand constraints.

### 13.1 Importing the asm Package

```
import "asm"
```

### 13.2 Simple Instructions

For instructions without operands, use `asm.Instr()`:

```
func CLI() {
    asm.Instr("cli")  // disable interrupts
}

func HLT() {
    asm.Instr("hlt")  // halt CPU
}

func STI() {
    asm.Instr("sti")  // enable interrupts
}

func NOP() {
    asm.Instr("nop")  // no operation
}
```

### 13.3 Instruction Blocks

For multiple instructions, use `asm.Block()` with a raw string literal:

```
func EnableSSE() {
    asm.Block(`
        mov %cr4, %rax
        or $0x600, %rax
        mov %rax, %cr4
    `)
}

func SwapStack(newStack uintptr) {
    asm.Block(`
        mov %rsp, %rbx
        mov newStack, %rsp
        push %rbx
    `)
}
```

### 13.4 Operand Constraints

For instructions that read or write variables, use operand constraints:

```
// Operand types
asm.In(constraint, variable)   // input operand
asm.Out(constraint, variable)  // output operand
asm.InOut(constraint, variable) // read/write operand
```

#### Register Constraints

| Constraint | Register |
|------------|----------|
| `"a"` | AL / AX / EAX / RAX |
| `"b"` | BL / BX / EBX / RBX |
| `"c"` | CL / CX / ECX / RCX |
| `"d"` | DL / DX / EDX / RDX |
| `"S"` | SIL / SI / ESI / RSI |
| `"D"` | DIL / DI / EDI / RDI |
| `"r"` | Any general-purpose register |
| `"m"` | Memory location |
| `"i"` | Immediate constant |

#### Output Constraints

Output constraints start with `"="`:

```
"=a"  // output to RAX
"=r"  // output to any register
"=m"  // output to memory
```

### 13.5 Port I/O

```
func Outb(port uint16, value byte) {
    asm.Instr("outb %al, %dx",
        asm.In("a", &value),  // value -> AL
        asm.In("d", &port),   // port -> DX
    )
}

func Inb(port uint16) byte {
    var value byte
    asm.Instr("inb %dx, %al",
        asm.Out("=a", &value), // AL -> value
        asm.In("d", &port),    // port -> DX
    )
    return value
}

func Outl(port uint16, value uint32) {
    asm.Instr("outl %eax, %dx",
        asm.In("a", &value),
        asm.In("d", &port),
    )
}

func Inl(port uint16) uint32 {
    var value uint32
    asm.Instr("inl %dx, %eax",
        asm.Out("=a", &value),
        asm.In("d", &port),
    )
    return value
}
```

### 13.6 Control Registers

```
func ReadCR0() uintptr {
    var val uintptr
    asm.Instr("mov %cr0, %rax",
        asm.Out("=a", &val),
    )
    return val
}

func ReadCR2() uintptr {
    var val uintptr
    asm.Instr("mov %cr2, %rax",
        asm.Out("=a", &val),
    )
    return val
}

func ReadCR3() uintptr {
    var val uintptr
    asm.Instr("mov %cr3, %rax",
        asm.Out("=a", &val),
    )
    return val
}

func WriteCR3(val uintptr) {
    asm.Instr("mov %rax, %cr3",
        asm.In("a", &val),
    )
}
```

### 13.7 CPUID

```
func CPUID(eaxIn uint32) (eax, ebx, ecx, edx uint32) {
    asm.Block("cpuid",
        asm.InOut("a", &eaxIn, &eax),  // RAX: input -> output
        asm.Out("=b", &ebx),
        asm.Out("=c", &ecx),
        asm.Out("=d", &edx),
    )
    return
}

// Get CPU vendor string
func CPUVendor() string {
    var ebx, ecx, edx uint32
    asm.Block("cpuid",
        asm.In("a", 0),
        asm.Out("=b", &ebx),
        asm.Out("=c", &ecx),
        asm.Out("=d", &edx),
    )
    // Convert registers to string "GenuineIntel" or "AuthenticAMD"
    return vendorFromRegs(ebx, ecx, edx)
}
```

### 13.8 Clobbers

Specify registers that are modified by the assembly:

```
func Multiply(a, b uint64) uint64 {
    var result uint64
    asm.Block(`
        mul %rdx
        mov %rax, result
    `,
        asm.In("a", &a),       // RAX = a
        asm.In("d", &b),       // RDX = b
        asm.Out("=a", &result), // RAX -> result
        asm.Clobbers("rax", "rdx"), // these registers are modified
    )
    return result
}
```

`asm.Clobbers()` specifies registers that the assembly code modifies, preventing the compiler from using them for other purposes.

### 13.9 Complete Block Syntax

```
asm.Block(asmTemplate, operands...)
```

Where `operands` can be:
- `asm.In(constraint, variable)` - input only
- `asm.Out(constraint, variable)` - output only
- `asm.InOut(constraint, inVar, outVar)` - input and output
- `asm.Clobbers(registers...)` - clobbered registers

Example with all operand types:

```
func ComplexOp(input uint32) (output uint32, carry bool) {
    var flags uint64
    asm.Block(`
        mov input, %eax
        add $1, %eax
        pushf
        pop flags
        mov %eax, output
    `,
        asm.In("m", &input),
        asm.Out("=a", &output),
        asm.Out("=m", &flags),
        asm.Clobbers("eax", "flags"),
    )
    carry = (flags & 0x1) != 0
    return
}
```

### 13.10 Instruction with Operands

For single instructions with operands, use `asm.Instr()`:

```
asm.Instr(template, operands...)
```

```
// Atomic swap
func Xchg(ptr *uint32, new uint32) uint32 {
    var old uint32
    asm.Instr("xchg %eax, (%rbx)",
        asm.InOut("a", &new, &old), // RAX: new -> old
        asm.In("b", &ptr),          // RBX = ptr
    )
    return old
}

// Compare and swap
func Cas(ptr *uint32, expected, new uint32) bool {
    var result uint32
    asm.Instr(`
        lock cmpxchg new, (%rbx)
        setz %al
        movzx %al, result
    `,
        asm.InOut("a", &expected, &result),
        asm.In("b", &ptr),
        asm.In("m", &new),
    )
    return result != 0
}
```

---

## 14. Interrupt Handling

### 14.1 Interrupt Handler Definition

```
//z:interrupt
func TimerHandler(frame *InterruptFrame) {
    ticks++
    Schedule()
}

//z:interrupt
func KeyboardHandler(frame *InterruptFrame) {
    scancode := Inb(0x60)
    ProcessKeyboard(scancode)
}

//z:interrupt
func PageFaultHandler(frame *InterruptFrame) {
    faultAddr := ReadCR2()
    if !HandlePageFault(faultAddr) {
        Panic("unhandled page fault")
    }
}
```

### 14.2 Interrupt Frame

```
type InterruptFrame struct {
    // Pushed by CPU
    IP    uintptr
    CS    uintptr
    Flags uintptr
    SP    uintptr
    SS    uintptr

    // Pushed by interrupt wrapper
    AX, BX, CX, DX uintptr
    SI, DI, BP     uintptr
    R8, R9, R10, R11, R12, R13, R14, R15 uintptr
}
```

---

## 15. Memory Management

### 15.1 No Built-in Allocator

Z has no `new()` or `make()`. The kernel provides its own allocator:

```
type BumpAllocator struct {
    start   unsafe.Pointer
    current unsafe.Pointer
    end     unsafe.Pointer
}

func (a *BumpAllocator) Alloc(size, align uintptr) unsafe.Pointer {
    aligned := AlignUp(a.current, align)
    newCurrent := unsafe.Pointer(uintptr(aligned) + size)

    if uintptr(newCurrent) > uintptr(a.end) {
        return nil // out of memory
    }

    a.current = newCurrent
    return aligned
}
```

### 15.2 Zero Allocation Functions

```
//z:noalloc
func InterruptHandler(frame *InterruptFrame) {
    // Compiler ensures no allocations
}
```

### 15.3 Volatile Access

```
//z:volatile
var mmioReg uint32

func ReadMMIO(addr uintptr) uint32 {
    p := (*uint32)(unsafe.Pointer(addr))
    return *p  // volatile read
}

func WriteMMIO(addr uintptr, value uint32) {
    p := (*uint32)(unsafe.Pointer(addr))
    *p = value  // volatile write
}
```

---

## 16. Concurrency Primitives

### 16.1 Atomic Operations

```
var counter int32

// Load
v := atomic.LoadInt32(&counter)

// Store
atomic.StoreInt32(&counter, 42)

// Add
atomic.AddInt32(&counter, 1)

// Swap
old := atomic.SwapInt32(&counter, 0)

// Compare and Swap
if atomic.CompareAndSwapInt32(&counter, expected, new) {
    // success
}
```

### 16.2 Spin Lock

```
type SpinLock struct {
    locked uint32
}

func (l *SpinLock) Lock() {
    for !atomic.CompareAndSwapUint32(&l.locked, 0, 1) {
        // spin
    }
}

func (l *SpinLock) Unlock() {
    atomic.StoreUint32(&l.locked, 0)
}
```

---

## 17. Memory Layout Control

### 17.1 Packed Structs

```
// Normal struct (with padding)
type Header struct {
    Flags uint8   // 1 byte
    // 3 bytes padding
    Size  uint32  // 4 bytes
}
// sizeof(Header) = 8

//z:packed
type PackedHeader struct {
    Flags uint8   // offset 0
    Size  uint32  // offset 1 (unaligned!)
}
// sizeof(PackedHeader) = 5
```

### 17.2 Alignment Control

```
//z:align(4096)
var pageTable [512]uint64

//z:align(16)
type KernelStack [16384]byte
```

### 17.3 Sizeof, Offsetof, Alignof

```
size := unsafe.Sizeof(Header{})
offset := unsafe.Offsetof(Header{}.Size)
align := unsafe.Alignof(Header{})
```

---

## 18. unsafe Package

### 18.1 unsafe.Pointer

```
import "unsafe"

// Pointer conversion
ptr := unsafe.Pointer(&x)

// Pointer arithmetic
ptr = unsafe.Pointer(uintptr(ptr) + 16)

// Convert to typed pointer
p := (*int32)(ptr)
```

### 18.2 unsafe.Slice

```
// Convert array pointer to slice
arr := [1024]byte{}
slice := unsafe.Slice(&arr[0], len(arr))
```

### 18.3 unsafe.Sizeof, Offsetof, Alignof

```
size := unsafe.Sizeof(Header{})
offset := unsafe.Offsetof(Header{}.Size)
align := unsafe.Alignof(Header{})
```

---

## 19. Packages

### 19.1 Package Structure

```
kernel/
├── main.z           # package main
├── mm/
│   ├── paging.z     # package mm
│   ├── heap.z       # package mm
│   └── alloc.z      # package mm
├── arch/
│   └── x86_64/
│       ├── cpu.z    # package x86_64
│       ├── idt.z    # package x86_64
│       └── gdt.z    # package x86_64
└── drivers/
    ├── keyboard.z   # package drivers
    └── serial.z     # package drivers
```

### 19.2 Export Rules

Capitalized names are exported:

```
package mm

var pageSize uintptr     // private
var PageSize uintptr      // exported

func alloc() []byte       // private
func Alloc() []byte       // exported

type allocator struct {}  // private
type Allocator struct {}  // exported
```

---

## 20. Build Configuration

### 20.1 Build Tags

```
//z:build amd64
//z:build linux

// This file only compiles for amd64 linux
```

```
//z:build !debug

// Excluded in debug builds
```

### 20.2 Linker Symbols

```
//z:linker
var (
    KernelStart uintptr
    KernelEnd   uintptr
)

func KernelSize() uintptr {
    return KernelEnd - KernelStart
}
```

---

## 21. Minimal Kernel Example

```
// kernel/main.z
package main

import (
    "asm"
    "kernel/arch/x86_64"
    "kernel/mm"
)

var ticks uint64

//z:align(16)
var kernelStack [16384]byte

//z:interrupt
func TimerHandler(frame *x86_64.InterruptFrame) {
    ticks++
    x86_64.SendEOI(0)
}

//z:export
func KernelMain(info *MultibootInfo) {
    x86_64.InitVGA()
    Println("Z Kernel v0.1")

    x86_64.InitGDT()
    Println("GDT initialized")

    x86_64.InitIDT()
    Println("IDT initialized")

    mm.InitPaging(info)
    Println("Paging enabled")

    x86_64.SetInterruptHandler(32, TimerHandler)
    x86_64.EnableInterrupts()

    Println("Kernel ready!")

    for {
        asm.Instr("hlt")
    }
}

//z:interrupt
func PageFaultHandler(frame *x86_64.InterruptFrame) {
    faultAddr := x86_64.ReadCR2()
    Print("Page fault at 0x")
    Println(Hex(faultAddr))

    for {
        asm.Instr("hlt")
    }
}
```

---

## Appendix A: Compiler Directives Summary

| Directive | Usage |
|-----------|-------|
| `//z:export` | Export function to linker |
| `//z:inline` | Force inline function |
| `//z:noalloc` | Prohibit allocations |
| `//z:interrupt` | Mark as interrupt handler |
| `//z:packed` | Pack struct (no padding) |
| `//z:align(N)` | Align variable/type to N bytes |
| `//z:volatile` | Mark as volatile |
| `//z:linker` | Define linker symbol |
| `//z:build tags` | Build constraints |

---

## Appendix B: Differences from Go

| Feature | Go | Z |
|---------|-----|---|
| Language name | Go | Z (zlang) |
| File extension | `.go` | `.z` |
| Directives | `//go:` | `//z:` |
| Inline assembly | Limited | `asm` package |
| Garbage Collection | Yes | No |
| Goroutines | Yes | No |
| Channels | Yes | No |
| Maps | Yes | No |
| panic/recover | Yes | No |
| Reflection | Yes | No |
| Type assertions | Yes | No |
| new/make | Yes | No |
| Packed struct | No | `//z:packed` |
| Interrupt support | No | `//z:interrupt` |
| Noalloc check | No | `//z:noalloc` |

---

## Appendix C: Minimal Standard Library

```
package asm      // inline assembly: Instr, Block, In, Out, InOut, Clobbers
package unsafe   // unsafe.Pointer, unsafe.Slice, unsafe.Sizeof, etc.
package atomic   // atomic operations
package bits     // bit manipulation
package mem      // Memcpy, Memset, Memmove
package sync     // SpinLock
package errors   // error type, New
package strconv  // string conversions
package hex      // Hex encoding
package binary   // binary encoding
package fmt      // minimal formatting (kernel console)
```

---

## Appendix D: asm Package API

### Functions

```
// Execute a single assembly instruction
func Instr(template string, operands ...Operand)

// Execute multiple assembly instructions
func Block(template string, operands ...Operand)
```

### Operand Constructors

```
// Input operand: variable -> register/memory
func In(constraint string, variable any) Operand

// Output operand: register/memory -> variable
func Out(constraint string, variable any) Operand

// Input/output operand: variable in, variable out
func InOut(constraint string, inVar, outVar any) Operand

// Specify clobbered registers
func Clobbers(registers ...string) Operand
```

### Register Constraints (x86-64)

| Constraint | Register |
|------------|----------|
| `"a"` | RAX (AL/AX/EAX/RAX) |
| `"b"` | RBX (BL/BX/EBX/RBX) |
| `"c"` | RCX (CL/CX/ECX/RCX) |
| `"d"` | RDX (DL/DX/EDX/RDX) |
| `"S"` | RSI (SIL/SI/ESI/RSI) |
| `"D"` | RDI (DIL/DI/EDI/RDI) |
| `"r"` | Any general-purpose register |
| `"m"` | Memory location |
| `"i"` | Immediate constant |
| `"x"` | XMM register (SSE) |

### Output Constraint Prefixes

| Prefix | Meaning |
|--------|---------|
| `"="` | Write-only output |
| `"+"` | Read-write (input and output) |

### Examples

```
// Simple instruction (no operands)
asm.Instr("cli")
asm.Instr("hlt")

// Instruction with input operands
asm.Instr("mov %rax, %rbx",
    asm.In("a", &x),
    asm.Out("=b", &y),
)

// Block with multiple operands
asm.Block(`
    mov input, %eax
    cpuid
    mov %eax, output
`,
    asm.In("a", &input),
    asm.Out("=a", &output),
    asm.Clobbers("ebx", "ecx", "edx"),
)

// Port I/O
func Inb(port uint16) byte {
    var val byte
    asm.Instr("inb %dx, %al",
        asm.Out("=a", &val),
        asm.In("d", &port),
    )
    return val
}

// Control register
func ReadCR2() uintptr {
    var val uintptr
    asm.Instr("mov %cr2, %rax",
        asm.Out("=a", &val),
    )
    return val
}
```

---

*End of Specification*