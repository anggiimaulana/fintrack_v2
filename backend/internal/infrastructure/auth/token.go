package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Access token claim constants.
const (
	Issuer   = "fintrack"
	Audience = "fintrack-web"
)

// ErrInvalidToken is returned when an access token cannot be validated.
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims are the custom JWT claims carried by an access token.
type Claims struct {
	Sub   string `json:"sub"`
	Role  string `json:"role"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// IssueAccessToken creates a signed HS256 access token.
func IssueAccessToken(userID uuid.UUID, role, email string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Sub:   userID.String(),
		Role:  role,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseAccessToken validates a token's signature, issuer, audience and expiry,
// returning the parsed claims on success.
func ParseAccessToken(tok string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
	)
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Sub == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// NewRefreshToken generates a fresh opaque refresh token: the raw value is
// returned to the client, the sha256 hex hash is stored in the database.
func NewRefreshToken() (raw string, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashRefreshToken(raw)
}

// HashRefreshToken returns the sha256 hex digest of a raw refresh token.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
