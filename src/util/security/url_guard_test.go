package security

import (
	"context"
	"errors"
	"net"
	"testing"
)

func withLookupIPAddr(t *testing.T, fn func(context.Context, string) ([]net.IPAddr, error)) {
	t.Helper()
	old := lookupIPAddr
	lookupIPAddr = fn
	t.Cleanup(func() {
		lookupIPAddr = old
	})
}

func TestValidatePublicHTTPURLAllowsPublicHTTPURL(t *testing.T) {
	withLookupIPAddr(t, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		if host != "example.com" {
			t.Fatalf("lookup host = %q, want example.com", host)
		}
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
	})

	if err := ValidatePublicHTTPURL("https://example.com/api/"); err != nil {
		t.Fatalf("ValidatePublicHTTPURL returned error: %v", err)
	}
}

func TestValidatePublicHTTPURLRejectsUnsafeURLs(t *testing.T) {
	tests := []string{
		"",
		"://bad",
		"ftp://example.com/file",
		"https:///missing-host",
		"http://localhost/notify",
		"http://localhost./notify",
		"http://127.0.0.1/notify",
		"http://[::1]/notify",
		"http://10.0.0.1/notify",
		"http://172.16.0.1/notify",
		"http://172.31.255.255/notify",
		"http://192.168.1.1/notify",
		"http://169.254.1.1/notify",
		"http://169.254.169.254/latest/meta-data/",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if err := ValidatePublicHTTPURL(raw); err == nil {
				t.Fatalf("ValidatePublicHTTPURL(%q) returned nil, want error", raw)
			}
		})
	}
}

func TestValidatePublicHTTPURLRejectsDomainResolvingToPrivateIP(t *testing.T) {
	withLookupIPAddr(t, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("10.0.0.1")}}, nil
	})

	if err := ValidatePublicHTTPURL("https://example.com/api/"); err == nil {
		t.Fatal("ValidatePublicHTTPURL returned nil, want error")
	}
}

func TestValidatePublicHTTPURLRejectsUnresolvableDomain(t *testing.T) {
	withLookupIPAddr(t, func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return nil, errors.New("lookup failed")
	})

	if err := ValidatePublicHTTPURL("https://example.com/api/"); err == nil {
		t.Fatal("ValidatePublicHTTPURL returned nil, want error")
	}
}
