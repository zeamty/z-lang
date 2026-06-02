package codegen

import (
	"fmt"

	"github.com/zeamty/z-lang/pkg/parser"
)

func (g *Generator) emitMemCall(name string, args []parser.Expr) string {
	switch name {
	case "Memcpy", "Memmove":
		if len(args) >= 3 {
			dst := g.emitExpr(args[0])
			src := g.emitExpr(args[1])
			n := g.emitExpr(args[2])
			g.emitter.emitf("call void @llvm.memcpy.p0i8.p0i8.i64(i8* %s, i8* %s, i64 %s, i1 false)",
				dst, src, n)
		}
	case "Memset":
		if len(args) >= 3 {
			dst := g.emitExpr(args[0])
			c := g.emitExpr(args[1])
			n := g.emitExpr(args[2])
			g.emitter.emitf("call void @llvm.memset.p0i8.i64(i8* %s, i8 %s, i64 %s, i1 false)",
				dst, c, n)
		}
	}
	return ""
}

func (g *Generator) emitAtomicCall(name string, args []parser.Expr) string {
	switch name {
	case "LoadInt32", "LoadUint32":
		if len(args) >= 1 {
			ptr := g.emitExpr(args[0])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = load atomic i32, i32* %s seq_cst", tmp, ptr)
			return tmp
		}
	case "StoreInt32", "StoreUint32":
		if len(args) >= 2 {
			ptr := g.emitExpr(args[0])
			val := g.emitExpr(args[1])
			g.emitter.emitf("store atomic i32 %s, i32* %s seq_cst", val, ptr)
		}
	case "AddInt32":
		if len(args) >= 2 {
			ptr := g.emitExpr(args[0])
			val := g.emitExpr(args[1])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = atomicrmw add i32* %s, i32 %s seq_cst", tmp, ptr, val)
			return tmp
		}
	case "SwapInt32":
		if len(args) >= 2 {
			ptr := g.emitExpr(args[0])
			val := g.emitExpr(args[1])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = atomicrmw xchg i32* %s, i32 %s seq_cst", tmp, ptr, val)
			return tmp
		}
	case "CompareAndSwapInt32", "CompareAndSwapUint32":
		if len(args) >= 3 {
			ptr := g.emitExpr(args[0])
			oldVal := g.emitExpr(args[1])
			newVal := g.emitExpr(args[2])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = cmpxchg i32* %s, i32 %s, i32 %s seq_cst seq_cst",
				tmp, ptr, oldVal, newVal)
			return tmp
		}
	}
	return ""
}

func (g *Generator) emitUnsafeCall(name string, args []parser.Expr) string {
	switch name {
	case "Pointer":
		if len(args) >= 1 {
			val := g.emitExpr(args[0])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = inttoptr i64 %s to i8*", tmp, val)
			return tmp
		}
	case "PtrToInt":
		if len(args) >= 1 {
			val := g.emitExpr(args[0])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = ptrtoint i8* %s to i64", tmp, val)
			return tmp
		}
	case "Sizeof":
		if len(args) >= 1 {
			typ := g.tr.exprType(args[0])
			return fmt.Sprintf("%d", typ.Size())
		}
	case "Offsetof":
		return "0"
	case "Alignof":
		return "8"
	case "Slice":
		if len(args) >= 2 {
			ptr := g.emitExpr(args[0])
			length := g.emitExpr(args[1])
			tmp := g.emitter.newTmp()
			g.emitter.emitf("%s = insertvalue { i8*, i64 } undef, i8* %s, 0", tmp, ptr)
			tmp2 := g.emitter.newTmp()
			g.emitter.emitf("%s = insertvalue { i8*, i64 } %s, i64 %s, 1", tmp2, tmp, length)
			return tmp2
		}
	}
	return "0"
}