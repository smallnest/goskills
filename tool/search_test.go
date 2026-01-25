package tool

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestTavilySearchNoAPIKey(t *testing.T) {
	// Unset API key
	os.Unsetenv("TAVILY_API_KEY")

	_, err := TavilySearch("test query")
	if err == nil {
		t.Error("TavilySearch() should error when API key is not set")
	}
	if !strings.Contains(err.Error(), "TAVILY_API_KEY") {
		t.Errorf("TavilySearch() error = %v, want API key error", err)
	}
}

func TestTavilySearchWithMockServer(t *testing.T) {
	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got %s", auth)
		}

		// Send mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"results": [
				{
					"title": "Test Result",
					"url": "https://example.com",
					"content": "Test content"
				}
			],
			"images": ["https://example.com/image.jpg"]
		}`))
	}))
	defer mockServer.Close()

	// Set API key
	os.Setenv("TAVILY_API_KEY", "test-api-key")
	defer os.Unsetenv("TAVILY_API_KEY")

	// Test with mock server URL
	result, err := TavilySearchWithLimitAndURL("test query", 5, mockServer.URL)
	if err != nil {
		t.Errorf("TavilySearchWithLimitAndURL() error = %v", err)
	}

	if !strings.Contains(result, "Test Result") {
		t.Errorf("TavilySearchWithLimitAndURL() = %v, want 'Test Result'", result)
	}
	if !strings.Contains(result, "https://example.com") {
		t.Errorf("TavilySearchWithLimitAndURL() = %v, want URL", result)
	}
}

func TestTavilySearchLimitValidation(t *testing.T) {
	os.Setenv("TAVILY_API_KEY", "test-key")
	defer os.Unsetenv("TAVILY_API_KEY")

	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request to check limit
		// For simplicity, just respond OK
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": [], "images": []}`))
	}))
	defer mockServer.Close()

	tests := []struct {
		name    string
		limit   int
		wantErr bool
	}{
		{"zero limit", 0, false},
		{"negative limit", -1, false},
		{"valid limit", 10, false},
		{"max limit", 100, false},
		{"over max limit", 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := TavilySearchWithLimitAndURL("test", tt.limit, mockServer.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("TavilySearchWithLimitAndURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
