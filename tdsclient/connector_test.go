//go:build integration
// +build integration

package tdsclient_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	db "github.com/ncotds/nco-lib/dbconnector"
	"github.com/ncotds/nco-lib/tdsclient"
)

func TestTDSConnector_Connect(t *testing.T) {
	ctx := context.Background()
	credentials := db.Credentials{TestConfig.User, TestConfig.Password}

	type args struct {
		ctx         context.Context
		addr        db.Addr
		credentials db.Credentials
	}
	tests := []struct {
		name      string
		args      args
		wantErrIs error
	}{
		{
			"connect ok",
			args{ctx: ctx, addr: TestConfig.Address, credentials: credentials},
			nil,
		},
		{
			"bad address fails",
			args{ctx: ctx, addr: db.Addr(WordFactory()), credentials: credentials},
			db.ErrConnectionFailed,
		},
		{
			"bad credentials fails",
			args{ctx: ctx, addr: TestConfig.Address, credentials: db.Credentials{WordFactory(), WordFactory()}},
			db.ErrConnectionFailed,
		},
		{
			"context cancel",
			args{
				ctx: func() context.Context {
					c, cancel := context.WithCancel(context.Background())
					cancel()
					return c
				}(),
				addr:        TestConfig.Address,
				credentials: credentials,
			},
			context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &tdsclient.TDSConnector{AppLabel: TestConnLabel, TimeoutSec: TestConnTimeoutSec}
			gotConn, err := c.Connect(tt.args.ctx, tt.args.addr, tt.args.credentials)
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
				return
			}

			assert.NoError(t, err)
			if assert.NotNil(t, gotConn) {
				assert.NoError(t, gotConn.Close())
			}
		})
	}
}
