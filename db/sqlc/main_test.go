package db

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"os"
	"testing"

	"github.com/heyrmi/goslack/util"
	_ "github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"
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

// createTestIPAddress creates an IP address in the format expected by PostgreSQL
func createTestIPAddress(ipStr string) pqtype.Inet {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		panic("invalid IP address: " + ipStr)
	}

	// PostgreSQL INET type stores IPv4 addresses as IPv4 (4 bytes)
	// Ensure we use IPv4 format for IPv4 addresses
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4 // Use IPv4 format
	}

	return pqtype.Inet{
		IPNet: net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(24, 32),
		},
		Valid: true,
	}
}

// requireEqualJSON compares two JSON values for semantic equality
func requireEqualJSON(t *testing.T, expected, actual pqtype.NullRawMessage) {
	if !expected.Valid && !actual.Valid {
		return
	}

	require.Equal(t, expected.Valid, actual.Valid)

	if expected.Valid && actual.Valid {
		var expectedObj, actualObj interface{}
		require.NoError(t, json.Unmarshal(expected.RawMessage, &expectedObj))
		require.NoError(t, json.Unmarshal(actual.RawMessage, &actualObj))
		require.Equal(t, expectedObj, actualObj)
	}
}
