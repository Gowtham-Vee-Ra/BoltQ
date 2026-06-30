package imageproc

import (
	"net"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"169.254.169.254", true}, // cloud metadata endpoint
		{"127.0.0.1", true},       // loopback
		{"10.0.0.5", true},        // private
		{"192.168.1.10", true},    // private
		{"172.16.0.1", true},      // private
		{"0.0.0.0", true},         // unspecified
		{"::1", true},             // ipv6 loopback
		{"fe80::1", true},         // ipv6 link-local
		{"fc00::1", true},         // ipv6 unique-local (private)
		{"8.8.8.8", false},        // public
		{"1.1.1.1", false},        // public
		{"93.184.216.34", false},  // public (example.com)
	}

	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("could not parse %q", c.ip)
		}
		if got := isBlockedIP(ip); got != c.blocked {
			t.Errorf("isBlockedIP(%s) = %v, want %v", c.ip, got, c.blocked)
		}
	}
}
