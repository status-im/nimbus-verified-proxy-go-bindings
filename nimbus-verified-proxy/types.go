package verifproxy

const (
	// RET_SUCCESS indicates the call to eth api frontend was successful
	RET_SUCCESS int = 0
	// RET_ERROR indicates the call to eth api frontend failed with an error
	RET_ERROR int = -1
	// RET_CANCELLED indicates the call to the eth api frontend was cancelled
	RET_CANCELLED int = -2
	// RET_DESER_ERROR indicates an error occurred while deserializing arguments from C to Nim
	RET_DESER_ERROR int = -3
)
