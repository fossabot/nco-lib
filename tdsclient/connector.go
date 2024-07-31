package tdsclient

import (
	"context"
	"strings"

	"github.com/ncotds/go-dblib/dsn"
	"github.com/ncotds/go-dblib/tds"

	db "github.com/ncotds/nco-lib/dbconnector"
)

const (
	DefaultConnHost      = "localhost"
	DefaultConnPort      = "4100"
	DefaultConnTransport = "tcp"

	// TDSRxQueueSize is a buffer length for TDS packages receiver.
	//
	// To prevent deadlocks, it should be enough to contain one TDS packet
	// parsed into response packages.
	//
	// OMNIbus server returns max 8192-bytes packets,
	// min response is RowResult with at least 2 bytes (int8 token and 1+ bytes for data).
	// The worst case is: 8192 / 2 = 4096, 4100 should be enough:)
	TDSRxQueueSize = 4100
)

var _ db.DBConnector = (*TDSConnector)(nil)

type TDSConnector struct {
	AppLabel   string
	TimeoutSec uint
}

func (c *TDSConnector) Connect(
	ctx context.Context,
	addr db.Addr,
	credentials db.Credentials,
) (conn db.ExecutorCloser, err error) {
	host, port, _ := strings.Cut(string(addr), ":")
	if host == "" {
		host = DefaultConnHost
	}
	if port == "" {
		port = DefaultConnPort
	}
	newConn := &Connection{
		appName:    c.AppLabel,
		connCancel: noopCancelFn,
		dsn: &tds.Info{
			Info: dsn.Info{
				Host:     host,
				Port:     port,
				Username: credentials.UserName,
				Password: credentials.Password,
			},
			Network:                 DefaultConnTransport,
			PacketReadTimeout:       int(c.TimeoutSec),
			ChannelPackageQueueSize: TDSRxQueueSize,
		},
	}
	err = newConn.open(ctx)
	if err != nil {
		return nil, makeError(db.ErrConnectionFailed, err)
	}
	return newConn, nil
}
