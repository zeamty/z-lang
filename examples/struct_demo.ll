; Z Language Compiled Output
; LLVM IR

target datalayout = "e-m:e-p:64:64-i64:64-n8:16:32:64-S128"
target triple = "x86_64-unknown-none-elf"

declare void @llvm.memcpy.p0i8.p0i8.i64(i8* nocapture, i8* nocapture, i64, i1) nounwind
declare void @llvm.memset.p0i8.i64(i8* nocapture, i8, i64, i1) nounwind


%struct.Point = type {i32, i32}
%struct.Header = type {i8, i16}

define internal void @main() {
entry:
  %v.p = alloca {i32, i32}
  %v.h = alloca {i8, i16}
  ret void
}

define internal {i32, i32} @NewPoint(i32 %x, i32 %y) {
entry:
  %v.p = alloca {i32, i32}
  %t1 = load i64, i64* %v.p
  ret i64 %t1
}

