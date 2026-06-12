package settings

import "testing"

func TestUserInfoIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{name: "letters numbers underscore", username: "audit_admin1", want: true},
		{name: "too short", username: "ab", want: false},
		{name: "hyphen rejected", username: "audit-admin", want: false},
		{name: "space rejected", username: "audit admin", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserInfo{Username: tt.username}.IsValidUsername()
			if got != tt.want {
				t.Fatalf("IsValidUsername(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}
