package tdsclient

import "errors"

var (
	// ErrConnectionInUse returned if caller tries to use Connection concurrently,
	// that is not possible with TDS protocol 'one-query-at-a-time' limitation:
	// OMNIbus server does not support multiplexing
	ErrConnectionInUse = errors.New("connection in use")
	// ErrQueryFailed means that 'transaction error' message has been received from the server
	ErrQueryFailed = errors.New("query failed")

	ErrTranNotCompleted = errors.New("transaction completed message did not found")
	ErrTranFailed       = errors.New("transaction failed")
)
