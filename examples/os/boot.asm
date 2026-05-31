; Multiboot1 header and entry point for Z OS (32-bit)
; QEMU loads with -kernel option

; Multiboot constants
MBOOT_PAGE_ALIGN    equ 1<<0
MBOOT_MEM_INFO      equ 1<<1
MBOOT_MAGIC         equ 0x1BADB002
MBOOT_FLAGS         equ MBOOT_PAGE_ALIGN | MBOOT_MEM_INFO
MBOOT_CHECKSUM      equ -(MBOOT_MAGIC + MBOOT_FLAGS)

section .multiboot
align 4
    dd MBOOT_MAGIC
    dd MBOOT_FLAGS
    dd MBOOT_CHECKSUM

section .text
bits 32
global _start
extern KernelMain

_start:
    ; Set up stack
    mov esp, stack_top

    ; Clear direction flag
    cld

    ; Call Z kernel
    call KernelMain

    ; Halt if kernel returns
    cli
    hlt
    jmp $

section .bss
align 16
stack_bottom:
    resb 16384      ; 16 KB stack
stack_top: