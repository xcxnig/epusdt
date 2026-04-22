package http

import (
	"path/filepath"
	"testing"
)

func TestResolveSPAFilePath(t *testing.T) {
	root := filepath.Join("tmp", "www")
	indexPath := filepath.Join(root, "index.html")

	tests := []struct {
		name        string
		wildcard    string
		wantPath    string
		wantTryStat bool
	}{
		{
			name:        "relative asset path",
			wildcard:    "assets/app.js",
			wantPath:    filepath.Join(root, "assets", "app.js"),
			wantTryStat: true,
		},
		{
			name:        "absolute style asset path",
			wildcard:    "/assets/app.js",
			wantPath:    filepath.Join(root, "assets", "app.js"),
			wantTryStat: true,
		},
		{
			name:        "empty wildcard",
			wildcard:    "",
			wantPath:    indexPath,
			wantTryStat: false,
		},
		{
			name:        "dot wildcard",
			wildcard:    ".",
			wantPath:    indexPath,
			wantTryStat: false,
		},
		{
			name:        "directory traversal fallback",
			wildcard:    "../../etc/passwd",
			wantPath:    indexPath,
			wantTryStat: false,
		},
		{
			name:        "absolute directory traversal fallback",
			wildcard:    "/../../etc/passwd",
			wantPath:    filepath.Join(root, "etc", "passwd"),
			wantTryStat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotTryStat := ResolveSPAFilePath(root, tt.wildcard)

			if gotTryStat != tt.wantTryStat {
				t.Fatalf("tryStat = %v, want %v", gotTryStat, tt.wantTryStat)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestShouldSkipSPAFallback(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "api root", path: "/api", want: true},
		{name: "api sub path", path: "/api/install/defaults", want: true},
		{name: "admin api root", path: "/admin/api", want: true},
		{name: "admin api sub path", path: "/admin/api/v1/orders", want: true},
		{name: "payments root", path: "/payments", want: true},
		{name: "payments sub path", path: "/payments/gmpay/v1/order/create-transaction", want: true},
		{name: "pay root", path: "/pay", want: true},
		{name: "pay sub path", path: "/pay/checkout-counter/abc", want: true},
		{name: "not exact prefix", path: "/apiary", want: false},
		{name: "normal spa route", path: "/install", want: false},
		{name: "normal asset route", path: "/assets/index.js", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipSPAFallback(tt.path)
			if got != tt.want {
				t.Fatalf("ShouldSkipSPAFallback(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
