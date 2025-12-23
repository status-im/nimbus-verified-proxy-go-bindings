package verifproxy_test

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	verifproxy "github.com/status-im/nimbus-verified-proxy-go-bindings/nimbus-verified-proxy"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// ValidConfigJSON provides an example configuration JSON for testing
const ValidConfigJSON = `{
	"eth2Network": "mainnet",
	"trustedBlockRoot": "0x14362e96fd93aea71fd6d2bdd368d4b8571b627f5e525d19ca82710bd3cff761",
	"backendUrl": "https://eth.blockrazor.xyz",
	"beaconApiUrls": "http://testing.mainnet.beacon-api.nimbus.team",
	"logLevel": "TRACE",
	"logStdout": "auto"
}`

// InvalidConfigJSON provides an example configuration JSON for testing
const InvalidConfigJSON = `{
	"eth2Network": "mainnet",
	"trustedBlockRoot": "0x2558d82e8b29c4151a0683e4f9d480d229d84b27b51a976f56722e014227e723",
	"backendUrl": "https://invalid.backend.url",
	"beaconApiUrls": "https://invalid.beacon-api.url",
	"logLevel": "TRACE",
	"logStdout": "auto"
}`

// VerifProxyTestSuite is a test suite for the verification proxy API
type VerifProxyTestSuite struct {
	suite.Suite
	ctx    *verifproxy.Context
	logger *zap.Logger

	// Test data shared across tests
	testBlockHash string
	testTxHash    string
	testFilterId  string
}

// SetupSuite runs once before all tests in the suite
func (s *VerifProxyTestSuite) SetupSuite() {
	logger, _ := zap.NewDevelopment()
	s.logger = logger

	ctx, err := verifproxy.StartVerifProxy(logger, ValidConfigJSON, nil)
	s.Require().NoError(err, "Failed to start proxy - ensure NIM_VERIFPROXY_HEADER_PATH and NIM_VERIFPROXY_LIB_PATH are set")
	s.Require().NotNil(ctx)

	s.ctx = ctx
	s.logger.Info("proxy initialization complete")
}

// TearDownSuite runs once after all tests in the suite
func (s *VerifProxyTestSuite) TearDownSuite() {
	if s.ctx != nil {
		s.ctx.Stop()
		s.ctx.Free()
	}
	if s.logger != nil {
		s.logger.Sync()
	}
}

// TestStartStop tests basic lifecycle of starting and stopping the proxy
func TestStartStop(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	ctx, err := verifproxy.StartVerifProxy(logger, ValidConfigJSON, nil)
	require.NoError(t, err, "Failed to start proxy - ensure NIM_VERIFPROXY_HEADER_PATH and NIM_VERIFPROXY_LIB_PATH are set")
	require.NotNil(t, ctx)

	logger.Info("proxy initialization complete")

	ctx.Stop()
	ctx.Free()
}

// TestStartWithCallback tests starting the proxy with an error callback
func TestStartWithCallback(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	var callbackCalled atomic.Bool
	onStart := func(ctx *verifproxy.Context, status verifproxy.Status, result string) {
		callbackCalled.Store(true)
		t.Logf("Start callback invoked with status: %d, result: %s", status, result)
	}

	// Start proxy with invalid config, since onStart callback is only called on error
	ctx, err := verifproxy.StartVerifProxy(logger, InvalidConfigJSON, onStart)
	require.NoError(t, err, "Failed to start proxy - ensure NIM_VERIFPROXY_HEADER_PATH and NIM_VERIFPROXY_LIB_PATH are set")
	require.NotNil(t, ctx)
	defer func() {
		ctx.Stop()
		ctx.Free()
	}()

	// Process tasks to allow the start callback to be invoked
	// The onStart callback is triggered via ProcessTasks when an error occurs during initialization
	tStart := time.Now()
	for {
		ctx.ProcessTasks()
		if callbackCalled.Load() {
			return
		}
		if time.Since(tStart) > 10*time.Second {
			t.Fatalf("Start callback not called after 10 seconds")
		}
	}
}

// TestVerifProxyAPI runs the complete API test suite
func TestVerifProxyAPI(t *testing.T) {
	suite.Run(t, new(VerifProxyTestSuite))
}

// Test methods for the suite - Basic Chain Data

func (s *VerifProxyTestSuite) TestBlockNumber() {
	status, result, err := s.ctx.BlockNumber()
	s.Assert().NoError(err, "BlockNumber call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Block number result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestBlobBaseFee() {
	status, result, err := s.ctx.BlobBaseFee()
	s.Assert().NoError(err, "BlobBaseFee call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Blob base fee result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestGasPrice() {
	status, result, err := s.ctx.GasPrice()
	s.Assert().NoError(err, "GasPrice call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Gas price result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestMaxPriorityFeePerGas() {
	status, result, err := s.ctx.MaxPriorityFeePerGas()
	s.Assert().NoError(err, "MaxPriorityFeePerGas call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Max priority fee per gas result: %s", result)
	s.Assert().NotEmpty(result)
}

// Account & Storage Access

func (s *VerifProxyTestSuite) TestGetBalance() {
	address := "0x0000000000000000000000000000000000000000"
	blockTag := "latest"

	status, result, err := s.ctx.GetBalance(address, blockTag)
	s.Assert().NoError(err, "GetBalance call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Balance result: %s", result)
}

func (s *VerifProxyTestSuite) TestGetStorageAt() {
	// USDT contract address and storage slot 0
	address := "0xdac17f958d2ee523a2206206994597c13d831ec7"
	slot := "0x0000000000000000000000000000000000000000000000000000000000000000"
	blockTag := "latest"

	status, result, err := s.ctx.GetStorageAt(address, slot, blockTag)
	s.Assert().NoError(err, "GetStorageAt call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Storage result: %s", result)
}

func (s *VerifProxyTestSuite) TestGetTransactionCount() {
	// Vitalik's address
	address := "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
	blockTag := "latest"

	status, result, err := s.ctx.GetTransactionCount(address, blockTag)
	s.Assert().NoError(err, "GetTransactionCount call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Transaction count result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestGetCode() {
	// USDT contract address
	address := "0xdac17f958d2ee523a2206206994597c13d831ec7"
	blockTag := "latest"

	status, result, err := s.ctx.GetCode(address, blockTag)
	s.Assert().NoError(err, "GetCode call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Code result length: %d bytes", len(result))
	s.Assert().NotEmpty(result)
	s.Assert().True(len(result) > 2, "Code should have content beyond 0x prefix")
}

// Block & Uncle Queries

func (s *VerifProxyTestSuite) TestGetBlockByNumber() {
	status, result, err := s.ctx.GetBlockByNumber("latest", false)
	s.Assert().NoError(err, "GetBlockByNumber call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Block result: %s", result)
	var blockData map[string]interface{}
	err = json.Unmarshal([]byte(result), &blockData)
	s.Assert().NoError(err, "Failed to parse block JSON")
	s.T().Logf("Successfully parsed block JSON")
	// Store block hash for other tests
	if hash, ok := blockData["hash"].(string); ok {
		s.testBlockHash = hash
	}
}

func (s *VerifProxyTestSuite) TestGetBlockByHash() {
	// First ensure we have a block hash
	if s.testBlockHash == "" {
		status, result, err := s.ctx.GetBlockByNumber("latest", false)
		s.Assert().NoError(err)
		s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Failed to get latest block")

		var blockData map[string]interface{}
		err = json.Unmarshal([]byte(result), &blockData)
		s.Assert().NoError(err)
		s.testBlockHash = blockData["hash"].(string)
	}

	status, result, err := s.ctx.GetBlockByHash(s.testBlockHash, false)
	s.Assert().NoError(err, "GetBlockByHash call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Block by hash result: hash=%s", s.testBlockHash)
	var block map[string]interface{}
	err = json.Unmarshal([]byte(result), &block)
	s.Assert().NoError(err, "Failed to parse block JSON")
	s.Assert().Equal(s.testBlockHash, block["hash"].(string))
}

func (s *VerifProxyTestSuite) TestGetUncleCountByBlockNumber() {
	status, result, err := s.ctx.GetUncleCountByBlockNumber("latest")
	s.Assert().NoError(err, "GetUncleCountByBlockNumber call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Uncle count by number result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestGetUncleCountByBlockHash() {
	if s.testBlockHash == "" {
		s.T().Skip("Skipping: no block hash available")
	}
	status, result, err := s.ctx.GetUncleCountByBlockHash(s.testBlockHash)
	s.Assert().NoError(err, "GetUncleCountByBlockHash call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Uncle count by hash result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestGetBlockTransactionCountByNumber() {
	status, result, err := s.ctx.GetBlockTransactionCountByNumber("latest")
	s.Assert().NoError(err, "GetBlockTransactionCountByNumber call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Block transaction count by number result: %s", result)
	s.Assert().NotEmpty(result)
}

func (s *VerifProxyTestSuite) TestGetBlockTransactionCountByHash() {
	if s.testBlockHash == "" {
		s.T().Skip("Skipping: no block hash available")
	}
	status, result, err := s.ctx.GetBlockTransactionCountByHash(s.testBlockHash)
	s.Assert().NoError(err, "GetBlockTransactionCountByHash call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Block transaction count by hash result: %s", result)
	s.Assert().NotEmpty(result)
}

// Transaction Queries

func (s *VerifProxyTestSuite) TestGetTransactionByBlockNumberAndIndex() {
	status, result, err := s.ctx.GetTransactionByBlockNumberAndIndex("latest", 0)
	s.Assert().NoError(err, "GetTransactionByBlockNumberAndIndex call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Transaction by block number and index result")
	var txData map[string]interface{}
	err = json.Unmarshal([]byte(result), &txData)
	s.Assert().NoError(err, "Failed to parse transaction JSON")
	if txData["hash"] != nil {
		s.testTxHash = txData["hash"].(string)
		s.T().Logf("Got transaction hash: %s", s.testTxHash)
	}
}

func (s *VerifProxyTestSuite) TestGetTransactionByBlockHashAndIndex() {
	if s.testBlockHash == "" {
		s.T().Skip("Skipping: no block hash available")
	}
	status, result, err := s.ctx.GetTransactionByBlockHashAndIndex(s.testBlockHash, 0)
	s.Assert().NoError(err, "GetTransactionByBlockHashAndIndex call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Transaction by block hash and index result")
	var txData map[string]interface{}
	err = json.Unmarshal([]byte(result), &txData)
	s.Assert().NoError(err, "Failed to parse transaction JSON")
	s.T().Logf("Successfully parsed transaction JSON")
}

func (s *VerifProxyTestSuite) TestGetTransactionByHash() {
	if s.testTxHash == "" {
		s.T().Skip("Skipping: no transaction hash available from previous test")
	}
	status, result, err := s.ctx.GetTransactionByHash(s.testTxHash)
	s.Assert().NoError(err, "GetTransactionByHash call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Transaction by hash result")
	var txData map[string]interface{}
	err = json.Unmarshal([]byte(result), &txData)
	s.Assert().NoError(err, "Failed to parse transaction JSON")
	s.Assert().Equal(s.testTxHash, txData["hash"].(string))
}

func (s *VerifProxyTestSuite) TestGetTransactionReceipt() {
	if s.testTxHash == "" {
		s.T().Skip("Skipping: no transaction hash available from previous test")
	}
	status, result, err := s.ctx.GetTransactionReceipt(s.testTxHash)
	s.Assert().NoError(err, "GetTransactionReceipt call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Transaction receipt result")
	var receiptData map[string]interface{}
	err = json.Unmarshal([]byte(result), &receiptData)
	s.Assert().NoError(err, "Failed to parse receipt JSON")
	s.Assert().Equal(s.testTxHash, receiptData["transactionHash"].(string))
}

// Call / Gas / Access Lists

func (s *VerifProxyTestSuite) TestCall() {
	s.T().Skip("Skipping: Implement proper test")
	txArgs := `{
		"to": "0x0000000000000000000000000000000000000000",
		"data": "0x"
	}`
	blockTag := "latest"

	status, result, err := s.ctx.Call(txArgs, blockTag, false)
	s.Assert().NoError(err, "Call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Call result: %s", result)
}

func (s *VerifProxyTestSuite) TestCreateAccessList() {
	txArgs := `{
		"to": "0xdac17f958d2ee523a2206206994597c13d831ec7",
		"data": "0x70a08231000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045"
	}`
	blockTag := "latest"

	status, result, err := s.ctx.CreateAccessList(txArgs, blockTag, false)
	s.Assert().NoError(err, "CreateAccessList call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Access list result")
	var accessList map[string]interface{}
	err = json.Unmarshal([]byte(result), &accessList)
	s.Assert().NoError(err, "Failed to parse access list JSON")
	s.T().Logf("Successfully parsed access list JSON")
}

func (s *VerifProxyTestSuite) TestEstimateGas() {
	txArgs := `{
		"from": "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		"to": "0x0000000000000000000000000000000000000000",
		"value": "0x1"
	}`
	blockTag := "latest"

	status, result, err := s.ctx.EstimateGas(txArgs, blockTag, false)
	s.Assert().NoError(err, "EstimateGas call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Estimated gas result: %s", result)
	s.Assert().NotEmpty(result)
}

// Logs & Filters

func (s *VerifProxyTestSuite) TestGetLogs() {
	filterOptions := `{
		"fromBlock": "latest",
		"toBlock": "latest"
	}`

	status, result, err := s.ctx.GetLogs(filterOptions)
	s.Assert().NoError(err, "GetLogs call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Logs result: %s", result)
}

func (s *VerifProxyTestSuite) TestNewFilter() {
	filterOptions := `{
		"fromBlock": "latest",
		"toBlock": "latest"
	}`

	status, result, err := s.ctx.NewFilter(filterOptions)
	s.Assert().NoError(err, "NewFilter call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.testFilterId = result
	s.T().Logf("New filter ID: %s", s.testFilterId)
	s.Assert().NotEmpty(s.testFilterId)
}

func (s *VerifProxyTestSuite) TestGetFilterLogs() {
	if s.testFilterId == "" {
		s.T().Skip("Skipping: no filter ID available from NewFilter test")
	}
	status, result, err := s.ctx.GetFilterLogs(s.testFilterId)
	s.Assert().NoError(err, "GetFilterLogs call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Filter logs result")
	var logs []interface{}
	err = json.Unmarshal([]byte(result), &logs)
	s.Assert().NoError(err, "Failed to parse filter logs JSON")
	s.T().Logf("Successfully parsed filter logs JSON, got %d logs", len(logs))
}

func (s *VerifProxyTestSuite) TestGetFilterChanges() {
	if s.testFilterId == "" {
		s.T().Skip("Skipping: no filter ID available from NewFilter test")
	}
	status, result, err := s.ctx.GetFilterChanges(s.testFilterId)
	s.Assert().NoError(err, "GetFilterChanges call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Filter changes result")
	var changes []interface{}
	err = json.Unmarshal([]byte(result), &changes)
	s.Assert().NoError(err, "Failed to parse filter changes JSON")
	s.T().Logf("Successfully parsed filter changes JSON, got %d changes", len(changes))
}

func (s *VerifProxyTestSuite) TestUninstallFilter() {
	if s.testFilterId == "" {
		s.T().Skip("Skipping: no filter ID available from NewFilter test")
	}
	status, result, err := s.ctx.UninstallFilter(s.testFilterId)
	s.Assert().NoError(err, "UninstallFilter call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Uninstall filter result: %s", result)
}

// Receipt Queries

func (s *VerifProxyTestSuite) TestGetBlockReceipts() {
	status, result, err := s.ctx.GetBlockReceipts("latest")
	s.Assert().NoError(err, "GetBlockReceipts call failed")
	s.Assert().Equal(verifproxy.RET_SUCCESS, status, "Expected RET_SUCCESS, got status: %d, result: %s", status, result)
	s.T().Logf("Block receipts result")
	var receipts []interface{}
	err = json.Unmarshal([]byte(result), &receipts)
	s.Assert().NoError(err, "Failed to parse block receipts JSON")
	s.T().Logf("Successfully parsed block receipts JSON, got %d receipts", len(receipts))
}

// Note: SendRawTransaction is intentionally omitted as it requires a valid signed transaction
// and would actually broadcast to the network, which is not suitable for a basic test suite
