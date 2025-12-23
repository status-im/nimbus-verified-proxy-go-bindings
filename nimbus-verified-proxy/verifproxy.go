//go:build !lint

package verifproxy

/*
	#include <verifproxy.h>
	#include <stdio.h>
	#include <stdlib.h>
	#include <string.h>

	extern void goCallbackWrapper(Context *ctx, int status, char *result);
	extern void goStartCallbackWrapper(Context *ctx, int status, char *result);

	// Use weak linkage to allow duplicate symbols (linker will use one)
	// This works around CGO splitting C code into multiple object files
	__attribute__((weak)) void cGoCallback(Context *ctx, int status, char *result) {
		goCallbackWrapper(ctx, status, result);
	}

	__attribute__((weak)) void cGoStartCallback(Context *ctx, int status, char *result) {
		goStartCallbackWrapper(ctx, status, result);
	}

*/
import "C"
import (
	"errors"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

var (
	errEmptyContext = errors.New("empty context")
	errTimeout      = errors.New("request timeout")
	nimMainCalled   bool
	nimMainMu       sync.Mutex
)

// initNimMain ensures NimMain is called once before any C API calls
func initNimMain() {
	nimMainMu.Lock()
	defer nimMainMu.Unlock()
	if !nimMainCalled {
		C.NimMain()
		nimMainCalled = true
	}
}

//export goStartCallbackWrapper
func goStartCallbackWrapper(ctx *C.Context, status C.int, result *C.char) {
	ctxPtr := unsafe.Pointer(ctx)
	goCtx := getContextFromPtr(ctxPtr)
	if goCtx == nil {
		return
	}

	resultStr := ""
	if result != nil {
		resultStr = C.GoString(result)
		// Free the result string as per the C API documentation
		C.freeResponse(result)
	}

	// Invoke the onStart callback if it was provided
	goCtx.mu.Lock()
	onStart := goCtx.onStartCallback
	goCtx.mu.Unlock()

	if onStart != nil {
		onStart(goCtx, Status(status), resultStr)
	}
}

//export goCallbackWrapper
func goCallbackWrapper(ctx *C.Context, status C.int, result *C.char) {
	ctxPtr := unsafe.Pointer(ctx)
	goCtx := getContextFromPtr(ctxPtr)
	if goCtx == nil {
		return
	}

	resultStr := ""
	if result != nil {
		resultStr = C.GoString(result)
		// Free the result string as per the C API documentation
		C.freeResponse(result)
	}

	// Find pending callback for this context
	// Note: We only support one pending callback at a time per context
	goCtx.mu.Lock()
	pending, ok := goCtx.pendingCallbacks[ctxPtr]
	if ok {
		delete(goCtx.pendingCallbacks, ctxPtr)
	}
	goCtx.mu.Unlock()

	if ok && pending != nil {
		pending(goCtx, Status(status), resultStr)
	}
}

// StartVerifProxy starts the verification proxy with a given configuration.
// configJson: JSON string describing the configuration for the verification proxy.
// onStart: Callback invoked once the proxy has started. NOTE: The callback is invoked
// only on error otherwise the proxy runs indefinitely
//
// Returns a new Context object representing the running proxy.
func StartVerifProxy(logger *zap.Logger, configJson string, onStart CallbackProc) (*Context, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx := &Context{
		logger:           logger,
		stopChan:         make(chan struct{}),
		pendingCallbacks: make(map[unsafe.Pointer]CallbackProc),
		onStartCallback:  onStart,
	}

	// Initialize Nim runtime
	initNimMain()

	cConfigJson := C.CString(configJson)
	defer C.free(unsafe.Pointer(cConfigJson))

	// Call startVerifProxy with the start callback wrapper
	// The onStart callback will be invoked by the C library if there's an error
	ctxPtr := C.startVerifProxy(cConfigJson, (*[0]byte)(C.cGoStartCallback))
	if ctxPtr == nil {
		return nil, errors.New("failed to start verification proxy")
	}

	ctx.ctx = unsafe.Pointer(ctxPtr)
	registerContext(ctx)

	ctx.logger.Info("successfully started verification proxy")
	return ctx, nil
}

// Stop stops a running verification proxy.
// After calling this, the context is no longer valid and must not be used.
func (ctx *Context) Stop() error {
	if ctx == nil || ctx.ctx == nil {
		return errEmptyContext
	}

	C.stopVerifProxy((*C.Context)(ctx.ctx))
	close(ctx.stopChan)
	unregisterContext(ctx)
	ctx.ctx = nil
	return nil
}

// Free frees the Context object. This should be called when the context is no longer needed.
func (ctx *Context) Free() {
	if ctx == nil || ctx.ctx == nil {
		return
	}

	C.freeContext((*C.Context)(ctx.ctx))
	unregisterContext(ctx)
	ctx.ctx = nil
}

// ProcessTasks processes pending tasks for a running verification proxy.
// This function should be called periodically to allow the proxy to handle
// queued tasks, callbacks, and events. It is non-blocking.
func (ctx *Context) ProcessTasks() {
	if ctx == nil || ctx.ctx == nil {
		return
	}

	C.processVerifProxyTasks((*C.Context)(ctx.ctx))
}

// callAsync makes an asynchronous call to the C library and waits for the callback
func (ctx *Context) callAsync(callFunc func(), timeout time.Duration) (Status, string, error) {
	if ctx == nil || ctx.ctx == nil {
		return RET_ERROR, "", errEmptyContext
	}

	var wg sync.WaitGroup
	var resultStatus Status
	var resultStr string
	var resultErr error

	callback := func(ctx *Context, status Status, result string) {
		resultStatus = status
		resultStr = result
		wg.Done()
	}

	ctx.mu.Lock()
	ctx.pendingCallbacks[ctx.ctx] = callback
	ctx.mu.Unlock()

	wg.Add(1)
	callFunc()

	// Wait for callback with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case <-done:
			// Callback received
			ctx.mu.Lock()
			delete(ctx.pendingCallbacks, ctx.ctx)
			ctx.mu.Unlock()
			return resultStatus, resultStr, resultErr
		case <-time.After(timeout):
			ctx.mu.Lock()
			delete(ctx.pendingCallbacks, ctx.ctx)
			ctx.mu.Unlock()
			return RET_ERROR, "", errTimeout
		default:
			ctx.ProcessTasks()
		}
	}
}

/* ========================================================================== */
/*                               BASIC CHAIN DATA                              */
/* ========================================================================== */

// BlockNumber retrieves the current blockchain head block number.
func (ctx *Context) BlockNumber() (Status, string, error) {
	return ctx.callAsync(func() {
		C.eth_blockNumber((*C.Context)(ctx.ctx), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// BlobBaseFee retrieves the EIP-4844 blob base fee.
func (ctx *Context) BlobBaseFee() (Status, string, error) {
	return ctx.callAsync(func() {
		C.eth_blobBaseFee((*C.Context)(ctx.ctx), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GasPrice retrieves the current gas price.
func (ctx *Context) GasPrice() (Status, string, error) {
	return ctx.callAsync(func() {
		C.eth_gasPrice((*C.Context)(ctx.ctx), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// MaxPriorityFeePerGas retrieves the suggested priority fee per gas.
func (ctx *Context) MaxPriorityFeePerGas() (Status, string, error) {
	return ctx.callAsync(func() {
		C.eth_maxPriorityFeePerGas((*C.Context)(ctx.ctx), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

/* ========================================================================== */
/*                          ACCOUNT & STORAGE ACCESS                           */
/* ========================================================================== */

// GetBalance retrieves an account balance.
// address: 20-byte hex Ethereum address.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetBalance(address, blockTag string) (Status, string, error) {
	cAddress := C.CString(address)
	defer C.free(unsafe.Pointer(cAddress))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getBalance((*C.Context)(ctx.ctx), cAddress, cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetStorageAt retrieves storage from a contract.
// address: 20-byte hex Ethereum address.
// slot: 32-byte hex-encoded storage slot index.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetStorageAt(address, slot, blockTag string) (Status, string, error) {
	cAddress := C.CString(address)
	defer C.free(unsafe.Pointer(cAddress))
	cSlot := C.CString(slot)
	defer C.free(unsafe.Pointer(cSlot))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getStorageAt((*C.Context)(ctx.ctx), cAddress, cSlot, cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetTransactionCount retrieves an address's transaction count (nonce).
// address: 20-byte hex Ethereum address.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetTransactionCount(address, blockTag string) (Status, string, error) {
	cAddress := C.CString(address)
	defer C.free(unsafe.Pointer(cAddress))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getTransactionCount((*C.Context)(ctx.ctx), cAddress, cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetCode retrieves bytecode stored at an address.
// address: 20-byte hex Ethereum address.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetCode(address, blockTag string) (Status, string, error) {
	cAddress := C.CString(address)
	defer C.free(unsafe.Pointer(cAddress))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getCode((*C.Context)(ctx.ctx), cAddress, cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

/* ========================================================================== */
/*                            BLOCK & UNCLE QUERIES                            */
/* ========================================================================== */

// GetBlockByHash retrieves a block by hash.
// blockHash: 32-byte hex encoded block hash.
// fullTransactions: Whether full tx objects should be included.
func (ctx *Context) GetBlockByHash(blockHash string, fullTransactions bool) (Status, string, error) {
	cBlockHash := C.CString(blockHash)
	defer C.free(unsafe.Pointer(cBlockHash))

	return ctx.callAsync(func() {
		C.eth_getBlockByHash((*C.Context)(ctx.ctx), cBlockHash, C.bool(fullTransactions), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetBlockByNumber retrieves a block by number or tag.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
// fullTransactions: Whether full tx objects should be included.
func (ctx *Context) GetBlockByNumber(blockTag string, fullTransactions bool) (Status, string, error) {
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getBlockByNumber((*C.Context)(ctx.ctx), cBlockTag, C.bool(fullTransactions), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetUncleCountByBlockNumber gets the number of uncles in a block.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetUncleCountByBlockNumber(blockTag string) (Status, string, error) {
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getUncleCountByBlockNumber((*C.Context)(ctx.ctx), cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetUncleCountByBlockHash gets the number of uncles in a block.
// blockHash: 32-byte hex encoded block hash.
func (ctx *Context) GetUncleCountByBlockHash(blockHash string) (Status, string, error) {
	cBlockHash := C.CString(blockHash)
	defer C.free(unsafe.Pointer(cBlockHash))

	return ctx.callAsync(func() {
		C.eth_getUncleCountByBlockHash((*C.Context)(ctx.ctx), cBlockHash, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetBlockTransactionCountByNumber gets the number of transactions in a block.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetBlockTransactionCountByNumber(blockTag string) (Status, string, error) {
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getBlockTransactionCountByNumber((*C.Context)(ctx.ctx), cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetBlockTransactionCountByHash gets the number of transactions in a block identified by hash.
// blockHash: 32-byte hex encoded block hash.
func (ctx *Context) GetBlockTransactionCountByHash(blockHash string) (Status, string, error) {
	cBlockHash := C.CString(blockHash)
	defer C.free(unsafe.Pointer(cBlockHash))

	return ctx.callAsync(func() {
		C.eth_getBlockTransactionCountByHash((*C.Context)(ctx.ctx), cBlockHash, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

/* ========================================================================== */
/*                           TRANSACTION QUERIES                               */
/* ========================================================================== */

// GetTransactionByBlockNumberAndIndex retrieves a transaction in a block by index.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
// index: Zero-based transaction index.
func (ctx *Context) GetTransactionByBlockNumberAndIndex(blockTag string, index uint64) (Status, string, error) {
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getTransactionByBlockNumberAndIndex((*C.Context)(ctx.ctx), cBlockTag, C.ulonglong(index), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetTransactionByBlockHashAndIndex retrieves a transaction by block hash and index.
// blockHash: 32-byte hex encoded block hash.
// index: Zero-based transaction index.
func (ctx *Context) GetTransactionByBlockHashAndIndex(blockHash string, index uint64) (Status, string, error) {
	cBlockHash := C.CString(blockHash)
	defer C.free(unsafe.Pointer(cBlockHash))

	return ctx.callAsync(func() {
		C.eth_getTransactionByBlockHashAndIndex((*C.Context)(ctx.ctx), cBlockHash, C.ulonglong(index), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetTransactionByHash retrieves a transaction by hash.
// txHash: 32-byte hex encoded transaction hash.
func (ctx *Context) GetTransactionByHash(txHash string) (Status, string, error) {
	cTxHash := C.CString(txHash)
	defer C.free(unsafe.Pointer(cTxHash))

	return ctx.callAsync(func() {
		C.eth_getTransactionByHash((*C.Context)(ctx.ctx), cTxHash, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetTransactionReceipt retrieves a transaction receipt by hash.
// txHash: 32-byte hex encoded transaction hash.
func (ctx *Context) GetTransactionReceipt(txHash string) (Status, string, error) {
	cTxHash := C.CString(txHash)
	defer C.free(unsafe.Pointer(cTxHash))

	return ctx.callAsync(func() {
		C.eth_getTransactionReceipt((*C.Context)(ctx.ctx), cTxHash, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

/* ========================================================================== */
/*                          CALL / GAS / ACCESS LISTS                          */
/* ========================================================================== */

// Call executes an eth_call.
// txArgs: JSON encoded string containing call parameters.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
// optimisticStateFetch: Whether optimistic state fetching is allowed.
func (ctx *Context) Call(txArgs, blockTag string, optimisticStateFetch bool) (Status, string, error) {
	cTxArgs := C.CString(txArgs)
	defer C.free(unsafe.Pointer(cTxArgs))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_call((*C.Context)(ctx.ctx), cTxArgs, cBlockTag, C.bool(optimisticStateFetch), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// CreateAccessList generates an EIP-2930 access list.
// txArgs: JSON encoded string containing call parameters.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
// optimisticStateFetch: Whether optimistic state fetching is allowed.
func (ctx *Context) CreateAccessList(txArgs, blockTag string, optimisticStateFetch bool) (Status, string, error) {
	cTxArgs := C.CString(txArgs)
	defer C.free(unsafe.Pointer(cTxArgs))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_createAccessList((*C.Context)(ctx.ctx), cTxArgs, cBlockTag, C.bool(optimisticStateFetch), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// EstimateGas estimates gas for a transaction.
// txArgs: JSON encoded string containing call parameters.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
// optimisticStateFetch: Whether optimistic state fetching is allowed.
func (ctx *Context) EstimateGas(txArgs, blockTag string, optimisticStateFetch bool) (Status, string, error) {
	cTxArgs := C.CString(txArgs)
	defer C.free(unsafe.Pointer(cTxArgs))
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_estimateGas((*C.Context)(ctx.ctx), cTxArgs, cBlockTag, C.bool(optimisticStateFetch), (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

/* ========================================================================== */
/*                               LOGS & FILTERS                                */
/* ========================================================================== */

// GetLogs retrieves logs matching a filter.
// filterOptions: JSON encoded string specifying the log filtering rules.
func (ctx *Context) GetLogs(filterOptions string) (Status, string, error) {
	cFilterOptions := C.CString(filterOptions)
	defer C.free(unsafe.Pointer(cFilterOptions))

	return ctx.callAsync(func() {
		C.eth_getLogs((*C.Context)(ctx.ctx), cFilterOptions, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// NewFilter creates a new log filter.
// filterOptions: JSON encoded string specifying the log filtering rules.
func (ctx *Context) NewFilter(filterOptions string) (Status, string, error) {
	cFilterOptions := C.CString(filterOptions)
	defer C.free(unsafe.Pointer(cFilterOptions))

	return ctx.callAsync(func() {
		C.eth_newFilter((*C.Context)(ctx.ctx), cFilterOptions, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// UninstallFilter removes an installed filter.
// filterId: filter ID as a hex encoded string (as returned by NewFilter).
func (ctx *Context) UninstallFilter(filterId string) (Status, string, error) {
	cFilterId := C.CString(filterId)
	defer C.free(unsafe.Pointer(cFilterId))

	return ctx.callAsync(func() {
		C.eth_uninstallFilter((*C.Context)(ctx.ctx), cFilterId, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetFilterLogs retrieves all logs for an installed filter.
// filterId: filter ID as a hex encoded string (as returned by NewFilter).
func (ctx *Context) GetFilterLogs(filterId string) (Status, string, error) {
	cFilterId := C.CString(filterId)
	defer C.free(unsafe.Pointer(cFilterId))

	return ctx.callAsync(func() {
		C.eth_getFilterLogs((*C.Context)(ctx.ctx), cFilterId, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// GetFilterChanges retrieves new logs since the previous poll.
// filterId: filter ID as a hex encoded string (as returned by NewFilter).
func (ctx *Context) GetFilterChanges(filterId string) (Status, string, error) {
	cFilterId := C.CString(filterId)
	defer C.free(unsafe.Pointer(cFilterId))

	return ctx.callAsync(func() {
		C.eth_getFilterChanges((*C.Context)(ctx.ctx), cFilterId, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

/* ========================================================================== */
/*                              RECEIPT QUERIES                                */
/* ========================================================================== */

// GetBlockReceipts retrieves all receipts for a block.
// blockTag: A block identifier: "latest", "pending", "earliest", or a hex block number such as "0x10d4f".
func (ctx *Context) GetBlockReceipts(blockTag string) (Status, string, error) {
	cBlockTag := C.CString(blockTag)
	defer C.free(unsafe.Pointer(cBlockTag))

	return ctx.callAsync(func() {
		C.eth_getBlockReceipts((*C.Context)(ctx.ctx), cBlockTag, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}

// SendRawTransaction sends a signed transaction to the RPC provider to be relayed in the network.
// txHexBytes: Hex encoded signed transaction.
func (ctx *Context) SendRawTransaction(txHexBytes string) (Status, string, error) {
	cTxHexBytes := C.CString(txHexBytes)
	defer C.free(unsafe.Pointer(cTxHexBytes))

	return ctx.callAsync(func() {
		C.eth_sendRawTransaction((*C.Context)(ctx.ctx), cTxHexBytes, (*[0]byte)(C.cGoCallback))
	}, requestTimeout)
}
