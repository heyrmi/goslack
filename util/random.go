package util

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomInt generates a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomOwner generates a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// RandomMoney generates a random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

// RandomCurrency generates a random currency code
func RandomCurrency() string {
	currencies := []string{"USD", "EUR", "CAD"}
	n := len(currencies)
	return currencies[rand.Intn(n)]
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@example.com", RandomString(6))
}

// RandomOrganizationName generates a random organization name
func RandomOrganizationName() string {
	organizations := []string{"Tech Corp", "Innovation Inc", "Digital Solutions", "Future Systems", "Smart Tech"}
	return organizations[rand.Intn(len(organizations))]
}

// RandomBool generates a random boolean value
func RandomBool() bool {
	return rand.Intn(2) == 1
}

// IntToString converts an int64 to string
func IntToString(i int64) string {
	return fmt.Sprintf("%d", i)
}

// RandomIPNet generates a random IP network for testing
func RandomIPNet() net.IPNet {
	// Generate a random IP address
	ip := net.IPv4(
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
	)

	// Create a /24 network
	_, network, _ := net.ParseCIDR(fmt.Sprintf("%s/24", ip.String()))
	return *network
}
