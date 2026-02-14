//go:build !lint

package verifproxy

/*
	#include <verifproxy.h>
	#include <stdio.h>
	#include <stdlib.h>
	#include <string.h>

	extern void goCallbackWrapper(Context *ctx, int status, char *result, void *userData);

	extern void goTransportWrapper(Context *ctx, char *url, char *name, char *params, CallBackProc cb, void *userData);

	// Use weak linkage to allow duplicate symbols (linker will use one)
	// This works around CGO splitting C code into multiple object files
	__attribute__((weak)) void cGoCallback(Context *ctx, int status, char *result, void *userData) {
		goCallbackWrapper(ctx, status, result, userData);
	}

	// Use weak linkage to allow duplicate symbols (linker will use one)
	// This works around CGO splitting C code into multiple object files
	__attribute__((weak)) void cGoTransport(Context *ctx, char *url, char *name, char *params, CallBackProc cb, void *userData) {
		goTransportWrapper(ctx, url, name, params, cb, userData);
	}

	static inline void callCCallback(Context *ctx, int status, char *result, void *userData, CallBackProc cb) {
		cb(ctx, status, result, userData);
	}

	static inline Context *callStartVerifProxy(char *configJson, TransportProc transport, CallBackProc onStart, uintptr_t userData) {
		return startVerifProxy(configJson, transport, onStart, (void *)userData);
	}

	static inline void callNvp(Context *ctx, char *method, char *params, CallBackProc cb, uintptr_t userData) {
		nvp_call(ctx, method, params, cb, (void *)userData);
	}

*/
import "C"
import (
	"errors"
	"fmt"
	"runtime"
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

const requestTimeout = 5 * time.Second

type VerifyProxyResult struct {
	status   int
	response string
}

type VerifyProxyCallArgs struct {
	method     string
	params     string
	resultChan chan VerifyProxyResult
}

// Context represents an opaque execution context managed on the Nim side
type Context struct {
	ctxPtr          *C.Context
	logger          *zap.Logger
	stopChan        chan VerifyProxyResult
	executeTaskChan chan VerifyProxyCallArgs
}

var (
	errEmptyContext = errors.New("empty context")
	errTimeout      = errors.New("request timeout")
	startOnce       sync.Once
)

//export goCallbackWrapper
func goCallbackWrapper(ctx *C.Context, status C.int, result *C.char, userData unsafe.Pointer) {
	h := cgo.Handle(userData)
	defer h.Delete()

	ch := h.Value().(chan VerifyProxyResult)
	resultStr := C.GoString(result)
	C.freeNimAllocatedString(result)
	statusInt := int(status)

	ch <- VerifyProxyResult{status: statusInt, response: resultStr}
}

//export goTransportWrapper
func goTransportWrapper(ctx *C.Context, url *C.char, method *C.char, params *C.char, cb C.CallBackProc, userData unsafe.Pointer) {
	resp, err := SendRPC(C.GoString(url), C.GoString(method), C.GoString(params))
	defer C.freeNimAllocatedString(url)
	defer C.freeNimAllocatedString(params)

	if err == nil {
		cResult := C.CString(string(resp))
		defer C.free(unsafe.Pointer(cResult))

		C.callCCallback(ctx, C.RET_SUCCESS, cResult, userData, cb)
	} else {
		cError := C.CString(err.Error())
		defer C.free(unsafe.Pointer(cError))

		C.callCCallback(ctx, C.RET_ERROR, cError, userData, cb)
	}

}

// Stop stops a running verification proxy.
// After calling this, the context is no longer valid and must not be used.
func (ctx *Context) Stop() error {
	if ctx == nil {
		return errEmptyContext
	}

	ctx.stopChan <- VerifyProxyResult{status: RET_CANCELLED, response: "cancelled by user"}
	ctx.logger.Info("stopped verification proxy")
	return nil
}

// Start starts the verification proxy with a given configuration.
// configJson: JSON string describing the configuration for the verification proxy.
// onStart: Callback invoked once the proxy has started. NOTE: The callback is invoked
// only on error otherwise the proxy runs indefinitely
//
// Returns a new Context object representing the running proxy.
func Start(logger *zap.Logger, configJson string) (*Context, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	goCtx := &Context{
		logger:          logger,
		stopChan:        make(chan VerifyProxyResult, 1),    // buffered because there are two senders, Stop and callback for startVerifProxy
		executeTaskChan: make(chan VerifyProxyCallArgs, 64), //Task queue
	}

	// Initialize Nim runtime
	startOnce.Do(func() {
		C.NimMain()
	})

	cConfigJson := C.CString(configJson)
	defer C.free(unsafe.Pointer(cConfigJson))

	var wg sync.WaitGroup
	wg.Add(1)

	// start the event loop
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		// Call startVerifProxy with the start callback wrapper
		// The onStart callback will be invoked by the C library if there's an error

		h := cgo.NewHandle(goCtx.stopChan)
		defer h.Delete()

		ctxPtr := C.callStartVerifProxy(cConfigJson, (*[0]byte)(C.cGoTransport), (*[0]byte)(C.cGoCallback), C.uintptr_t(h))
		if ctxPtr == nil {
			goCtx.logger.Error("failed to start verification proxy")
		}
		goCtx.logger.Info("successfully started verification proxy")
		wg.Done()

	loop:
		for {
			select {

			case callArgs := <-goCtx.executeTaskChan:
				h := cgo.NewHandle(callArgs.resultChan)

				cMethod := C.CString(callArgs.method)
				cParams := C.CString(callArgs.params)

				C.callNvp(ctxPtr, cMethod, cParams, (*[0]byte)(C.cGoCallback), C.uintptr_t(h))
				C.free(unsafe.Pointer(cMethod))
				C.free(unsafe.Pointer(cParams))
			case startErrRes := <-goCtx.stopChan:
				goCtx.logger.Error(startErrRes.response)
				C.stopVerifProxy(ctxPtr)
				C.freeContext(ctxPtr)
				break loop
			default:
				C.processVerifProxyTasks(ctxPtr)
			}
		}

		goCtx.logger.Info("Stopping Verified Proxy Event Loop")

	}()

	wg.Wait()
	return goCtx, nil
}

// callAsync makes an asynchronous call to the C library and waits for the callback
func (ctx *Context) CallRpc(method string, params string, timeout time.Duration) (string, error) {
	if ctx == nil {
		return "", errEmptyContext
	}

	if timeout <= 0 {
		timeout = requestTimeout
	}

	resultChan := make(chan VerifyProxyResult, 0)
	ctx.executeTaskChan <- VerifyProxyCallArgs{method: method, params: params, resultChan: resultChan}

	select {

	case proxyResult := <-resultChan:
		if proxyResult.status == RET_SUCCESS {
			return proxyResult.response, nil
		} else {
			return "", errors.New(proxyResult.response)
		}
	case <-time.After(timeout):
		return "", errors.New("request timed out")
	}

}

/* ========================================================================== */
/*                               BASIC CHAIN DATA                              */
/* ========================================================================== */

func (ctx *Context) BlockNumber() (string, error) {
	return ctx.CallRpc("eth_blockNumber", "[]", 0)
}

func (ctx *Context) BlobBaseFee() (string, error) {
	return ctx.CallRpc("eth_blobBaseFee", "[]", 0)
}

func (ctx *Context) GasPrice() (string, error) {
	return ctx.CallRpc("eth_gasPrice", "[]", 0)
}

func (ctx *Context) MaxPriorityFeePerGas() (string, error) {
	return ctx.CallRpc("eth_maxPriorityFeePerGas", "[]", 0)
}

/* ========================================================================== */
/*                          ACCOUNT & STORAGE ACCESS                           */
/* ========================================================================== */

func (ctx *Context) GetBalance(address, blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s","%s"]`, address, blockTag)
	return ctx.CallRpc("eth_getBalance", params, 0)
}

func (ctx *Context) GetStorageAt(address, slot, blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s","%s","%s"]`, address, slot, blockTag)
	return ctx.CallRpc("eth_getStorageAt", params, 0)
}

func (ctx *Context) GetTransactionCount(address, blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s","%s"]`, address, blockTag)
	return ctx.CallRpc("eth_getTransactionCount", params, 0)
}

func (ctx *Context) GetCode(address, blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s","%s"]`, address, blockTag)
	return ctx.CallRpc("eth_getCode", params, 0)
}

/* ========================================================================== */
/*                            BLOCK & UNCLE QUERIES                            */
/* ========================================================================== */

func (ctx *Context) GetBlockByHash(blockHash string, fullTx bool) (string, error) {
	params := fmt.Sprintf(`["%s",%t]`, blockHash, fullTx)
	return ctx.CallRpc("eth_getBlockByHash", params, 0)
}

func (ctx *Context) GetBlockByNumber(blockTag string, fullTx bool) (string, error) {
	params := fmt.Sprintf(`["%s",%t]`, blockTag, fullTx)
	return ctx.CallRpc("eth_getBlockByNumber", params, 0)
}

func (ctx *Context) GetUncleCountByBlockNumber(blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, blockTag)
	return ctx.CallRpc("eth_getUncleCountByBlockNumber", params, 0)
}

func (ctx *Context) GetUncleCountByBlockHash(blockHash string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, blockHash)
	return ctx.CallRpc("eth_getUncleCountByBlockHash", params, 0)
}

func (ctx *Context) GetBlockTransactionCountByNumber(blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, blockTag)
	return ctx.CallRpc("eth_getBlockTransactionCountByNumber", params, 0)
}

func (ctx *Context) GetBlockTransactionCountByHash(blockHash string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, blockHash)
	return ctx.CallRpc("eth_getBlockTransactionCountByHash", params, 0)
}

/* ========================================================================== */
/*                           TRANSACTION QUERIES                               */
/* ========================================================================== */

func (ctx *Context) GetTransactionByBlockNumberAndIndex(blockTag string, index uint64) (string, error) {
	params := fmt.Sprintf(`["%s","0x%x"]`, blockTag, index)
	return ctx.CallRpc("eth_getTransactionByBlockNumberAndIndex", params, 0)
}

func (ctx *Context) GetTransactionByBlockHashAndIndex(blockHash string, index uint64) (string, error) {
	params := fmt.Sprintf(`["%s","0x%x"]`, blockHash, index)
	return ctx.CallRpc("eth_getTransactionByBlockHashAndIndex", params, 0)
}

func (ctx *Context) GetTransactionByHash(txHash string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, txHash)
	return ctx.CallRpc("eth_getTransactionByHash", params, 0)
}

func (ctx *Context) GetTransactionReceipt(txHash string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, txHash)
	return ctx.CallRpc("eth_getTransactionReceipt", params, 0)
}

/* ========================================================================== */
/*                          CALL / GAS / ACCESS LISTS                          */
/* ========================================================================== */

func (ctx *Context) Call(txArgs, blockTag string, optimistic bool) (string, error) {
	params := fmt.Sprintf(`[%s,"%s",%t]`, txArgs, blockTag, optimistic)
	return ctx.CallRpc("eth_call", params, 0)
}

func (ctx *Context) CreateAccessList(txArgs, blockTag string, optimistic bool) (string, error) {
	params := fmt.Sprintf(`[%s,"%s",%t]`, txArgs, blockTag, optimistic)
	return ctx.CallRpc("eth_createAccessList", params, 0)
}

func (ctx *Context) EstimateGas(txArgs, blockTag string, optimistic bool) (string, error) {
	params := fmt.Sprintf(`[%s,"%s",%t]`, txArgs, blockTag, optimistic)
	return ctx.CallRpc("eth_estimateGas", params, 0)
}

/* ========================================================================== */
/*                               LOGS & FILTERS                                */
/* ========================================================================== */

func (ctx *Context) GetLogs(filterOptions string) (string, error) {
	params := fmt.Sprintf(`[%s]`, filterOptions)
	return ctx.CallRpc("eth_getLogs", params, 0)
}

func (ctx *Context) NewFilter(filterOptions string) (string, error) {
	params := fmt.Sprintf(`[%s]`, filterOptions)
	return ctx.CallRpc("eth_newFilter", params, 0)
}

func (ctx *Context) UninstallFilter(filterId string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, filterId)
	return ctx.CallRpc("eth_uninstallFilter", params, 0)
}

func (ctx *Context) GetFilterLogs(filterId string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, filterId)
	return ctx.CallRpc("eth_getFilterLogs", params, 0)
}

func (ctx *Context) GetFilterChanges(filterId string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, filterId)
	return ctx.CallRpc("eth_getFilterChanges", params, 0)
}

/* ========================================================================== */
/*                              RECEIPT QUERIES                                */
/* ========================================================================== */

func (ctx *Context) GetBlockReceipts(blockTag string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, blockTag)
	return ctx.CallRpc("eth_getBlockReceipts", params, 0)
}

func (ctx *Context) SendRawTransaction(txHex string) (string, error) {
	params := fmt.Sprintf(`["%s"]`, txHex)
	return ctx.CallRpc("eth_sendRawTransaction", params, 0)
}
