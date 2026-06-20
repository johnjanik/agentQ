package coremcp

import "testing"

// SECURITY-REVIEW.md #5: the CoreMCP CORS layer must never reflect a wildcard.
// Only localhost dev origins and the configured production domain (plus its
// subdomains) are allowed; everything else is denied.
func TestIsAllowedOrigin(t *testing.T) {
	const domain = "app.agentrq.com"

	cases := []struct {
		name   string
		origin string
		domain string
		want   bool
	}{
		{"empty origin", "", domain, false},
		{"localhost http", "http://localhost:5173", domain, true},
		{"localhost https", "https://localhost", domain, true},
		{"loopback ip", "http://127.0.0.1:3000", domain, true},
		{"exact domain", "https://app.agentrq.com", domain, true},
		{"mcp subdomain", "https://mcp.app.agentrq.com", domain, true},
		{"workspace subdomain", "https://ws1.mcp.app.agentrq.com", domain, true},
		{"unrelated host", "https://evil.com", domain, false},
		{"suffix trick", "https://app.agentrq.com.evil.com", domain, false},
		{"prefix trick", "https://notapp.agentrq.com", domain, false},
		{"non-local origin when domain unset", "https://anything.com", "", false},
		{"localhost still ok when domain unset", "http://localhost", "", true},
		{"non-local origin when domain is localhost", "https://evil.com", "localhost", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAllowedOrigin(tc.origin, tc.domain); got != tc.want {
				t.Errorf("isAllowedOrigin(%q, %q) = %v, want %v", tc.origin, tc.domain, got, tc.want)
			}
		})
	}
}
