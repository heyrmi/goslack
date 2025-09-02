package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/heyrmi/goslack/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	// Try to load test config first, fallback to regular config
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Override with test database URL if running database tests
	config.DBSource = "postgresql://root:secret@localhost:5432/goslack?sslmode=disable"

	testDB, err = sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run())
}
