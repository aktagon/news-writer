package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		name     string
		videoURL string
		expected string
		wantErr  bool
	}{
		{
			name:     "youtube.com watch URL",
			videoURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
			wantErr:  false,
		},
		{
			name:     "youtu.be short URL",
			videoURL: "https://youtu.be/dQw4w9WgXcQ",
			expected: "dQw4w9WgXcQ",
			wantErr:  false,
		},
		{
			name:     "youtu.be with query params",
			videoURL: "https://youtu.be/i0P56Pm1Q3U?si=r_78flhyOFGnX58f",
			expected: "i0P56Pm1Q3U",
			wantErr:  false,
		},
		{
			name:     "invalid URL",
			videoURL: "not-a-url",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "non-youtube URL",
			videoURL: "https://example.com/watch?v=abc123",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "youtube URL without video ID",
			videoURL: "https://www.youtube.com/channel/UC123",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractVideoID(tt.videoURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("extractVideoID() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("extractVideoID() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("extractVideoID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFetchTranscript(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		wantErr        bool
	}{
		{
			name:           "successful transcript (200)",
			responseStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "missing url parameter (400)",
			responseStatus: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name:           "invalid API key (401)",
			responseStatus: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name:           "transcripts disabled (404)",
			responseStatus: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:           "server error (500)",
			responseStatus: http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				if tt.responseStatus == http.StatusOK {
					w.Write([]byte("Mock transcript content"))
				} else {
					w.Write([]byte("Error message"))
				}
			}))
			defer server.Close()

			result, err := fetchTranscript("dQw4w9WgXcQ", "test-key", server.URL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("fetchTranscript() expected error for status %d, got nil", tt.responseStatus)
				}
				return
			}

			if err != nil {
				t.Errorf("fetchTranscript() unexpected error: %v", err)
				return
			}

			if result == "" {
				t.Error("fetchTranscript() expected non-empty result")
			}
		})
	}
}

func TestYouTubeHandler_Handle_MissingConfig(t *testing.T) {
	handler := &YouTubeHandler{}

	// Save and clear environment variables
	originalKey := os.Getenv("YOUTUBE_TRANSCRIPT_API_KEY")
	originalURL := os.Getenv("YOUTUBE_TRANSCRIPT_API_URL")

	os.Unsetenv("YOUTUBE_TRANSCRIPT_API_KEY")
	os.Unsetenv("YOUTUBE_TRANSCRIPT_API_URL")

	defer func() {
		if originalKey != "" {
			os.Setenv("YOUTUBE_TRANSCRIPT_API_KEY", originalKey)
		}
		if originalURL != "" {
			os.Setenv("YOUTUBE_TRANSCRIPT_API_URL", originalURL)
		}
	}()

	result, err := handler.Handle("https://youtu.be/dQw4w9WgXcQ", nil)

	if err == nil {
		t.Error("Handle() expected error for missing config, got nil")
	}

	if result != nil {
		t.Error("Handle() expected nil result for missing config")
	}

	if !strings.Contains(err.Error(), "YouTube API configuration missing") {
		t.Errorf("Handle() error = %v, want config missing error", err)
	}
}

func TestYouTubeHandler_CanHandle(t *testing.T) {
	handler := &YouTubeHandler{}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "youtube.com watch URL",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: true,
		},
		{
			name:     "youtu.be short URL",
			url:      "https://youtu.be/dQw4w9WgXcQ",
			expected: true,
		},
		{
			name:     "non-YouTube URL",
			url:      "https://example.com/video",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.CanHandle(tt.url, nil)
			if result != tt.expected {
				t.Errorf("CanHandle() = %v, want %v", result, tt.expected)
			}
		})
	}
}
