package auth

import (
	"fmt"

	"github.com/SevenTV/Common/utils"
	"github.com/golang-jwt/jwt/v4"
)

func SignJWT(secret string, claim JWTClaimOptions) (string, error) {
	// Generate an unsigned token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	// Sign the token
	tokenStr, err := token.SignedString(utils.S2B(secret))

	return tokenStr, err
}

type JWTClaimOptions struct {
	UserID string `json:"id"`

	TokenVersion float32 `json:"ver"`
	jwt.RegisteredClaims
}

func VerifyJWT(secret string, token string, claim JWTClaimOptions) (*jwt.Token, error) {
	result, err := jwt.ParseWithClaims(
		token,
		claim,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("bad jwt signing method, expected HMAC but got %v", t.Header["alg"])
			}

			return utils.S2B(secret), nil
		},
	)

	return result, err
}
