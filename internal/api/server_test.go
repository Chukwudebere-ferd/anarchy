package api

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiterAllowsThenDenies(t *testing.T) {
	s := newServer(nil, nil)
	now := time.Now()
	for i := 0; i < requestsPerIP; i++ {
		if !s.allow("1.2.3.4", now) {
			t.Fatalf("request %d must be allowed", i+1)
		}
	}
	if s.allow("1.2.3.4", now) {
		t.Fatal("request 11 must be denied")
	}
	// Different IP unaffected.
	if !s.allow("5.6.7.8", now) {
		t.Fatal("second IP must be independent")
	}
}

func TestLimiterWindowResets(t *testing.T) {
	s := newServer(nil, nil)
	now := time.Now()
	for i := 0; i < requestsPerIP; i++ {
		s.allow("1.2.3.4", now)
	}
	if s.allow("1.2.3.4", now.Add(time.Minute)) {
		t.Fatal("still within window, must be denied")
	}
	if !s.allow("1.2.3.4", now.Add(windowPerIP+time.Second)) {
		t.Fatal("after window, must be allowed again")
	}
}

func TestLimiterMemoryStaysBounded(t *testing.T) {
	s := newServer(nil, nil)
	now := time.Now()
	// Churn 5000 unique stale IPs + a few fresh ones.
	for i := 0; i < 5000; i++ {
		s.allow(fakeIP(i), now.Add(-2*windowPerIP))
	}
	s.allow("fresh-ip", now)
	s.allow(fakeIP(1), now) // triggers the sweep via stale oldest entry
	if len(s.windows) > 100 {
		t.Fatalf("stale entries must be swept, got %d", len(s.windows))
	}
	if !s.allow("fresh-ip", now.Add(time.Second)) {
		t.Fatal("fresh ip must keep its slot after sweep")
	}
}

func TestClientIPPrefersForwardedFor(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/bots", nil)
	r.RemoteAddr = "10.0.0.1:443" // proxy
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
	if got := clientIP(r); got != "203.0.113.7" {
		t.Fatalf("want first XFF hop, got %q", got)
	}
	r2 := httptest.NewRequest("POST", "/v1/bots", nil)
	r2.RemoteAddr = "192.168.1.5:9999"
	if got := clientIP(r2); got != "192.168.1.5" {
		t.Fatalf("fallback to RemoteAddr host, got %q", got)
	}
}

func fakeIP(i int) string {
	return "198.51.100." + string(rune('a'+i%26)) + ":" + itoa(i)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
