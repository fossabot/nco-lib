//go:build integration
// +build integration

package tdsclient_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/doug-martin/goqu/v9"
	db "github.com/ncotds/nco-lib/dbconnector"
)

const (
	TestEnvPrefix      = "TESTTDSCLIENT"
	TestConnTimeoutSec = 2
	TestConnLabel      = "tdsclient-testing"

	TestExistingRowsCount = 5
	TestAlertsTable       = "alerts.status"
)

var (
	TestConfig = Config{
		Address:  db.Addr(os.Getenv(TestEnvPrefix + "_ADDRESS")),
		User:     os.Getenv(TestEnvPrefix + "_USER"),
		Password: os.Getenv(TestEnvPrefix + "_PASSWORD"),
	}

	NcoSql = func() goqu.DialectWrapper {
		opts := goqu.DefaultDialectOptions()
		opts.QuoteRune = rune(32)
		opts.SupportsConflictTarget = false
		opts.SupportsWithCTE = false
		goqu.RegisterDialect("netcool", opts)
		return goqu.Dialect("netcool")
	}()
)

type Config struct {
	Address  db.Addr
	User     string
	Password string
}

func TestMain(t *testing.M) {
	if TestConfig.Address == "" ||
		TestConfig.User == "" ||
		TestConfig.Password == "" {
		fmt.Printf("Invalid test configuration, check %s_* env vars", TestEnvPrefix)
		os.Exit(1)
	}
	os.Exit(t.Run())
}
