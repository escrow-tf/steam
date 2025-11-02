package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/rotisserie/eris"
)

type RefreshJwt struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat,omitempty"`
	Exp int64  `json:"exp,omitempty"`
}

func DecodeSimpleJwt(jwtString string) (RefreshJwt, error) {
	parts := strings.Split(jwtString, ".")
	if len(parts) != 3 {
		return RefreshJwt{}, eris.Errorf("expected 3 parts in JWT, got %d", len(parts))
	}

	standardBase64 := strings.ReplaceAll(parts[1], "-", "+")
	standardBase64 = strings.ReplaceAll(standardBase64, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(standardBase64)
	if err != nil {
		return RefreshJwt{}, eris.Errorf("base64 decoding failed: %v", err)
	}

	jwt := RefreshJwt{}
	err = json.Unmarshal(decoded, &jwt)
	if err != nil {
		return RefreshJwt{}, eris.Errorf("unmarshalling failed: %v", err)
	}

	return jwt, nil
}
