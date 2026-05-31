# Z Language Makefile

ZC = bin/zc
LLC = llc
LD = ld.lld

.PHONY: all build test examples clean install

all: build

# Build the compiler
build: $(ZC)

$(ZC): cmd/zc/main.go pkg/lexer/*.go pkg/parser/*.go pkg/types/*.go pkg/codegen/*.go
	go build -o $(ZC) ./cmd/zc

# Run basic tests
test: build
	@echo "Testing compiler..."
	$(ZC) -emit-llvm examples/hello.z -o /tmp/hello.ll
	@echo "Basic test passed"

# Build all examples
examples: build
	@for f in examples/*.z; do \
		echo "Compiling $$f"; \
		$(ZC) -emit-llvm $$f; \
	done

# Build OS demo (requires nasm, qemu)
os-demo: build
	$(MAKE) -C examples/os kernel.elf

# Run OS demo in QEMU
run-os: os-demo
	qemu-system-x86_64 -kernel examples/os/kernel.elf -vga std -no-reboot

# Clean build artifacts
clean:
	rm -rf bin/zc
	rm -f examples/*.ll examples/*.o
	$(MAKE) -C examples/os clean

# Install to system (optional)
install: build
	cp $(ZC) /usr/local/bin/zc

# Show help
help:
	@echo "Z Language Build System"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build the zc compiler"
	@echo "  test      - Run basic compiler tests"
	@echo "  examples  - Compile all example programs"
	@echo "  os-demo   - Build the OS kernel demo"
	@echo "  run-os    - Run OS demo in QEMU"
	@echo "  clean     - Remove build artifacts"
	@echo "  install   - Install compiler to /usr/local/bin"