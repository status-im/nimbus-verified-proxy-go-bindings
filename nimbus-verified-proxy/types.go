package verifproxy

// Status represents the return status from ETH API calls
type Status int

const (
	// RET_SUCCESS indicates the call to eth api frontend was successful
	RET_SUCCESS Status = 0
	// RET_ERROR indicates the call to eth api frontend failed with an error
	RET_ERROR Status = -1
	// RET_CANCELLED indicates the call to the eth api frontend was cancelled
	RET_CANCELLED Status = -2
	// RET_DESER_ERROR indicates an error occurred while deserializing arguments from C to Nim
	RET_DESER_ERROR Status = -3
)

// CallbackProc is the callback function type used for all asynchronous ETH API calls
// status: return code (RET_SUCCESS, RET_ERROR, RET_CANCELLED, or RET_DESER_ERROR)
// result: JSON encoded result string (allocated by Nim - must be freed using freeResponse)
type CallbackProc func(ctx *Context, status Status, result string)
