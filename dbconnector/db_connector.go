//go:generate mockery
package dbconnector

import (
	"context"
	"io"
)

// Addr is connection address, usually in "host:port" format
type Addr string

// DBConnector is object that can open new connections
type DBConnector interface {
	// Connect creates new database connection.
	//
	// NOTE:
	//
	// Connect implementation should return instance of ErrConnectionFailed if target DB is unavailable.
	//
	// If caller canceled the context, impl can return context.Canceled/context.DeadlineExceeded errors
	Connect(ctx context.Context, addr Addr, credentials Credentials) (conn ExecutorCloser, err error)
}

// ExecutorCloser is object that can make database queries
type ExecutorCloser interface {
	// Exec performs DB query and return results.
	//
	// NOTE:
	//
	// Implementation of Exec should return instance of app.ErrUnavailable if DB connection is loosed.
	// So, caller can decide: try to reconnect and repeat or not.
	//
	// If caller canceled the context, impl can return context.Canceled/context.DeadlineExceeded errors
	Exec(ctx context.Context, query Query) (rows RowSet, affectedRows int, err error)
	io.Closer
}
