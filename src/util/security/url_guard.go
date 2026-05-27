package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const dnsLookupTimeout = 3 * time.Second

var lookupIPAddr = func(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

// ValidatePublicHTTPURL verifies that raw is an HTTP(S) URL whose host does not
// point to localhost, private networks, link-local addresses, or metadata IPs.
func ValidatePublicHTTPURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("url is required")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("url scheme must be http or https")
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return errors.New("url host is required")
	}
	if strings.Contains(host, "%") {
		return fmt.Errorf("url host %q is not allowed", host)
	}
	if strings.TrimSuffix(strings.ToLower(host), ".") == "localhost" {
		return fmt.Errorf("url host %q is not allowed", host)
	}

	if ip := net.ParseIP(host); ip != nil {
		return validatePublicIP(host, ip)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dnsLookupTimeout)
	defer cancel()

	addrs, err := lookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve url host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("resolve url host %q: no addresses", host)
	}
	for _, addr := range addrs {
		if err := validatePublicIP(host, addr.IP); err != nil {
			return err
		}
	}
	return nil
}

func validatePublicIP(host string, ip net.IP) error {
	if isUnsafeIP(ip) {
		return fmt.Errorf("url host %q resolves to disallowed address %s", host, ip.String())
	}
	return nil
}

func isUnsafeIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		!ip.IsGlobalUnicast()
}
