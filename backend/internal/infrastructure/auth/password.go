package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost is the work factor for password hashing (cost 12 per plan).
const bcryptCost = 12

// HashPassword returns a bcrypt hash (cost 12) of the given plaintext password.
func HashPassword(p string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword reports whether the plaintext password matches the bcrypt hash.
func CheckPassword(hash, p string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)) == nil
}
