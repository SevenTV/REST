package auth

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/SevenTV/REST/src/configure"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
)

/*
	Generate keyfiles:

	openssl genrsa -out Credentials/app.rsa 4096 && \
	openssl rsa -in Credentials/app.rsa -pubout > Credentials/app.rsa.pub
*/

var signingKey *rsa.PrivateKey

func init() {
	// Read the signing key
	signingBytes, err := os.ReadFile("Credentials/app.rsa")
	if err != nil {
		log.WithError(err).Fatal("auth, rsa")
	}

	signingKey, err = jwt.ParseRSAPrivateKeyFromPEM(signingBytes)
	if err != nil {
		log.WithError(err).Fatal("auth, rsa")
	}
}

type RSA struct {
}

type RSAClaim struct {
	*json.RawMessage
	jwt.RegisteredClaims
}

func (RSA) Sign(data json.RawMessage) (string, error) {
	claims := RSAClaim{
		&data,
		jwt.RegisteredClaims{
			Issuer:    fmt.Sprintf("7TV (%s)", configure.PodName),
			Subject:   "",
			Audience:  []string{},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        "",
		},
	}

	token, err := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims).SignedString(signingKey)
	if err != nil {
		return "", err
	}

	return token, nil
}
