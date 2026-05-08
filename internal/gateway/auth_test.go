package gateway

import "testing"

func TestCheckAuthorization(t *testing.T) {
	tests := []struct {
		name         string
		allowedUsers string
		userID       string
		want         bool
	}{
		{"empty allowed", "", "123", false},
		{"match single", "123", "123", true},
		{"no match single", "123", "456", false},
		{"match multi", "123,456,789", "456", true},
		{"no match multi", "123,456,789", "000", false},
		{"match with spaces", " 123 , 456 ", "456", true},
		{"match with whitespace", " 123 , 456 ", "123", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckAuthorization(tt.allowedUsers, tt.userID)
			if got != tt.want {
				t.Errorf("CheckAuthorization(%q, %q) = %v, want %v", tt.allowedUsers, tt.userID, got, tt.want)
			}
		})
	}
}
