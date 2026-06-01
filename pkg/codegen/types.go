package codegen

import (
	"fmt"

	"github.com/zeamty/z-lang/pkg/lexer"
	"github.com/zeamty/z-lang/pkg/parser"
	"github.com/zeamty/z-lang/pkg/types"
)

// TypeResolver maps parser types to LLVM types
type TypeResolver struct {
	vars      map[string]*parser.VarDecl
	functions map[string]*parser.FuncDecl
	structs   map[string]*types.Struct
}

func newTypeResolver(
	vars map[string]*parser.VarDecl,
	functions map[string]*parser.FuncDecl,
) *TypeResolver {
	return &TypeResolver{
		vars:      vars,
		functions: functions,
		structs:   make(map[string]*types.Struct),
	}
}

func (tr *TypeResolver) resolveType(t parser.Type) types.Type {
	switch tt := t.(type) {
	case *parser.Ident:
		if basic, ok := types.BasicTypes[tt.Name]; ok {
			return basic
		}
		if s, ok := tr.structs[tt.Name]; ok {
			return s
		}
		return types.BasicTypes["int"]

	case *parser.PointerType:
		return &types.Pointer{Base: tr.resolveType(tt.Base)}

	case *parser.ArrayType:
		if lit, ok := tt.Len.(*parser.BasicLit); ok {
			len := types.ParseInt(lit.Value)
			return &types.Array{Len: int(len), Elt: tr.resolveType(tt.Elt)}
		}
		return &types.Array{Len: 0, Elt: tr.resolveType(tt.Elt)}

	case *parser.SliceType:
		return &types.Slice{Elt: tr.resolveType(tt.Elt)}

	case *parser.StructType:
		st := &types.Struct{Name: "anon"}
		for _, f := range tt.Fields {
			st.Fields = append(st.Fields, &types.Var{
				Name: f.Name.Name,
				Type: tr.resolveType(f.Type),
			})
		}
		return st

	case *parser.FuncType:
		fn := &types.Func{}
		for _, p := range tt.Params {
			fn.Params = append(fn.Params, tr.resolveType(p.Type))
		}
		for _, r := range tt.Results {
			fn.Results = append(fn.Results, tr.resolveType(r.Type))
		}
		return fn

	default:
		return types.BasicTypes["int"]
	}
}

func (tr *TypeResolver) exprType(expr parser.Expr) types.Type {
	switch e := expr.(type) {
	case *parser.Ident:
		if v, ok := tr.vars[e.Name]; ok {
			return tr.resolveType(v.Type)
		}
		return types.BasicTypes["int64"]

	case *parser.BasicLit:
		switch e.Type {
		case lexer.T_INT:
			return types.BasicTypes["int64"]
		case lexer.T_FLOAT:
			return types.BasicTypes["float64"]
		case lexer.T_STRING:
			return types.BasicTypes["string"]
		case lexer.T_CHAR:
			return types.BasicTypes["rune"]
		}
		return types.BasicTypes["int64"]

	case *parser.BinaryExpr:
		return tr.exprType(e.X)

	case *parser.UnaryExpr:
		return tr.exprType(e.X)

	case *parser.CallExpr:
		if ident, ok := e.Fun.(*parser.Ident); ok {
			if fn := tr.functions[ident.Name]; fn != nil {
				if len(fn.Type.Results) > 0 {
					return tr.resolveType(fn.Type.Results[0].Type)
				}
			}
		}
	}

	return types.BasicTypes["int64"]
}

func literalValue(lit *parser.BasicLit) string {
	switch lit.Type {
	case lexer.T_INT:
		return fmt.Sprintf("%d", types.ParseInt(lit.Value))
	case lexer.T_FLOAT:
		return fmt.Sprintf("%f", types.ParseFloat(lit.Value))
	case lexer.T_CHAR:
		if len(lit.Value) >= 3 {
			inner := lit.Value[1 : len(lit.Value)-1]
			if inner == "\\n" {
				return "10"
			}
			return fmt.Sprintf("%d", inner[0])
		}
		return "0"
	case lexer.T_STRING:
		return ""
	}
	return "0"
}