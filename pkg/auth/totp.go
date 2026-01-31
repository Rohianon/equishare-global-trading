package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// =============================================================================
// TOTP (Time-based One-Time Password) Implementation
// =============================================================================
// RFC 6238 compliant TOTP for 2FA. Secrets are encrypted at rest using AES-GCM.
// Recovery codes are generated and hashed (one-time use).
// =============================================================================

const (
	// TOTPDigits is the number of digits in a TOTP code
	TOTPDigits = 6
	// TOTPPeriod is the time step in seconds (standard is 30)
	TOTPPeriod = 30
	// TOTPSecretLength is the length of the raw TOTP secret in bytes
	TOTPSecretLength = 20
	// RecoveryCodeCount is the number of recovery codes generated
	RecoveryCodeCount = 10
	// RecoveryCodeLength is the length of each recovery code
	RecoveryCodeLength = 8
)

// TOTPManager handles TOTP operations
type TOTPManager struct {
	issuer        string
	encryptionKey []byte // 32 bytes for AES-256
}

// TOTPSetup contains the data needed to set up TOTP for a user
type TOTPSetup struct {
	Secret         string   `json:"secret"`          // Base32-encoded secret (for display)
	EncryptedKey   string   `json:"-"`               // Encrypted secret for storage
	QRCodeURL      string   `json:"qr_code_url"`     // otpauth:// URL for QR code
	RecoveryCodes  []string `json:"recovery_codes"`  // Plain recovery codes (show once)
	RecoveryHashes []string `json:"-"`               // Hashed recovery codes for storage
}

// NewTOTPManager creates a new TOTP manager
func NewTOTPManager(issuer string, encryptionKey string) (*TOTPManager, error) {
	// Derive a 32-byte key from the provided key using SHA-256
	hash := sha256.Sum256([]byte(encryptionKey))

	return &TOTPManager{
		issuer:        issuer,
		encryptionKey: hash[:],
	}, nil
}

// GenerateSetup creates a new TOTP setup for a user
func (m *TOTPManager) GenerateSetup(userIdentifier string) (*TOTPSetup, error) {
	// Generate random secret
	secret := make([]byte, TOTPSecretLength)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// Base32 encode the secret (standard for TOTP)
	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	// Encrypt the secret for storage
	encryptedKey, err := m.encryptSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Generate QR code URL
	qrURL := m.generateQRCodeURL(userIdentifier, secretBase32)

	// Generate recovery codes
	recoveryCodes, recoveryHashes, err := m.generateRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	return &TOTPSetup{
		Secret:         secretBase32,
		EncryptedKey:   encryptedKey,
		QRCodeURL:      qrURL,
		RecoveryCodes:  recoveryCodes,
		RecoveryHashes: recoveryHashes,
	}, nil
}

// ValidateCode validates a TOTP code against an encrypted secret
func (m *TOTPManager) ValidateCode(encryptedKey, code string) (bool, error) {
	// Decrypt the secret
	secret, err := m.decryptSecret(encryptedKey)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	// Allow 1 period of clock skew in each direction
	now := time.Now().Unix()
	for _, offset := range []int64{-TOTPPeriod, 0, TOTPPeriod} {
		expected := m.generateCode(secret, now+offset)
		if code == expected {
			return true, nil
		}
	}

	return false, nil
}

// ValidateRecoveryCode validates and consumes a recovery code
// Returns the index of the used code (-1 if invalid)
func (m *TOTPManager) ValidateRecoveryCode(code string, hashes []string) int {
	// Normalize code (remove spaces and dashes)
	code = strings.ReplaceAll(strings.ToUpper(code), "-", "")
	code = strings.ReplaceAll(code, " ", "")

	// Check against all hashes
	for i, hash := range hashes {
		if hash == "" {
			continue // Already used
		}
		if HashToken(code) == hash {
			return i
		}
	}

	return -1
}

// GenerateCode generates the current TOTP code for an encrypted secret
func (m *TOTPManager) GenerateCode(encryptedKey string) (string, error) {
	secret, err := m.decryptSecret(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return m.generateCode(secret, time.Now().Unix()), nil
}

// =============================================================================
// Internal Methods
// =============================================================================

func (m *TOTPManager) generateCode(secret []byte, timestamp int64) string {
	// Calculate counter value
	counter := uint64(timestamp / TOTPPeriod)

	// Convert counter to big-endian byte array
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, secret)
	mac.Write(counterBytes)
	hash := mac.Sum(nil)

	// Dynamic truncation
	offset := hash[len(hash)-1] & 0x0f
	truncatedHash := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Get the last N digits
	code := truncatedHash % 1000000 // 10^6 for 6 digits

	return fmt.Sprintf("%06d", code)
}

func (m *TOTPManager) generateQRCodeURL(userIdentifier, secret string) string {
	// otpauth://totp/Issuer:user@example.com?secret=XXX&issuer=Issuer&digits=6&period=30
	label := url.PathEscape(fmt.Sprintf("%s:%s", m.issuer, userIdentifier))
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", m.issuer)
	params.Set("digits", fmt.Sprintf("%d", TOTPDigits))
	params.Set("period", fmt.Sprintf("%d", TOTPPeriod))

	return fmt.Sprintf("otpauth://totp/%s?%s", label, params.Encode())
}

func (m *TOTPManager) generateRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, RecoveryCodeCount)
	hashes := make([]string, RecoveryCodeCount)

	for i := 0; i < RecoveryCodeCount; i++ {
		// Generate random bytes
		bytes := make([]byte, RecoveryCodeLength/2)
		if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
			return nil, nil, err
		}

		// Format as uppercase hex with dash in middle
		code := fmt.Sprintf("%X", bytes)
		codes[i] = code[:4] + "-" + code[4:]

		// Store hash (without dash)
		hashes[i] = HashToken(code)
	}

	return codes, hashes, nil
}

// =============================================================================
// Encryption (AES-256-GCM)
// =============================================================================

func (m *TOTPManager) encryptSecret(secret []byte) (string, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, secret, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (m *TOTPManager) decryptSecret(encrypted string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// =============================================================================
// TOTP Store Interface
// =============================================================================

// TOTPData represents stored 2FA data for a user
type TOTPData struct {
	UserID         string   `json:"user_id"`
	EncryptedKey   string   `json:"encrypted_key"`
	Enabled        bool     `json:"enabled"`
	RecoveryHashes []string `json:"recovery_hashes"` // Hashed recovery codes
	EnabledAt      int64    `json:"enabled_at"`
}

// TOTPStore defines the interface for TOTP data storage
type TOTPStore interface {
	// Save stores TOTP data for a user
	Save(userID string, data *TOTPData) error

	// Get retrieves TOTP data for a user
	Get(userID string) (*TOTPData, error)

	// Delete removes TOTP data for a user
	Delete(userID string) error

	// UseRecoveryCode marks a recovery code as used
	UseRecoveryCode(userID string, index int) error
}
