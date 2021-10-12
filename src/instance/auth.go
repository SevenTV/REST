package instance

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v4"
)

type Auth interface {
	Sign(podName string, data json.RawMessage) (string, error)
	Verify(t string) (*jwt.Token, error)
}
