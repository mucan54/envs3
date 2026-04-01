package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mucan54/envs3/internal/format"
)

const tokenPrefix = "envs3_v1_"

// ParseServiceToken decodes a service token string into its payload.
func ParseServiceToken(raw string) (*format.TokenPayload, error) {
	if !strings.HasPrefix(raw, tokenPrefix) {
		return nil, fmt.Errorf("invalid token: missing prefix")
	}

	encoded := raw[len(tokenPrefix):]
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid token: decode error: %w", err)
	}

	var payload format.TokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("invalid token: parse error: %w", err)
	}

	return &payload, nil
}

// EncodeServiceToken encodes a token payload into a token string.
func EncodeServiceToken(payload *format.TokenPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(data)
	return tokenPrefix + encoded, nil
}
