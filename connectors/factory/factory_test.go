package factory

import (
	"testing"
)

func TestNewRTEClient(t *testing.T) {
	tests := []struct {
		id          string
		expectedErr bool
	}{
		{IDWholesaleMarket, false},
		{"unknown_id", true},
	}

	for _, tt := range tests {
		client, err := NewRTEClient(tt.id)
		if tt.expectedErr {
			if err == nil {
				t.Errorf("expected error for id %s, got nil", tt.id)
			}
		} else {
			if err != nil {
				t.Errorf("did not expect error for id %s, got %v", tt.id, err)
			}
			if client == nil {
				t.Errorf("expected non-nil client for id %s", tt.id)
			}
		}
	}
}
