; Z Language Compiled Output
; LLVM IR

target datalayout = "e-m:e-p:64:64-i64:64-n8:16:32:64-S128"
target triple = "x86_64-unknown-none-elf"

declare void @llvm.memcpy.p0i8.p0i8.i64(i8* nocapture, i8* nocapture, i64, i1) nounwind
declare void @llvm.memset.p0i8.i64(i8* nocapture, i8, i64, i1) nounwind



define internal void @main() {
entry:
  %v.a = alloca i32
  store i32 10, i32* %v.a
  %v.b = alloca i32
  store i32 20, i32* %v.b
  %v.sum = alloca i32
  %t1 = load i64, i64* %v.a
  %t2 = load i64, i64* %v.b
  %t3 = add i64 %t1, %t2
  store i32 %t3, i32* %v.sum
  %v.diff = alloca i32
  %t4 = load i64, i64* %v.a
  %t5 = load i64, i64* %v.b
  %t6 = sub i64 %t4, %t5
  store i32 %t6, i32* %v.diff
  %v.prod = alloca i32
  %t7 = load i64, i64* %v.a
  %t8 = load i64, i64* %v.b
  %t9 = mul i64 %t7, %t8
  store i32 %t9, i32* %v.prod
  ret void
}

define internal i32 @compute(i32 %x, i32 %y) {
entry:
  %t10 = load i64, i64* %v.x
  %t11 = load i64, i64* %v.y
  %t12 = add i64 %t10, %t11
  ret i64 %t12
}

