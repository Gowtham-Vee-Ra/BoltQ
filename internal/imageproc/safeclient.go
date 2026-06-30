package imageproc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// maxFetchBytes caps how much of a remote response we will read into memory
// while decoding an image, bounding memory use per job.
const maxFetchBytes = 25 << 20 // 25 MiB

// fetchTimeout bounds the total time spent fetching a remote image.
const fetchTimeout = 20 * time.Second

// errBlockedAddress is returned when a URL resolves to an address we refuse to
// connect to (loopback, private, link-local, etc.).
var errBlockedAddress = fmt.Errorf("address blocked: refusing to connect to non-public host")

// safeHTTPClient returns an http.Client that refuses to connect to private,
// loopback, or link-local addresses. The check runs in DialContext against the
// IP actually being dialed, so it also defends against DNS-rebinding and
// redirect-based bypasses (every hop is re-dialed and re-checked).
func safeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("dns lookup failed: %w", err)
			}

			for _, ip := range ips {
				if isBlockedIP(ip.IP) {
					return nil, errBlockedAddress
				}
			}

			// Dial the first resolved IP we already validated, so DNS can't
			// return a different (rebound) address between check and connect.
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}

	return &http.Client{
		Timeout:   fetchTimeout,
		Transport: transport,
	}
}

// isBlockedIP reports whether an IP is in a range we never allow outbound
// connections to from a user-supplied URL.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() || // 10/8, 172.16/12, 192.168/16, fc00::/7
		ip.IsLinkLocalUnicast() || // 169.254/16 (incl. cloud metadata), fe80::/10
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() // 0.0.0.0, ::
}
