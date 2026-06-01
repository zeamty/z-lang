; Z Language Compiled Output
; LLVM IR

target datalayout = "e-m:e-p:64:64-i64:64-n8:16:32:64-S128"
target triple = "x86_64-unknown-none-elf"

declare void @llvm.memcpy.p0i8.p0i8.i64(i8* nocapture, i8* nocapture, i64, i1) nounwind
declare void @llvm.memset.p0i8.i64(i8* nocapture, i8, i64, i1) nounwind



define internal void @main() {
entry:
  %v.x = alloca i32
  store i32 10, i32* %v.x
  %t1 = load i64, i64* %v.x
  %t2 = icmp sgt i64 %t1, 5
  br i1 %t2, label %L1, label %L2
L1:
  call void @Print({i8*, i64} )
  br label %L3
L2:
  call void @Print({i8*, i64} )
  br label %L3
L3:
  %v.count = alloca i32
  store i32 0, i32* %v.count
  br label %L4
L4:
  %t5 = load i64, i64* %v.count
  %t6 = icmp slt i64 %t5, 10
  br i1 %t6, label %L5, label %L7
L5:
  %t7 = load i64, i64* %v.count
  %t8 = add i64 %t7, 1
  store i64 %t8, i64* %v.count
  br label %L6
L6:
  br label %L4
L7:
  br label %L8
L8:
  br label %L9
L9:
  call void @breakLoop()
  br label %L10
L10:
  br label %L8
L11:
  ret void
}

define internal void @breakLoop() {
entry:
  ret void
}

define internal void @Print({i8*, i64} %msg) {
entry:
  ret void
}

