//go:build lint

package verifproxy

import (
	"errors"

	"go.uber.org/zap"
)

// ErrLintBuild indicates a stubbed, lint-only build without native libverifproxy.
var ErrLintBuild = errors.New("verifproxy: lint-only build stub: native libverifproxy not linked")

// StartVerifProxy returns an error in lint builds.
func StartVerifProxy(logger *zap.Logger, configJson string, onStart CallbackProc) (*Context, error) {
	return nil, ErrLintBuild
}

// Stop returns an error in lint builds.
func (ctx *Context) Stop() error {
	return ErrLintBuild
}

// Free does nothing in lint builds.
func (ctx *Context) Free() {
}

// ProcessTasks does nothing in lint builds.
func (ctx *Context) ProcessTasks() {
}

// BlockNumber returns an error in lint builds.
func (ctx *Context) BlockNumber() (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// BlobBaseFee returns an error in lint builds.
func (ctx *Context) BlobBaseFee() (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GasPrice returns an error in lint builds.
func (ctx *Context) GasPrice() (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// MaxPriorityFeePerGas returns an error in lint builds.
func (ctx *Context) MaxPriorityFeePerGas() (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetBalance returns an error in lint builds.
func (ctx *Context) GetBalance(address, blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetStorageAt returns an error in lint builds.
func (ctx *Context) GetStorageAt(address, slot, blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetTransactionCount returns an error in lint builds.
func (ctx *Context) GetTransactionCount(address, blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetCode returns an error in lint builds.
func (ctx *Context) GetCode(address, blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetBlockByHash returns an error in lint builds.
func (ctx *Context) GetBlockByHash(blockHash string, fullTransactions bool) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetBlockByNumber returns an error in lint builds.
func (ctx *Context) GetBlockByNumber(blockTag string, fullTransactions bool) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetUncleCountByBlockNumber returns an error in lint builds.
func (ctx *Context) GetUncleCountByBlockNumber(blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetUncleCountByBlockHash returns an error in lint builds.
func (ctx *Context) GetUncleCountByBlockHash(blockHash string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetBlockTransactionCountByNumber returns an error in lint builds.
func (ctx *Context) GetBlockTransactionCountByNumber(blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetBlockTransactionCountByHash returns an error in lint builds.
func (ctx *Context) GetBlockTransactionCountByHash(blockHash string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetTransactionByBlockNumberAndIndex returns an error in lint builds.
func (ctx *Context) GetTransactionByBlockNumberAndIndex(blockTag string, index uint64) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetTransactionByBlockHashAndIndex returns an error in lint builds.
func (ctx *Context) GetTransactionByBlockHashAndIndex(blockHash string, index uint64) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetTransactionByHash returns an error in lint builds.
func (ctx *Context) GetTransactionByHash(txHash string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetTransactionReceipt returns an error in lint builds.
func (ctx *Context) GetTransactionReceipt(txHash string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// Call returns an error in lint builds.
func (ctx *Context) Call(txArgs, blockTag string, optimisticStateFetch bool) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// CreateAccessList returns an error in lint builds.
func (ctx *Context) CreateAccessList(txArgs, blockTag string, optimisticStateFetch bool) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// EstimateGas returns an error in lint builds.
func (ctx *Context) EstimateGas(txArgs, blockTag string, optimisticStateFetch bool) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetLogs returns an error in lint builds.
func (ctx *Context) GetLogs(filterOptions string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// NewFilter returns an error in lint builds.
func (ctx *Context) NewFilter(filterOptions string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// UninstallFilter returns an error in lint builds.
func (ctx *Context) UninstallFilter(filterId string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetFilterLogs returns an error in lint builds.
func (ctx *Context) GetFilterLogs(filterId string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetFilterChanges returns an error in lint builds.
func (ctx *Context) GetFilterChanges(filterId string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// GetBlockReceipts returns an error in lint builds.
func (ctx *Context) GetBlockReceipts(blockTag string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}

// SendRawTransaction returns an error in lint builds.
func (ctx *Context) SendRawTransaction(txHexBytes string) (Status, string, error) {
	return RET_ERROR, "", ErrLintBuild
}
