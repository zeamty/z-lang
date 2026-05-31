package types

import (
	"fmt"
	"strconv"
)

// Type represents a type in the type system
type Type interface {
	String() string
	Size() int
	IsBasic() bool
	LLVMType() string
}

// Basic types
type Basic struct {
	Name string
	Kind BasicKind
}

type BasicKind int

const (
	Bool BasicKind = iota
	Int8
	Int16
	Int32
	Int64
	Int
	Uint8
	Uint16
	Uint32
	Uint64
	Uint
	Uintptr
	Float32
	Float64
	String
	Rune
	Byte
)

var BasicTypes = map[string]*Basic{
	"bool":     {Name: "bool", Kind: Bool},
	"int8":     {Name: "int8", Kind: Int8},
	"int16":    {Name: "int16", Kind: Int16},
	"int32":    {Name: "int32", Kind: Int32},
	"int64":    {Name: "int64", Kind: Int64},
	"int":      {Name: "int", Kind: Int},
	"uint8":    {Name: "uint8", Kind: Uint8},
	"uint16":   {Name: "uint16", Kind: Uint16},
	"uint32":   {Name: "uint32", Kind: Uint32},
	"uint64":   {Name: "uint64", Kind: Uint64},
	"uint":     {Name: "uint", Kind: Uint},
	"uintptr":  {Name: "uintptr", Kind: Uintptr},
	"float32":  {Name: "float32", Kind: Float32},
	"float64":  {Name: "float64", Kind: Float64},
	"string":   {Name: "string", Kind: String},
	"rune":     {Name: "rune", Kind: Rune},
	"byte":     {Name: "byte", Kind: Byte},
}

func (b *Basic) String() string { return b.Name }
func (b *Basic) IsBasic() bool  { return true }

func (b *Basic) Size() int {
	switch b.Kind {
	case Bool, Int8, Uint8, Byte:
		return 1
	case Int16, Uint16:
		return 2
	case Int32, Uint32, Float32, Rune:
		return 4
	case Int64, Uint64, Float64, Int, Uint, Uintptr:
		return 8
	case String:
		return 16 // ptr + len
	default:
		return 8
	}
}

func (b *Basic) LLVMType() string {
	switch b.Kind {
	case Bool:
		return "i1"
	case Int8, Uint8, Byte:
		return "i8"
	case Int16, Uint16:
		return "i16"
	case Int32, Uint32, Float32, Rune:
		return "i32"
	case Int64, Uint64, Int, Uint, Uintptr:
		return "i64"
	case Float64:
		return "double"
	case String:
		return "{i8*, i64}" // ptr + len
	default:
		return "i64"
	}
}

// Pointer type
type Pointer struct {
	Base Type
}

func (p *Pointer) String() string { return "*" + p.Base.String() }
func (p *Pointer) IsBasic() bool  { return false }
func (p *Pointer) Size() int      { return 8 }

func (p *Pointer) LLVMType() string {
	return p.Base.LLVMType() + "*"
}

// Array type
type Array struct {
	Len int
	Elt Type
}

func (a *Array) String() string { return fmt.Sprintf("[%d]%s", a.Len, a.Elt.String()) }
func (a *Array) IsBasic() bool  { return false }
func (a *Array) Size() int      { return a.Len * a.Elt.Size() }

func (a *Array) LLVMType() string {
	return fmt.Sprintf("[%d x %s]", a.Len, a.Elt.LLVMType())
}

// Slice type
type Slice struct {
	Elt Type
}

func (s *Slice) String() string { return "[]" + s.Elt.String() }
func (s *Slice) IsBasic() bool  { return false }
func (s *Slice) Size() int      { return 16 } // ptr + len

func (s *Slice) LLVMType() string {
	return fmt.Sprintf("{%s*, i64}", s.Elt.LLVMType())
}

// Struct type
type Struct struct {
	Name   string
	Fields []*Var
	Packed bool
}

type Var struct {
	Name string
	Type Type
}

func (s *Struct) String() string { return s.Name }
func (s *Struct) IsBasic() bool  { return false }
func (s *Struct) Size() int {
	size := 0
	for _, f := range s.Fields {
		size += f.Type.Size()
	}
	return size
}

func (s *Struct) LLVMType() string {
	fields := ""
	for i, f := range s.Fields {
		if i > 0 {
			fields += ", "
		}
		fields += f.Type.LLVMType()
	}
	if s.Packed {
		return fmt.Sprintf("<{%s}>", fields)
	}
	return fmt.Sprintf("{%s}", fields)
}

// Func type
type Func struct {
	Params  []Type
	Results []Type
}

func (f *Func) String() string { return "func" }
func (f *Func) IsBasic() bool  { return false }
func (f *Func) Size() int      { return 8 } // function pointer

func (f *Func) LLVMType() string {
	ret := "void"
	if len(f.Results) > 0 {
		ret = f.Results[0].LLVMType()
	}
	params := ""
	for i, p := range f.Params {
		if i > 0 {
			params += ", "
		}
		params += p.LLVMType()
	}
	return fmt.Sprintf("%s (%s)*", ret, params)
}

// LLVMType returns LLVM type string for a type
func LLVMType(t Type) string {
	switch tt := t.(type) {
	case *Basic:
		return tt.LLVMType()
	case *Pointer:
		return tt.LLVMType()
	case *Array:
		return tt.LLVMType()
	case *Slice:
		return tt.LLVMType()
	case *Struct:
		return tt.LLVMType()
	case *Func:
		return tt.LLVMType()
	default:
		return "i64"
	}
}

// ParseInt parses integer literal
func ParseInt(s string) int64 {
	// Handle hex, octal, binary
	if len(s) > 2 {
		switch s[:2] {
		case "0x", "0X":
			val, _ := strconv.ParseInt(s[2:], 16, 64)
			return val
		case "0o", "0O":
			val, _ := strconv.ParseInt(s[2:], 8, 64)
			return val
		case "0b", "0B":
			val, _ := strconv.ParseInt(s[2:], 2, 64)
			return val
		}
	}
	// Remove underscores
	clean := ""
	for _, c := range s {
		if c != '_' {
			clean += string(c)
		}
	}
	val, _ := strconv.ParseInt(clean, 10, 64)
	return val
}

// ParseFloat parses float literal
func ParseFloat(s string) float64 {
	clean := ""
	for _, c := range s {
		if c != '_' {
			clean += string(c)
		}
	}
	val, _ := strconv.ParseFloat(clean, 64)
	return val
}