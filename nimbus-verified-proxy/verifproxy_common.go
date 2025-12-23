package verifproxy

import (
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

const requestTimeout = 30 * time.Second

// Context represents an opaque execution context managed on the Nim side
type Context struct {
	ctx              unsafe.Pointer
	logger           *zap.Logger
	mu               sync.RWMutex
	stopChan         chan struct{}
	pendingCallbacks map[unsafe.Pointer]CallbackProc
	onStartCallback  CallbackProc
}

// The callback sends back the ctx to know to which
// context the callback is being emitted for. Since we only have a global
// callback in the go side, we register all the contexts that we create
// so we can later obtain which instance of `Context` it should
// be invoked depending on the ctx received
var ctxRegistry map[unsafe.Pointer]*Context
var ctxRegistryMu sync.RWMutex

func init() {
	ctxRegistry = make(map[unsafe.Pointer]*Context)
}

func registerContext(ctx *Context) {
	ctxRegistryMu.Lock()
	defer ctxRegistryMu.Unlock()
	ctxRegistry[ctx.ctx] = ctx
}

func unregisterContext(ctx *Context) {
	ctxRegistryMu.Lock()
	defer ctxRegistryMu.Unlock()
	delete(ctxRegistry, ctx.ctx)
}

func getContextFromPtr(ctxPtr unsafe.Pointer) *Context {
	ctxRegistryMu.RLock()
	defer ctxRegistryMu.RUnlock()
	return ctxRegistry[ctxPtr]
}
