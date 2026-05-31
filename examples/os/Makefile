ZC = ../../bin/zc
LLC = llc
LD = ld.lld
NASM = nasm

all: kernel.elf

boot.o: boot.asm
	$(NASM) -f elf32 -o $@ $<

%.ll: %.z
	$(ZC) -emit-llvm -o $@ $<

%.o: %.ll
	$(LLC) -filetype=obj -mtriple=i386-none-elf -o $@ $<

kernel.elf: boot.o main.o linker.ld
	$(LD) -T linker.ld -o $@ boot.o main.o

run: kernel.elf
	qemu-system-x86_64 -kernel kernel.elf -vga std -no-reboot

debug: kernel.elf
	qemu-system-x86_64 -kernel kernel.elf -s -S &
	sleep 1
	gdb -ex "target remote :1234" -ex "break KernelMain" -ex "continue"

clean:
	rm -f *.o *.ll kernel.elf