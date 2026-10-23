package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "normalizes case and whitespace", input: "  USER@Example.COM ", want: "user@example.com"},
		{name: "rejects display name", input: "User <user@example.com>", wantErr: true},
		{name: "rejects malformed address", input: "not-an-email", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeEmail(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("NormalizeEmail() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("NormalizeEmail() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if err := VerifyPassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if err := VerifyPassword(hash, "wrong password"); err == nil {
		t.Fatal("VerifyPassword() succeeded with wrong password")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("ValidatePassword() accepted a short password")
	}
	if err := ValidatePassword("eight888"); err != nil {
		t.Fatalf("ValidatePassword() error = %v", err)
	}
}
