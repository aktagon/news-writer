package main

import (
	"testing"
)

func TestNewAgentManager(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid api key",
			apiKey:  "test-api-key-123",
			wantErr: false,
		},
		{
			name:    "empty api key",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Settings: &Settings{},
			}

			am, err := NewAgentManager(tt.apiKey, config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgentManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if am == nil {
					t.Error("NewAgentManager() returned nil AgentManager")
				}
				if am.config != config {
					t.Error("NewAgentManager() config not set correctly")
				}
				if am.apiKey != tt.apiKey {
					t.Error("NewAgentManager() apiKey not set correctly")
				}
				if am.writerAgent == nil {
					t.Error("NewAgentManager() writerAgent not initialized")
				}
			}
		})
	}
}
