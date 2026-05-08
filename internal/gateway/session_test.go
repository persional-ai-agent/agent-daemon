package gateway

import "testing"

func TestBuildSessionKey(t *testing.T) {
	tests := []struct {
		platform string
		chatType string
		chatID   string
		want     string
	}{
		{"telegram", "dm", "123", "agent:main:telegram:dm:123"},
		{"telegram", "group", "-456", "agent:main:telegram:group:-456"},
		{"discord", "dm", "789", "agent:main:discord:dm:789"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := BuildSessionKey(tt.platform, tt.chatType, tt.chatID)
			if got != tt.want {
				t.Errorf("BuildSessionKey(%q, %q, %q) = %q, want %q", tt.platform, tt.chatType, tt.chatID, got, tt.want)
			}
		})
	}
}
