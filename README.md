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
2. Navigate to the module's directory:
   ```
   cd $(go list -m -f '{{.Dir}}' github.com/status-im/nimbus-verified-proxy-go-bindings)
   ```
3. Prepare third_party directory which will clone `nimbus-eth1`
   ```
   sudo mkdir third_party
   sudo chown $USER third_party
   ```
4. Build the dependencies:
   ```
   make -C nimbus-verified-proxy
   ```

Now the module is ready for use in your project.

### Note

In order to easily build the libnimbus-verified-proxy library on demand, it is recommended to add the following target in your project's Makefile:

```
LIBVERIFPROXY_DEP_PATH=$(shell go list -m -f '{{.Dir}}' github.com/status-im/nimbus-verified-proxy-go-bindings)

buildlib:
   cd $(LIBVERIFPROXY_DEP_PATH) &&\
   sudo mkdir -p third_party &&\
   sudo chown $(USER) third_party &&\
   make -C nimbus-verified-proxy
```
