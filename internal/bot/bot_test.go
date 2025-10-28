package bot

import (
	"testing"
)

func TestPrepareDestination(t *testing.T) {
	tests := []struct {
		name       string
		recipients string
		wantLen    int
	}{
		{
			name:       "single recipient",
			recipients: "123,0",
			wantLen:    1,
		},
		{
			name:       "multiple recipients",
			recipients: "123,0;456,1",
			wantLen:    2,
		},
		{
			name:       "three recipients",
			recipients: "123,0;456,1;789,2",
			wantLen:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareDestination(tt.recipients)
			if len(result) != tt.wantLen {
				t.Errorf("prepareDestination() len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestPrepareDestination_NumberParsing(t *testing.T) {
	tests := []struct {
		name         string
		recipients   string
		wantChatID   int64
		wantThreadID int
	}{
		{
			name:         "zero thread_id",
			recipients:   "123456,0",
			wantChatID:   123456,
			wantThreadID: 0,
		},
		{
			name:         "positive thread_id",
			recipients:   "789012,5",
			wantChatID:   789012,
			wantThreadID: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prepareDestination(tt.recipients)
			if len(result) != 1 {
				t.Fatalf("prepareDestination() returned %d recipients, want 1", len(result))
			}

			if result[0].User.ID != tt.wantChatID {
				t.Errorf("prepareDestination() User.ID = %d, want %d", result[0].User.ID, tt.wantChatID)
			}

			if result[0].ThreadID != tt.wantThreadID {
				t.Errorf("prepareDestination() ThreadID = %d, want %d", result[0].ThreadID, tt.wantThreadID)
			}
		})
	}
}
