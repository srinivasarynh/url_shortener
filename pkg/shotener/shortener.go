package shortener

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	// DefaultLength is the default length of the short URL code
	DefaultLength = 6

	// CharSet defines the characters to be used in shortcodes
	CharSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// Shortener handles the URL shortening logic
type Shortener struct {
	length int
}

// new Shortener creates new url
func NewShortener(length int) *Shortener {
	if length <= 0 {
		length = DefaultLength
	}

	return &Shortener{length: length}
}

// generate new unique short code
func (s *Shortener) Generate() (string, error) {
	return generateRandomString(s.length)
}

// checks if a custom short code is valid
func (s *Shortener) IsValidCustomCode(code string) bool {
	if len(code) < 3 || len(code) > 20 {
		return false
	}

	for _, c := range code {
		if !strings.ContainsRune(CharSet, c) {
			return false
		}
	}

	return true
}

func generateRandomString(length int) (string, error) {
	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(CharSet)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		result[i] = CharSet[randomIndex.Int64()]
	}

	return string(result), nil
}
