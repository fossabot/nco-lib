//go:build integration
// +build integration

package tdsclient_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/ncotds/go-dblib/dsn"
	"github.com/ncotds/go-dblib/tds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ncotds/nco-lib/dbconnector"
	"github.com/ncotds/nco-lib/tdsclient"
)

func TestConnection_Close(t *testing.T) {
	ctx := context.Background()
	credentials := db.Credentials{TestConfig.User, TestConfig.Password}

	client := &tdsclient.TDSConnector{TestConnLabel, TestConnTimeoutSec}
	conn, err := client.Connect(ctx, TestConfig.Address, credentials)
	require.NoError(t, err, "cannot establish connection")

	assert.NoError(t, conn.Close(), "first close")
	assert.NoError(t, conn.Close(), "second close")
}

func TestConnection_Close_Race(t *testing.T) {
	ctx := context.Background()
	credentials := db.Credentials{TestConfig.User, TestConfig.Password}

	client := &tdsclient.TDSConnector{TestConnLabel, TestConnTimeoutSec}
	conn, err := client.Connect(ctx, TestConfig.Address, credentials)
	require.NoError(t, err, "cannot establish connection")

	errs := make(chan error, 5)
	for i := 0; i < cap(errs); i++ {
		go func() {
			errs <- conn.Close()
		}()
	}
	var errCnt int
	for i := 0; i < cap(errs); i++ {
		if e := <-errs; e != nil {
			errCnt++
		}
	}
	assert.Greater(t, errCnt, 0, "expected a few goroutines failed")
}

func TestConnection_Exec_Insert(t *testing.T) {
	conn, _ := setUpTest(t, 0)
	defer tearDownTest(t, conn)

	toInsert := AlertStatusRecordFactory()
	stmt, _, _ := NcoSql.Insert(TestAlertsTable).Rows(toInsert).ToSQL()

	gotRows, gotAffected, gotErr := conn.Exec(context.Background(), db.Query{SQL: stmt})

	assert.Zero(t, gotRows)
	assert.Equal(t, 1, gotAffected)
	assert.NoError(t, gotErr)

	rowsFromDB, _ := selTestRows()
	assert.Equal(t, []AlertStatusRecord{toInsert}, rowsFromDB)
}

func TestConnection_Exec_Select(t *testing.T) {
	conn, existingRows := setUpTest(t, 3)
	defer tearDownTest(t, conn)

	var existingIds []interface{}
	for _, r := range existingRows {
		existingIds = append(existingIds, r.Identifier)
	}
	stmt, _, _ := NcoSql.From(TestAlertsTable).
		Select(&AlertStatusRecord{}).
		Where(goqu.C("Identifier").In(existingIds...)).
		ToSQL()

	gotRows, gotAffected, gotErr := conn.Exec(context.Background(), db.Query{SQL: stmt})

	var gotRecords []AlertStatusRecord
	for _, r := range gotRows.Rows {
		gotRecords = append(gotRecords, NewAlertStatusRecordFromCursor(gotRows.Columns, r))
	}

	assert.ElementsMatch(t, existingRows, gotRecords)
	assert.Equal(t, len(existingRows), gotAffected)
	assert.NoError(t, gotErr)
}

func TestConnection_Exec_Update(t *testing.T) {
	conn, existingRows := setUpTest(t, 3)
	defer tearDownTest(t, conn)

	updSummary := SentenceFactory()
	var existingIds []interface{}
	for _, r := range existingRows {
		existingIds = append(existingIds, r.Identifier)
	}
	stmt, _, _ := NcoSql.Update(TestAlertsTable).
		Set(goqu.Record{"Summary": updSummary}).
		Where(goqu.C("Identifier").In(existingIds...)).
		ToSQL()

	gotRows, gotAffected, gotErr := conn.Exec(context.Background(), db.Query{SQL: stmt})

	assert.Zero(t, gotRows)
	assert.Equal(t, len(existingRows), gotAffected)
	assert.NoError(t, gotErr)

	var expectedRows []AlertStatusRecord
	for _, r := range existingRows {
		r.Summary = updSummary
		expectedRows = append(expectedRows, r)
	}
	rowsFromDB, _ := selTestRows()
	assert.ElementsMatch(t, expectedRows, rowsFromDB)
}

func TestConnection_Exec_Delete(t *testing.T) {
	conn, existingRows := setUpTest(t, 3)
	defer tearDownTest(t, conn)

	var existingIds []interface{}
	for _, r := range existingRows {
		existingIds = append(existingIds, r.Identifier)
	}
	stmt, _, _ := NcoSql.Delete(TestAlertsTable).Where(goqu.C("Identifier").In(existingIds...)).ToSQL()

	gotRows, gotAffected, gotErr := conn.Exec(context.Background(), db.Query{SQL: stmt})

	assert.Zero(t, gotRows)
	assert.Equal(t, len(existingRows), gotAffected)
	assert.NoError(t, gotErr)

	rowsFromDB, _ := selTestRows()
	assert.Empty(t, rowsFromDB)
}

func TestConnection_Exec_BadStatement(t *testing.T) {
	conn, _ := setUpTest(t, 0)
	defer tearDownTest(t, conn)

	gotRows, gotAffected, gotErr := conn.Exec(context.Background(), db.Query{SQL: SentenceFactory()})

	assert.Zero(t, gotRows)
	assert.Zero(t, gotAffected)
	assert.ErrorIs(t, gotErr, tdsclient.ErrQueryFailed)
}

func TestConnection_Exec_ContextCancel(t *testing.T) {
	conn, _ := setUpTest(t, 0)
	defer tearDownTest(t, conn)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, errFirst := conn.Exec(ctx, db.Query{SQL: "describe status"})
	_, _, errSecond := conn.Exec(context.Background(), db.Query{SQL: "describe status"})

	assert.ErrorIsf(t, errFirst, context.Canceled, "first query not cancelled, reason: %v", errors.Unwrap(errFirst))
	assert.NoErrorf(t, errSecond, "second query not ok, reason: %v", errors.Unwrap(errSecond))
}

func TestConnection_Exec_ContextTimeout(t *testing.T) {
	conn, _ := setUpTest(t, 1000)
	defer tearDownTest(t, conn)

	selQuery, _, err := NcoSql.From(TestAlertsTable).
		Select(&AlertStatusRecord{}).
		Where(goqu.Ex{"Manager": TestManager, "Agent": TestAgent}).
		ToSQL()
	require.NoError(t, err, "cannot create test query")
	selQueryRepeat := strings.Join([]string{
		selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery,
		selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery,
		selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery, selQuery,
	}, "; ")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, errFirst := conn.Exec(ctx, db.Query{SQL: selQueryRepeat})
	_, _, errSecond := conn.Exec(context.Background(), db.Query{SQL: "describe status"})

	assert.ErrorIsf(t, errFirst, context.DeadlineExceeded, "first too long query not cancelled, reason: %v", errors.Unwrap(errFirst))
	assert.NoErrorf(t, errSecond, "second query not ok, reason: %v", errors.Unwrap(errSecond))
}

func TestConnection_Exec_Reconnect(t *testing.T) {
	conn, existingRows := setUpTest(t, 1)
	defer tearDownTest(t, conn)

	err := conn.Close()
	require.NoError(t, err, "cannot close connection")

	stmt, _, err := NcoSql.
		Select(goqu.L("top 1 Identifier")).
		From(TestAlertsTable).
		Where(goqu.C("Identifier").Eq(existingRows[0].Identifier)).
		ToSQL()
	require.NoError(t, err, "cannot make SQL stmt")

	rows, affected, err := conn.Exec(context.Background(), db.Query{SQL: stmt})

	assert.NoError(t, err)
	assert.Equal(t, 1, affected)
	assert.Len(t, rows.Rows, 1)
}

func TestConnection_Exec_ReconnectCancel(t *testing.T) {
	conn, _ := setUpTest(t, 0)
	defer tearDownTest(t, conn)

	err := conn.Close()
	require.NoError(t, err, "cannot close connection")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = conn.Exec(ctx, db.Query{SQL: "describe status"})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestConnection_Exec_Race(t *testing.T) {
	conn, existingRows := setUpTest(t, 3)
	defer tearDownTest(t, conn)

	var existingIds []interface{}
	for _, r := range existingRows {
		existingIds = append(existingIds, r.Identifier)
	}
	stmt, _, _ := NcoSql.From(TestAlertsTable).
		Select(&AlertStatusRecord{}).
		Where(goqu.C("Identifier").In(existingIds...)).
		ToSQL()

	errs := make(chan error, 5)
	for i := 0; i < cap(errs); i++ {
		go func() {
			_, _, err := conn.Exec(context.Background(), db.Query{SQL: stmt})
			errs <- err
		}()
	}
	var errCnt int
	for i := 0; i < cap(errs); i++ {
		if e := <-errs; e != nil {
			errCnt++
		}
	}
	assert.Greater(t, errCnt, 0, "expected a few goroutines failed")
}

func setUpTest(t *testing.T, rowsCount int) (conn db.ExecutorCloser, rows []AlertStatusRecord) {
	require.NoError(t, delTestRows(), "cannot delete previous test data")
	for i := 0; i < rowsCount; i++ {
		row, err := insTestRow()
		require.NoError(t, err, "cannot insert test data")
		rows = append(rows, row)
	}

	ctx := context.Background()
	credentials := db.Credentials{TestConfig.User, TestConfig.Password}

	client := &tdsclient.TDSConnector{TestConnLabel, TestConnTimeoutSec}
	conn, err := client.Connect(ctx, TestConfig.Address, credentials)
	require.NoError(t, err, "cannot establish connection")

	return conn, rows
}

func tearDownTest(t *testing.T, conn db.ExecutorCloser) {
	require.NoError(t, delTestRows(), "cannot clear test data")
	require.NoError(t, conn.Close(), "cannot close connection")
}

type Conn struct {
	conn *tds.Conn
	ch   *tds.Channel
}

func (c *Conn) Exec(stmt string) ([]tds.Package, error) {
	ctx := context.Background()
	pkg := &tds.LanguagePackage{Cmd: stmt}

	err := c.ch.SendPackage(ctx, pkg)
	if err != nil {
		c.ch.Reset() // clear output queue
		return nil, err
	}

	var pkgs []tds.Package

	_, err = c.ch.NextPackageUntil(ctx, true, func(p tds.Package) (bool, error) {
		pkgs = append(pkgs, p)
		if typed, ok := p.(*tds.DonePackage); ok && typed.Status == tds.TDS_DONE_FINAL {
			return true, nil
		}
		return false, nil
	})

	return pkgs, err
}

func (c *Conn) Close() {
	_ = c.conn.Close()
}

func makeConn() (*Conn, error) {
	host, port, _ := strings.Cut(string(TestConfig.Address), ":")
	conf := &tds.Info{
		Info: dsn.Info{
			Host:     host,
			Port:     port,
			Username: TestConfig.User,
			Password: TestConfig.Password,
		},
		Network:                 tdsclient.DefaultConnTransport,
		PacketReadTimeout:       TestConnTimeoutSec,
		ChannelPackageQueueSize: tdsclient.TDSRxQueueSize,
	}

	conn, err := tds.NewConn(context.Background(), conf)
	if err != nil {
		return nil, err
	}

	ch, err := conn.NewChannel()
	if err != nil {
		return nil, err
	}

	login, err := tds.NewLoginConfig(conf)
	if err != nil {
		return nil, err
	}
	login.AppName = TestConnLabel
	login.Encrypt = 0

	ctx, cancel := context.WithTimeout(context.Background(), TestConnTimeoutSec*time.Second)
	defer cancel()

	err = ch.Login(ctx, login)
	if err != nil {
		return nil, err
	}

	return &Conn{conn: conn, ch: ch}, nil
}

func insTestRow() (AlertStatusRecord, error) {
	insRow := AlertStatusRecordFactory()

	conn, err := makeConn()
	if err != nil {
		return insRow, err
	}
	defer conn.Close()

	insQuery, _, err := NcoSql.Insert(TestAlertsTable).Rows(insRow).ToSQL()
	if err != nil {
		return insRow, err
	}
	_, err = conn.Exec(insQuery)
	return insRow, err
}

func delTestRows() error {
	conn, err := makeConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	delQuery, _, err := NcoSql.Delete(TestAlertsTable).
		Where(goqu.Ex{
			"Manager": TestManager,
			"Agent":   TestAgent,
		}).
		ToSQL()
	if err != nil {
		return err
	}
	_, err = conn.Exec(delQuery)
	return err
}

func selTestRows() (rows []AlertStatusRecord, err error) {
	conn, err := makeConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	selQuery, _, err := NcoSql.From(TestAlertsTable).
		Select(&AlertStatusRecord{}).
		Where(goqu.Ex{"Manager": TestManager, "Agent": TestAgent}).
		ToSQL()
	if err != nil {
		return nil, err
	}
	pkgs, err := conn.Exec(selQuery)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		if typed, ok := pkg.(*tds.DonePackage); ok && typed.TranState == tds.TDS_TRAN_COMPLETED {
			break
		}
		if typed, ok := pkg.(*tds.RowPackage); ok {
			record := AlertStatusRecord{}
			rv := reflect.ValueOf(&record).Elem()
			for _, col := range typed.DataFields {
				val := col.Value()
				if valStr, ok := val.(string); ok {
					val = strings.TrimSuffix(valStr, "\x00")
				}
				field := rv.FieldByName(col.Format().Name())
				field.Set(reflect.ValueOf(val).Convert(field.Type()))
			}
			rows = append(rows, record)
		}
	}

	return rows, err
}
