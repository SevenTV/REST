package auth

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SevenTV/Common/utils"
	"github.com/golang-jwt/jwt/v4"
)

func SignJWT(secret string, claim jwt.Claims) (string, error) {
	// Generate an unsigned token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	// Sign the token
	tokenStr, err := token.SignedString(utils.S2B(secret))

	return tokenStr, err
}

type JWTClaimUser struct {
	UserID       string  `json:"id"`
	TokenVersion float32 `json:"ver"`

	jwt.RegisteredClaims
}

type JWTClaimOAuth2CSRF struct {
	State     string `json:"s"`
	CreatedAt int64  `json:"at"`

	jwt.RegisteredClaims
}

func VerifyJWT(secret string, token []string, claim jwt.Claims, out interface{}) (*jwt.Token, error) {
	result, err := jwt.ParseWithClaims(
		strings.Join(token, ""),
		claim,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("bad jwt signing method, expected HMAC but got %v", t.Header["alg"])
			}

			return utils.S2B(secret), nil
		},
	)

	val, err := jwt.DecodeSegment(token[1])
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(val, out); err != nil {
		return nil, err
	}

	return result, err
}
