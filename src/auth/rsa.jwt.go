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

	openssl genrsa -out Credentials/app.rsa 512 && \
	openssl rsa -in Credentials/app.rsa -pubout > Credentials/app.rsa.pub
*/

var (
	signingKey   *rsa.PrivateKey
	verifyingKey *rsa.PublicKey
)

func init() {
	// Read the signing key
	signingBytes, err := os.ReadFile("Credentials/app.rsa")
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot read private key")
	}

	signingKey, err = jwt.ParseRSAPrivateKeyFromPEM(signingBytes)
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot parse private key")
	}

	verifyingBytes, err := os.ReadFile("Credentials/app.rsa.pub")
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot read public key")
	}

	verifyingKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyingBytes)
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot parse public key")
	}
}

var RSA = rsaClient{}

type rsaClient struct{}

type RSAClaim struct {
	*json.RawMessage
	jwt.RegisteredClaims
}

func (rsaClient) Sign(data json.RawMessage) (string, error) {
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

func (rsaClient) Verify(t string) (*jwt.Token, error) {
	token, err := jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
		return verifyingKey, nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}
