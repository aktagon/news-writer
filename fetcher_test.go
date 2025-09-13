package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mock handler for testing
type mockHandler struct {
	canHandleResult bool
	handleResult    *ContentResult
	handleError     error
}

func (m *mockHandler) CanHandle(url string, resp *http.Response) bool {
	return m.canHandleResult
}

func (m *mockHandler) Handle(url string, resp *http.Response) (*ContentResult, error) {
	return m.handleResult, m.handleError
}

func TestNewContentFetcher(t *testing.T) {
	apiKey := "test-key"

	fetcher := NewContentFetcher(apiKey)

	if fetcher == nil {
		t.Fatal("NewContentFetcher() returned nil")
	}

	if fetcher.client == nil {
		t.Error("NewContentFetcher() did not initialize HTTP client")
	}

	if len(fetcher.handlers) == 0 {
		t.Error("NewContentFetcher() did not register any handlers")
	}

	expectedHandlerCount := 3 // YouTube, PDF, HTML
	if len(fetcher.handlers) != expectedHandlerCount {
		t.Errorf("NewContentFetcher() registered %d handlers, want %d",
			len(fetcher.handlers), expectedHandlerCount)
	}
}

func TestAddHandler(t *testing.T) {
	fetcher := &ContentFetcher{}
	initialCount := len(fetcher.handlers)

	mockH := &mockHandler{canHandleResult: true}
	fetcher.AddHandler(mockH)

	if len(fetcher.handlers) != initialCount+1 {
		t.Errorf("AddHandler() handlers count = %d, want %d",
			len(fetcher.handlers), initialCount+1)
	}

	lastHandler := fetcher.handlers[len(fetcher.handlers)-1]
	if lastHandler != mockH {
		t.Error("AddHandler() did not add handler to the end of the chain")
	}
}

func TestFetchContentHTTPError(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := &ContentFetcher{
		client: server.Client(),
	}

	result, err := fetcher.FetchContent(server.URL)

	if result != nil {
		t.Error("FetchContent() should return nil result on HTTP error")
	}

	if err == nil {
		t.Fatal("FetchContent() should return error on HTTP 404")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Errorf("FetchContent() should return HTTPError, got %T", err)
	} else {
		if httpErr.StatusCode != http.StatusNotFound {
			t.Errorf("HTTPError.StatusCode = %d, want %d",
				httpErr.StatusCode, http.StatusNotFound)
		}
		if httpErr.URL != server.URL {
			t.Errorf("HTTPError.URL = %q, want %q", httpErr.URL, server.URL)
		}
	}
}

func TestFetchContentHandlerChain(t *testing.T) {
	// Create test server that returns HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<h1>Test HTML</h1>"))
	}))
	defer server.Close()

	// Create mock handlers to test chain order
	handler1 := &mockHandler{
		canHandleResult: false, // This handler won't handle
	}

	handler2 := &mockHandler{
		canHandleResult: true, // This handler will handle
		handleResult:    &ContentResult{Text: "handler2 result"},
	}

	handler3 := &mockHandler{
		canHandleResult: true, // This handler would handle but won't be reached
		handleResult:    &ContentResult{Text: "handler3 result"},
	}

	fetcher := &ContentFetcher{
		client:   server.Client(),
		handlers: []ContentHandler{handler1, handler2, handler3},
	}

	result, err := fetcher.FetchContent(server.URL)

	if err != nil {
		t.Fatalf("FetchContent() error = %v", err)
	}

	if result == nil {
		t.Fatal("FetchContent() returned nil result")
	}

	if result.Text != "handler2 result" {
		t.Errorf("FetchContent() result.Text = %q, want %q",
			result.Text, "handler2 result")
	}

	if !strings.Contains(result.Text, "handler2") {
		t.Error("Wrong handler was used - should use first matching handler")
	}
}

func TestFetchContentNoMatchingHandler(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("some content"))
	}))
	defer server.Close()

	// Create handlers that won't match
	handler1 := &mockHandler{canHandleResult: false}
	handler2 := &mockHandler{canHandleResult: false}

	fetcher := &ContentFetcher{
		client:   server.Client(),
		handlers: []ContentHandler{handler1, handler2},
	}

	result, err := fetcher.FetchContent(server.URL)

	if result != nil {
		t.Error("FetchContent() should return nil when no handler matches")
	}

	if err == nil {
		t.Fatal("FetchContent() should return error when no handler matches")
	}

	expectedMsg := "no handler found for " + server.URL
	if err.Error() != expectedMsg {
		t.Errorf("FetchContent() error = %q, want %q", err.Error(), expectedMsg)
	}
}
