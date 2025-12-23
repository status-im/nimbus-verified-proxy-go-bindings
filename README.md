# Nimbus Verified Proxy Go Bindings

This repository provides Go bindings for the [Nimbus Verified Proxy](https://github.com/status-im/nimbus-eth1/tree/master/nimbus_verified_proxy) [C library](https://github.com/status-im/nimbus-eth1/tree/master/nimbus_verified_proxy/libverifproxy), enabling seamless integration with Go projects.

## Installation

To build the required dependencies for this module, the `make` command needs to be executed. If you are integrating this module into another project via `go get`, ensure that you navigate to the `nimbus-verified-proxy-go-bindings/nimbus-verified-proxy` directory and run `make`.

### Steps to Install

Follow these steps to install and set up the module:

1. Retrieve the module using `go get`:
   ```
   go get -u github.com/status-im/nimbus-verified-proxy-go-bindings
   ```
2. Define environment variables for libverifproxy dependencies:
   ```
   export NIM_VERIFPROXY_LIB_PATH=/path/to/libverifproxy/lib
   export NIM_VERIFPROXY_HEADER_PATH=/path/to/libverifproxy/header
   export NIM_LIBBACKTRACE_LIB_PATH=/path/to/libbacktrace/and/libbacktracenim/libs
   ```
3. Build the bindings:
   ```
   make -C nimbus-verified-proxy
   ```
Now the module is ready for use in your project.

### Note

In order to easily build the libverifproxy library on demand, it is recommended to add the following target in your project's Makefile:

```
NIMBUS_VERIFPROXY_GO_BINDINGS_DIR=$(shell go list -m -f '{{.Dir}}' github.com/status-im/nimbus-verified-proxy-go-bindings)

NIM_VERIFPROXY_LIB_PATH=${NIMBUS_VERIFPROXY_GO_BINDINGS_DIR}/third_party/nimbus-eth1/build/libverifproxy
NIM_VERIFPROXY_HEADER_PATH=${NIMBUS_VERIFPROXY_GO_BINDINGS_DIR}/third_party/nimbus-eth1/build/libverifproxy
NIM_LIBBACKTRACE_LIB_PATH=${NIMBUS_VERIFPROXY_GO_BINDINGS_DIR}/third_party/nimbus-eth1/vendor/nim-libbacktrace/install/usr/lib

libverifproxy:
   cd $(NIMBUS_VERIFPROXY_GO_BINDINGS_DIR) &&\
   sudo rm -rf third_party &&\
   sudo mkdir -p third_party &&\
   sudo chown $(USER) third_party &&\
   git clone https://github.com/status-im/nimbus-eth1.git third_party/nimbus-eth1 &&\
   make -C third_party/nimbus-eth1 libverifproxy -j10
```

### Testing

All tests:
```
make -C nimbus-verified-proxy test
```

Single test:
```
make -C nimbus-verified-proxy test-single TestBlockNumber
```
