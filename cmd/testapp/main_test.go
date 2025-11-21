package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func setupTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	setupHandlers(mux)
	return mux
}

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantBody     string
		wantDuration time.Duration
	}{
		{
			name:         "simple ping",
			url:          "/ping",
			wantBody:     "pong\n",
			wantDuration: 0,
		},
		{
			name:         "ping with delay",
			url:          "/ping?delay=100ms",
			wantBody:     "pong\n",
			wantDuration: 100 * time.Millisecond,
		},
	}

	mux := setupTestMux()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			start := time.Now()
			mux.ServeHTTP(w, req)
			duration := time.Since(start)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, got)
			}

			if tt.wantDuration > 0 && duration < tt.wantDuration {
				t.Errorf("expected delay of at least %v, got %v", tt.wantDuration, duration)
			}
		})
	}
}

func TestHostnameHandler(t *testing.T) {
	mux := setupTestMux()
	req := httptest.NewRequest("GET", "/hostname", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	hostname, _ := os.Hostname()
	if got := w.Body.String(); got != hostname {
		t.Errorf("expected hostname %q, got %q", hostname, got)
	}
}

func TestVersionHandler(t *testing.T) {
	mux := setupTestMux()
	req := httptest.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "rev:") {
		t.Errorf("expected version to contain 'rev:', got %q", body)
	}
}

func TestMetricsHandler(t *testing.T) {
	mux := setupTestMux()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	expectedMetrics := []string{
		"nais_testapp_lead_time",
		"nais_testapp_time_since_deploy",
		"nais_testapp_deploy_timestamp",
		"nais_testapp_start_timestamp",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("expected metrics to contain %q", metric)
		}
	}
}

func TestConnectHandler(t *testing.T) {
	// Create a test server to connect to
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer ts.Close()

	// Override connectURL for testing
	oldConnectURL := connectURL
	connectURL = ts.URL
	defer func() { connectURL = oldConnectURL }()

	mux := setupTestMux()
	req := httptest.NewRequest("GET", "/connect", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "HTTP status: 200") {
		t.Errorf("expected response to contain 'HTTP status: 200', got %q", body)
	}
	if !strings.Contains(body, "test response") {
		t.Errorf("expected response to contain 'test response', got %q", body)
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback int64
		want     int64
	}{
		{
			name:     "existing env var",
			key:      "TEST_INT",
			value:    "42",
			fallback: 0,
			want:     42,
		},
		{
			name:     "missing env var",
			key:      "MISSING_INT",
			value:    "",
			fallback: 999,
			want:     999,
		},
		{
			name:     "invalid int",
			key:      "INVALID_INT",
			value:    "not-a-number",
			fallback: 123,
			want:     0, // ParseInt returns 0 on error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvInt(tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("getEnvInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTimeSinceDeploy(t *testing.T) {
	// Set deployStartTimestamp to 1 second ago
	deployStartTimestamp = time.Now().Add(-1 * time.Second).UnixNano()

	seconds := timeSinceDeploy()

	// Should be approximately 1 second
	if seconds < 0.9 || seconds > 1.1 {
		t.Errorf("expected timeSinceDeploy() to be around 1.0, got %f", seconds)
	}
}

func TestEnvHandler(t *testing.T) {
	mux := setupTestMux()
	req := httptest.NewRequest("GET", "/env", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Response should contain environment variables
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty environment response")
	}
}

func TestLogHandlers(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"info log", "/loginfo"},
		{"error log", "/logerror"},
		{"debug log", "/logdebug"},
	}

	mux := setupTestMux()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}
		})
	}
}

func TestConnectHandlerError(t *testing.T) {
	// Test with invalid URL
	oldConnectURL := connectURL
	connectURL = "http://invalid-host-that-does-not-exist.local"
	defer func() { connectURL = oldConnectURL }()

	mux := setupTestMux()
	req := httptest.NewRequest("GET", "/connect", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should return 200 but with error message in body
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "error performing http get") {
		t.Errorf("expected error message in response, got %q", string(body))
	}
}
