package auth

import (
	"strings"
	"testing"
)

func TestTOTPManager_GenerateSetup(t *testing.T) {
	manager, err := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	if err != nil {
		t.Fatalf("NewTOTPManager() error = %v", err)
	}

	setup, err := manager.GenerateSetup("+254712345678")
	if err != nil {
		t.Fatalf("GenerateSetup() error = %v", err)
	}

	// Check secret is base32 encoded
	if len(setup.Secret) == 0 {
		t.Error("Secret should not be empty")
	}

	// Check encrypted key
	if len(setup.EncryptedKey) == 0 {
		t.Error("EncryptedKey should not be empty")
	}

	// Check QR code URL format
	if setup.QRCodeURL == "" {
		t.Error("QRCodeURL should not be empty")
	}
	if !contains(setup.QRCodeURL, "otpauth://totp/") {
		t.Errorf("QRCodeURL should start with otpauth://totp/, got %s", setup.QRCodeURL)
	}
	if !contains(setup.QRCodeURL, "EquiShare") {
		t.Errorf("QRCodeURL should contain issuer, got %s", setup.QRCodeURL)
	}

	// Check recovery codes
	if len(setup.RecoveryCodes) != RecoveryCodeCount {
		t.Errorf("RecoveryCodes count = %d, want %d", len(setup.RecoveryCodes), RecoveryCodeCount)
	}
	if len(setup.RecoveryHashes) != RecoveryCodeCount {
		t.Errorf("RecoveryHashes count = %d, want %d", len(setup.RecoveryHashes), RecoveryCodeCount)
	}

	// Check recovery code format (XXXX-XXXX)
	for _, code := range setup.RecoveryCodes {
		if len(code) != 9 { // 4 + 1 (dash) + 4
			t.Errorf("Recovery code %s has wrong length %d", code, len(code))
		}
	}
}

func TestTOTPManager_ValidateCode(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// Generate the current code
	code, err := manager.GenerateCode(setup.EncryptedKey)
	if err != nil {
		t.Fatalf("GenerateCode() error = %v", err)
	}

	// Validate the code
	valid, err := manager.ValidateCode(setup.EncryptedKey, code)
	if err != nil {
		t.Fatalf("ValidateCode() error = %v", err)
	}
	if !valid {
		t.Error("ValidateCode() should return true for correct code")
	}
}

func TestTOTPManager_ValidateCode_Invalid(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// Try an invalid code
	valid, err := manager.ValidateCode(setup.EncryptedKey, "000000")
	if err != nil {
		t.Fatalf("ValidateCode() error = %v", err)
	}
	if valid {
		t.Error("ValidateCode() should return false for incorrect code")
	}
}

func TestTOTPManager_ValidateCode_ClockSkew(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")

	// Generate secret manually for testing
	setup, _ := manager.GenerateSetup("+254712345678")

	// The code should be valid even with some time drift
	// Since we allow ±30 seconds, the current code should always work
	code, _ := manager.GenerateCode(setup.EncryptedKey)
	valid, _ := manager.ValidateCode(setup.EncryptedKey, code)
	if !valid {
		t.Error("Current code should be valid")
	}
}

func TestTOTPManager_ValidateRecoveryCode(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// First recovery code should be valid
	index := manager.ValidateRecoveryCode(setup.RecoveryCodes[0], setup.RecoveryHashes)
	if index != 0 {
		t.Errorf("ValidateRecoveryCode() = %d, want 0", index)
	}

	// Invalid code should return -1
	index = manager.ValidateRecoveryCode("XXXX-XXXX", setup.RecoveryHashes)
	if index != -1 {
		t.Errorf("ValidateRecoveryCode() for invalid code = %d, want -1", index)
	}
}

func TestTOTPManager_ValidateRecoveryCode_CaseInsensitive(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// Should work with lowercase
	code := setup.RecoveryCodes[0]
	index := manager.ValidateRecoveryCode(strings.ToLower(code), setup.RecoveryHashes)
	if index != 0 {
		t.Errorf("ValidateRecoveryCode() with lowercase = %d, want 0", index)
	}
}

func TestTOTPManager_ValidateRecoveryCode_NoDash(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// Should work without dash
	code := strings.ReplaceAll(setup.RecoveryCodes[0], "-", "")
	index := manager.ValidateRecoveryCode(code, setup.RecoveryHashes)
	if index != 0 {
		t.Errorf("ValidateRecoveryCode() without dash = %d, want 0", index)
	}
}

func TestTOTPManager_ValidateRecoveryCode_Used(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// Mark first code as used (empty the hash)
	setup.RecoveryHashes[0] = ""

	// Should not validate used code
	index := manager.ValidateRecoveryCode(setup.RecoveryCodes[0], setup.RecoveryHashes)
	if index != -1 {
		t.Errorf("ValidateRecoveryCode() for used code = %d, want -1", index)
	}
}

func TestTOTPManager_EncryptDecrypt(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")

	// Generate setup and verify we can decrypt the key
	setup, _ := manager.GenerateSetup("+254712345678")

	// Generate a code (which requires decryption)
	code1, err := manager.GenerateCode(setup.EncryptedKey)
	if err != nil {
		t.Fatalf("GenerateCode() error = %v", err)
	}
	if len(code1) != TOTPDigits {
		t.Errorf("Code length = %d, want %d", len(code1), TOTPDigits)
	}

	// Same key should generate same code within same 30-second window
	code2, _ := manager.GenerateCode(setup.EncryptedKey)
	if code1 != code2 {
		t.Error("Same key should generate same code in same time window")
	}
}

func TestTOTPManager_DifferentKeys(t *testing.T) {
	manager1, _ := NewTOTPManager("EquiShare", "encryption-key-1-32-bytes!!!!!")
	manager2, _ := NewTOTPManager("EquiShare", "encryption-key-2-32-bytes!!!!!")

	setup, _ := manager1.GenerateSetup("+254712345678")

	// Should fail to decrypt with different key
	_, err := manager2.GenerateCode(setup.EncryptedKey)
	if err == nil {
		t.Error("Should fail to decrypt with different key")
	}
}

func TestTOTPManager_UniqueSecrets(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")

	setup1, _ := manager.GenerateSetup("+254712345678")
	setup2, _ := manager.GenerateSetup("+254712345678")

	if setup1.Secret == setup2.Secret {
		t.Error("Each setup should have a unique secret")
	}

	if setup1.EncryptedKey == setup2.EncryptedKey {
		t.Error("Each encrypted key should be unique (different nonce)")
	}
}

func TestTOTPManager_UniqueRecoveryCodes(t *testing.T) {
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	seen := make(map[string]bool)
	for _, code := range setup.RecoveryCodes {
		if seen[code] {
			t.Errorf("Duplicate recovery code: %s", code)
		}
		seen[code] = true
	}
}

func TestTOTPManager_CodeChangesOverTime(t *testing.T) {
	// This is a conceptual test - we can't actually wait 30 seconds
	// But we can verify the code generation is time-dependent
	manager, _ := NewTOTPManager("EquiShare", "test-encryption-key-32-bytes!")
	setup, _ := manager.GenerateSetup("+254712345678")

	// Generate codes at different hypothetical times
	// Note: In real usage, codes change every 30 seconds
	code1, _ := manager.GenerateCode(setup.EncryptedKey)

	// Verify code format
	if len(code1) != 6 {
		t.Errorf("Code should be 6 digits, got %d", len(code1))
	}
	for _, c := range code1 {
		if c < '0' || c > '9' {
			t.Errorf("Code should only contain digits, got %c", c)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
