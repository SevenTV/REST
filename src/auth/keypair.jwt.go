package auth

import (
	"crypto/ecdsa"
	"encoding/json"
	"time"

	"github.com/SevenTV/Common/utils"
	"github.com/golang-jwt/jwt/v4"
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

func New(publicKey string, privateKey string) (*KeypairJWT, error) {
	signingKey, err := jwt.ParseECPrivateKeyFromPEM(utils.S2B(privateKey))
	if err != nil {
		return nil, err
	}

	verifyingKey, err := jwt.ParseECPublicKeyFromPEM(utils.S2B(publicKey))
	if err != nil {
		return nil, err
	}

	return &KeypairJWT{
		signingKey:   signingKey,
		verifyingKey: verifyingKey,
	}, nil
}

type KeypairJWT struct {
	signingKey   *ecdsa.PrivateKey
	verifyingKey *ecdsa.PublicKey
}

type KeyPairClaim struct {
	*json.RawMessage
	jwt.RegisteredClaims
}

func (k *KeypairJWT) Sign(podName string, data json.RawMessage) (string, error) {
	claims := KeyPairClaim{
		&data,
		jwt.RegisteredClaims{
			Subject:   "",
			Audience:  []string{},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        "",
		},
	}

	token, err := jwt.NewWithClaims(jwt.GetSigningMethod("ES256"), claims).SignedString(k.signingKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (k *KeypairJWT) Verify(t string) (*jwt.Token, error) {
	token, err := jwt.Parse(t, func(t *jwt.Token) (interface{}, error) {
		return k.verifyingKey, nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}
