package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestResolvePathParams_EscapesAndRejectsInjection is the regression guard for
// the URL/path-parameter injection vulnerability (fixed 2026-07-07): a route
// param value must never be able to break out of its single path segment into
// the upstream host, path, or query. Values with separators (/ ? #) or
// traversal (..) are rejected; everything else is URL-path-escaped.
func TestResolvePathParams_EscapesAndRejectsInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		template string
		params   gin.Params
		want     string
		wantOK   bool
	}{
		{"plain id passes through", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: "h-1"}}, "/hosts/h-1", true},
		{"no params is unchanged", "/hosts", nil, "/hosts", true},
		{"empty param leaves template segment", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: ""}}, "/hosts/:hostId", true},
		{"special chars are escaped", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: "a b"}}, "/hosts/a%20b", true},
		{"reject embedded slash", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: "a/b"}}, "", false},
		{"reject query injection", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: "a?evil=1"}}, "", false},
		{"reject fragment injection", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: "a#frag"}}, "", false},
		{"reject bare traversal", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: ".."}}, "", false},
		{"reject embedded traversal", "/hosts/:hostId", gin.Params{{Key: "hostId", Value: "a..b"}}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Params = tc.params

			got, ok := resolvePathParams(tc.template, c)
			if ok != tc.wantOK {
				t.Fatalf("resolvePathParams ok = %v, want %v (value should be %s)",
					ok, tc.wantOK, map[bool]string{true: "accepted", false: "rejected"}[tc.wantOK])
			}
			if ok && got != tc.want {
				t.Fatalf("resolvePathParams = %q, want %q", got, tc.want)
			}
		})
	}
}
