package configload

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testNamespace   = "service"
	testEnvironment = "production"
	testAPIKey      = "test-api-key-value"
)

func TestLoadFromDatabase(t *testing.T) {
	t.Parallel()

	called := false
	result, err := Load(context.Background(), Settings{
		Enabled:     true,
		Source:      SourceDatabase,
		Namespace:   testNamespace,
		Environment: testEnvironment,
	}, func(_ context.Context, namespace, environment string) (*Result, error) {
		called = true
		if namespace != testNamespace || environment != testEnvironment {
			t.Fatalf("database loader identifiers = %s/%s", namespace, environment)
		}
		return &Result{Config: map[string]any{"enabled": true}, Version: 4}, nil
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !called || result.Source != SourceDatabase || result.Version != 4 {
		t.Fatalf("Load() result = %#v, called = %t", result, called)
	}
}

func TestLoadFromHTTP(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config-center/api/runtime/service/production" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("format = %q", r.URL.Query().Get("format"))
		}
		if r.Header.Get("X-API-Key") != testAPIKey {
			t.Error("request API Key does not match")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":0,"message":"ok","data":{"version":7,"config":{"port":8080,"ratio":1.5}}}`))
	}))
	defer server.Close()

	result, err := Load(context.Background(), validHTTPSettings(server.URL), nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if result.Source != SourceHTTP || result.Version != 7 {
		t.Fatalf("Load() result = %#v", result)
	}
	if result.Config["port"] != json.Number("8080") || result.Config["ratio"] != json.Number("1.5") {
		t.Fatalf("Load() numeric values = %#v", result.Config)
	}
}

func TestLoadFromHTTPRejectsErrorStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := Load(context.Background(), validHTTPSettings(server.URL), nil)
	if !errors.Is(err, ErrHTTP) {
		t.Fatalf("Load() error = %v, want ErrHTTP", err)
	}
	statusError := &HTTPStatusError{}
	if !errors.As(err, &statusError) || statusError.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Load() error = %v, want HTTP 401 classification", err)
	}
}

func TestLoadRejectsInvalidSettingsAndResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr  error
		name     string
		loader   DatabaseLoader
		settings Settings
	}{
		{
			name:     "disabled",
			settings: Settings{Source: SourceDatabase, Namespace: testNamespace, Environment: testEnvironment},
			wantErr:  ErrLoaderDisabled,
		},
		{
			name: "invalid namespace",
			settings: Settings{
				Enabled: true, Source: SourceDatabase, Namespace: "Invalid Namespace", Environment: testEnvironment,
			},
			wantErr: ErrNamespaceInvalid,
		},
		{
			name: "unsupported source",
			settings: Settings{
				Enabled: true, Source: "file", Namespace: testNamespace, Environment: testEnvironment,
			},
			wantErr: ErrUnsupportedSource,
		},
		{
			name: "database loader is missing",
			settings: Settings{
				Enabled: true, Source: SourceDatabase, Namespace: testNamespace, Environment: testEnvironment,
			},
			wantErr: ErrDatabaseLoader,
		},
		{
			name: "database result is missing",
			settings: Settings{
				Enabled: true, Source: SourceDatabase, Namespace: testNamespace, Environment: testEnvironment,
			},
			loader:  func(context.Context, string, string) (*Result, error) { return nil, nil },
			wantErr: ErrInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(context.Background(), tt.settings, tt.loader)
			if !errors.Is(err, ErrLoad) || !errors.Is(err, tt.wantErr) {
				t.Fatalf("Load() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func validHTTPSettings(baseURL string) Settings {
	return Settings{
		Enabled:     true,
		Source:      SourceHTTP,
		Namespace:   testNamespace,
		Environment: testEnvironment,
		HTTP: HTTPSettings{
			BaseURL: baseURL,
			APIKey:  testAPIKey,
			Timeout: time.Second,
		},
	}
}
