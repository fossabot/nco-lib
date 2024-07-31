package tdsclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/ncotds/go-dblib/tds"

	db "github.com/ncotds/nco-lib/dbconnector"
)

var _ db.ExecutorCloser = (*Connection)(nil)

type Connection struct {
	inUse      atomic.Bool
	conn       *tds.Conn
	connCancel context.CancelFunc
	dsn        *tds.Info
	ch         *tds.Channel
	appName    string
}

func (c *Connection) Exec(ctx context.Context, query db.Query) (rows db.RowSet, affectedRows int, err error) {
	if !c.inUse.CompareAndSwap(false, true) {
		return rows, affectedRows, ErrConnectionInUse
	}
	defer c.inUse.Store(false)
	defer func() {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			// we cannot predict when the query will be really completed
			// and cannot use this connection until query in progress...
			// so, force any communication with the server
			c.connCancel()
			_ = c.close()
		}
	}()

	err = c.open(ctx) // ensure that connection exists
	if err != nil {
		return rows, affectedRows, makeError(db.ErrConnectionFailed, err)
	}

	pkgs, err := c.exec(ctx, query)
	// unwrapped io.EOF is OK, it means server has sent all data
	// and next has closed the connection.
	// other non-nil err means that smth goes wrong
	if err != nil && err != io.EOF {
		return rows, affectedRows, makeError(ErrQueryFailed, err)
	}

	return parseResults(pkgs)
}

func (c *Connection) Close() error {
	if !c.inUse.CompareAndSwap(false, true) {
		return ErrConnectionInUse
	}
	defer c.inUse.Store(false)

	if err := c.close(); err != nil {
		return makeError(db.ErrConnectionFailed, err)
	}
	return nil
}

func (c *Connection) open(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}

	connCtx, connCancel := context.WithCancel(context.Background())

	conn, err := tds.NewConn(connCtx, c.dsn)
	if err != nil {
		connCancel()
		return err
	}

	// NOTE: OMNIbus does not support multiplexing,
	// use the main channel for all communications
	ch, err := conn.NewChannel()
	if err != nil {
		connCancel()
		return err
	}

	login, err := tds.NewLoginConfig(c.dsn)
	if err != nil {
		connCancel()
		return err
	}
	login.AppName = c.appName
	login.Encrypt = 0

	err = ch.Login(ctx, login)
	if err != nil {
		connCancel()
		return err
	}

	c.conn = conn
	c.connCancel = connCancel
	c.ch = ch
	return nil
}

func (c *Connection) close() error {
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	if errors.Is(err, context.Canceled) {
		err = nil
	}

	c.conn = nil
	c.connCancel = noopCancelFn
	c.ch = nil
	return err
}

func (c *Connection) exec(ctx context.Context, query db.Query) ([]tds.Package, error) {
	pkg := &tds.LanguagePackage{Cmd: query.SQL}

	err := c.ch.SendPackage(ctx, pkg)
	if err != nil {
		c.ch.Reset() // clear output queue
		return nil, err
	}

	var pkgs []tds.Package

	_, err = c.ch.NextPackageUntil(ctx, true, func(p tds.Package) (bool, error) {
		pkgs = append(pkgs, p)
		switch typedP := p.(type) {
		case *tds.DonePackage:
			if typedP.Status == tds.TDS_DONE_ERROR {
				return false, ErrTranFailed
			}
			return typedP.Status == tds.TDS_DONE_FINAL, nil
		default:
			return false, nil
		}
	})

	return pkgs, err
}

func makeError(base, reason error) error {
	var (
		err    error
		eedErr *tds.EEDError
	)
	switch {
	case reason == nil:
		err = base
	case errors.Is(reason, context.Canceled), errors.Is(reason, context.DeadlineExceeded):
		err = reason // return context errs as is
	case errors.As(reason, &eedErr):
		err = base
		for _, eed := range eedErr.EEDPackages {
			err = fmt.Errorf("%w: %d - %s", err, eed.MsgNumber, eed.Msg)
		}
	default:
		err = fmt.Errorf("%w: %s", base, reason.Error())
	}
	return err
}

func parseResults(pkgs []tds.Package) (rows db.RowSet, affectedRows int, err error) {
	affectedRows, err = checkTransactionCompleted(pkgs)
	if err != nil {
		return rows, affectedRows, err
	}

	rows = makeRowSet(pkgs, affectedRows)
	return rows, affectedRows, nil
}

func checkTransactionCompleted(pkgs []tds.Package) (affectedRows int, err error) { // check from the last to find 'transaction complete' msg
	var tranCompleted *tds.DonePackage
	for i := len(pkgs) - 1; i >= 0; i-- {
		pkg, ok := pkgs[i].(*tds.DonePackage)
		if ok && pkg.TranState == tds.TDS_TRAN_COMPLETED {
			tranCompleted = pkg
			break
		}
	}

	if tranCompleted == nil {
		return affectedRows, ErrTranNotCompleted
	}

	affectedRows = int(tranCompleted.Count)

	if tranCompleted.Status == tds.TDS_DONE_ERROR {
		err = ErrTranFailed
	}
	return affectedRows, err
}

func makeRowSet(pkgs []tds.Package, affectedRows int) db.RowSet {
	rows := db.RowSet{}
	rows.Columns = makeCols(pkgs)
	if rows.Columns == nil {
		return rows
	}

	rows.Rows = make([][]any, 0, affectedRows)
	for _, pkg := range pkgs {
		rowP, ok := pkg.(*tds.RowPackage)
		if !ok {
			continue
		}
		row := make([]any, len(rowP.DataFields))
		for fIdx, field := range rowP.DataFields {
			val := field.Value()
			if valStr, ok := val.(string); ok {
				val = strings.TrimSuffix(valStr, "\x00")
			}
			row[fIdx] = val
		}
		rows.Rows = append(rows.Rows, row)
	}
	return rows
}

func makeCols(pkgs []tds.Package) []string {
	var rows []string
	for i := 0; i < len(pkgs); i++ {
		rowFmt, ok := pkgs[i].(*tds.RowFmtPackage)
		if ok {
			rows = make([]string, 0, len(rowFmt.Fmts))
			for _, field := range rowFmt.Fmts {
				rows = append(rows, field.Name())
			}
			break
		}
	}
	return rows
}
