package crypto

import (
	"testing"
)

func TestGenerateOTP(t *testing.T) {
	otp, err := GenerateOTP(6)
	if err != nil {
		t.Fatalf("GenerateOTP() error = %v", err)
	}

	if len(otp) != 6 {
		t.Errorf("GenerateOTP() length = %d, want 6", len(otp))
	}

	for _, c := range otp {
		if c < '0' || c > '9' {
			t.Errorf("GenerateOTP() contains non-digit: %c", c)
		}
	}

	otp2, _ := GenerateOTP(6)
	if otp == otp2 {
		t.Log("Warning: Two consecutive OTPs are the same (unlikely but possible)")
	}
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == password {
		t.Error("HashPassword() should not return the same string")
	}

	if len(hash) < 20 {
		t.Error("HashPassword() hash seems too short")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	hash, _ := HashPassword(password)

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword() should return true for correct password")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestHashPIN(t *testing.T) {
	pin := "1234"

	hash, err := HashPIN(pin)
	if err != nil {
		t.Fatalf("HashPIN() error = %v", err)
	}

	if !CheckPIN(pin, hash) {
		t.Error("CheckPIN() should return true for correct PIN")
	}

	if CheckPIN("4321", hash) {
		t.Error("CheckPIN() should return false for wrong PIN")
	}
}
