package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// GenerateID generates unique ID with prefix
func GenerateID(prefix string) string {
	b := make([]byte, 16)
	rand.Read(b)
	random := base64.URLEncoding.EncodeToString(b)[:22]
	random = strings.ReplaceAll(random, "-", "")
	random = strings.ReplaceAll(random, "_", "")
	return fmt.Sprintf("%s_%s", prefix, random)
}

// StringPtr returns pointer to string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns pointer to int
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns pointer to bool
func BoolPtr(b bool) *bool {
	return &b
}