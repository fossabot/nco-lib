package dbconnector

import "errors"

var (
	// ErrConnectionFailed means that client cannot communicate with DB server
	ErrConnectionFailed = errors.New("connection failed")
)
