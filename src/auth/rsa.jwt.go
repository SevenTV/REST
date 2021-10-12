package auth

import (
	"crypto/ecdsa"
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

	-- ECDSA --
	openssl ecparam -name prime256v1 -genkey -noout -out Credentials/privkey.pem && \
	openssl ec -in Credentials/privkey.pem -pubout -out Credentials/pubkey.pem && \
	openssl req -new -x509 -key Credentials/privkey.pem -out Credentials/ca.pem -days 30

	-- RSA --
	openssl genrsa -out Credentials/app.rsa 512 && \
	openssl rsa -in Credentials/app.rsa -pubout > Credentials/app.rsa.pub

*/

var (
	signingKey   *ecdsa.PrivateKey
	verifyingKey *ecdsa.PublicKey
)

func init() {
	// Read the signing key
	signingBytes, err := os.ReadFile("Credentials/privkey.pem")
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot read private key")
	}

	signingKey, err = jwt.ParseECPrivateKeyFromPEM(signingBytes)
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot parse private key")
	}

	verifyingBytes, err := os.ReadFile("Credentials/pubkey.pem")
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot read public key")
	}

	verifyingKey, err = jwt.ParseECPublicKeyFromPEM(verifyingBytes)
	if err != nil {
		log.WithError(err).Fatal("auth, rsa: cannot parse public key")
	}
}

var ECDSA = ecdsaClient{}

type ecdsaClient struct{}

type RSAClaim struct {
	*json.RawMessage
	jwt.RegisteredClaims
}

func (ecdsaClient) Sign(data json.RawMessage) (string, error) {
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

	token, err := jwt.NewWithClaims(jwt.GetSigningMethod("ES256"), claims).SignedString(signingKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (ecdsaClient) Verify(t string) (*jwt.Token, error) {
	token, err := jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
		return verifyingKey, nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}
