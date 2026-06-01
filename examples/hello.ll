; Z Language Compiled Output
; LLVM IR

target datalayout = "e-m:e-p:64:64-i64:64-n8:16:32:64-S128"
target triple = "x86_64-unknown-none-elf"

declare void @llvm.memcpy.p0i8.p0i8.i64(i8* nocapture, i8* nocapture, i64, i1) nounwind
declare void @llvm.memset.p0i8.i64(i8* nocapture, i8, i64, i1) nounwind



define internal void @main() {
entry:
  %v.msg = alloca {i8*, i64}
  store {i8*, i64} , {i8*, i64}* %v.msg
  %t1 = load i64, i64* %v.msg
  call void @Print(i64 %t1)
  ret void
}

define internal void @Print({i8*, i64} %msg) {
entry:
  ret void
}

