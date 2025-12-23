# Makefile for Nimbus Verified Proxy Go Bindings

# The following variables should be defined beforehand:
# NIM_VERIFPROXY_LIB_PATH: Path to the Nimbus Verified Proxy library
# NIM_VERIFPROXY_HEADER_PATH: Path to the Nimbus Verified Proxy headers
# NIM_LIBBACKTRACE_LIB_PATH: Path to the libbacktrace library
export CGO_CFLAGS=-I${NIM_VERIFPROXY_HEADER_PATH}/
export CGO_LDFLAGS=-L${NIM_VERIFPROXY_LIB_PATH}/ -lverifproxy -L${NIM_LIBBACKTRACE_LIB_PATH}/ -lbacktrace -lbacktracenim -lstdc++ -Wl

# Expected files
NIM_VERIFPROXY_HEADER_FILE := $(NIM_VERIFPROXY_HEADER_PATH)/verifproxy.h
NIM_VERIFPROXY_LIB_FILES   := $(wildcard $(NIM_VERIFPROXY_LIB_PATH)/libverifproxy.*)

NIM_LIBBACKTRACE_LIB_FILES   	:= $(wildcard $(NIM_LIBBACKTRACE_LIB_PATH)/libbacktrace.*)
NIM_LIBBACKTRACE_NIM_LIB_FILES   := $(wildcard $(NIM_LIBBACKTRACE_LIB_PATH)/libbacktracenim.*)

.PHONY: all clean build test

# Default target
all: build

# Validate necessary folders and files
check-folders:
	@echo Checking header directory ...
	@if [ -z "$(NIM_VERIFPROXY_HEADER_PATH)" ]; then \
		echo "ERROR: NIM_VERIFPROXY_HEADER_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(NIM_VERIFPROXY_HEADER_PATH)" ]; then \
		echo "ERROR: Header path does not exist: $(NIM_VERIFPROXY_HEADER_PATH)"; exit 1; \
	fi

	@echo Checking lib directory ...
	@if [ -z "$(NIM_VERIFPROXY_LIB_PATH)" ]; then \
		echo "ERROR: NIM_VERIFPROXY_LIB_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(NIM_VERIFPROXY_LIB_PATH)" ]; then \
		echo "ERROR: Library path does not exist: $(NIM_VERIFPROXY_LIB_PATH)"; exit 1; \
	fi
	@echo Checking libbacktrace directory ...
	@if [ -z "$(NIM_LIBBACKTRACE_LIB_PATH)" ]; then \
		echo "ERROR: NIM_LIBBACKTRACE_LIB_PATH not set"; exit 1; \
	fi
	@if [ ! -d "$(NIM_LIBBACKTRACE_LIB_PATH)" ]; then \
		echo "ERROR: Library path does not exist: $(NIM_LIBBACKTRACE_LIB_PATH)"; exit 1; \
	fi
	@echo Checking for libbacktrace library files ...
	@if [ -z "$(NIM_LIBBACKTRACE_LIB_FILES)" ]; then \
		echo "ERROR: No libbacktrace library file found in: $(NIM_LIBBACKTRACE_LIB_PATH)"; exit 1; \
	fi
	@echo Checking for libbacktracenim library files ...
	@if [ -z "$(NIM_LIBBACKTRACE_NIM_LIB_FILES)" ]; then \
		echo "ERROR: No libbacktracenim library file found in: $(NIM_LIBBACKTRACE_LIB_PATH)"; exit 1; \
	fi

	@echo Checking for verifproxy.h ...
	@if [ ! -f "$(NIM_VERIFPROXY_HEADER_FILE)" ]; then \
		echo "ERROR: verifproxy.h not found at: $(NIM_VERIFPROXY_HEADER_FILE)"; exit 1; \
	fi

	@echo Checking for verifproxy library file ...
	@if [ -z "$(NIM_VERIFPROXY_LIB_FILES)" ]; then \
		echo "ERROR: No verifproxy library file found in: $(NIM_VERIFPROXY_LIB_PATH)"; exit 1; \
	fi

	@echo Checking for libbacktrace library files ...
	@if [ -z "$(NIM_LIBBACKTRACE_LIB_FILES)" ]; then \
		echo "ERROR: No libbacktrace library file found in: $(NIM_LIBBACKTRACE_LIB_PATH)"; exit 1; \
	fi

	@echo Checking for libbacktracenim library files ...
	@if [ -z "$(NIM_LIBBACKTRACE_NIM_LIB_FILES)" ]; then \
		echo "ERROR: No libbacktracenim library file found in: $(NIM_LIBBACKTRACE_LIB_PATH)"; exit 1; \
	fi

# Build VERIFPROXY Go Bindings
build: check-folders
	@echo "Building Nimbus Verified Proxy Go Bindings..."
	go build ./...

# Run tests (requires NIM_VERIFPROXY_HEADER_PATH and NIM_VERIFPROXY_LIB_PATH to be set)
test: check-folders
	@echo "Running tests..."
	go test -v ./...

# Run a specific test
test-single: check-folders
	@echo "Running specific test: $(TEST)"
	@if [ -z "$(TEST)" ]; then \
		echo "ERROR: TEST variable not set. Usage: make test-single TEST=TestName"; exit 1; \
	fi
	go test -v -run $(TEST) ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -f nimbus-verified-proxy-go-bindings
