package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func GenerateOTP(length int) (string, error) {
	const digits = "0123456789"
	otp := make([]byte, length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("failed to generate OTP: %w", err)
		}
		otp[i] = digits[n.Int64()]
	}

	return string(otp), nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HashPIN(pin string) (string, error) {
	return HashPassword(pin)
}

func CheckPIN(pin, hash string) bool {
	return CheckPassword(pin, hash)
}
