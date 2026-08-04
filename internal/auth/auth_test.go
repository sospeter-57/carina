package auth

import (
	"testing"
)

// first we test our password hashing wrapper functions
func TestHashPassword(t *testing.T) {
	var plain = "Admin123"
	var _, err = HashPassword([]byte(plain), 12)
	if err != nil {
		t.Errorf("Error: %s", err)
	}

	_, err = HashPassword([]byte(plain), 8)
	if err == nil {
		t.Errorf("Error: shouldn't accept a cost less than 10")
	}

	_, err = HashPassword([]byte(plain), 15)
	if err == nil {
		t.Errorf("Error: shoudn't accept a cost more than 14")
	}

	// regardless, the cost shouldn't be a bit issue but we have
	// to test for edge cases anyway
}

// next we test our comparison function
func TestVerifyPassword(t *testing.T) {
	var hash = "$2a$10$nMTQF6vOQlpNXfHHhP1S6eN3FagjuGP7A1H5MI0R6sipLC3SiNZ/O"
	var plain = "Password123"

	err := VerifyPassword([]byte(hash), []byte(plain))
	if err != nil {
		t.Errorf("Error: %s", err)
	}
}

