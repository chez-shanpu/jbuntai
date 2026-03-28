package codex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// extractAccountID extracts the ChatGPT account ID from a JWT access token.
// It decodes the payload without verifying the signature (server-side validation is sufficient).
func extractAccountID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid JWT: expected at least 2 parts, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	authRaw, ok := claims["https://api.openai.com/auth"]
	if !ok {
		return "", fmt.Errorf("JWT missing https://api.openai.com/auth claim")
	}

	var auth struct {
		ChatGPTAccountID string `json:"chatgpt_account_id"`
	}
	if err := json.Unmarshal(authRaw, &auth); err != nil {
		return "", fmt.Errorf("failed to parse auth claim: %w", err)
	}

	if auth.ChatGPTAccountID == "" {
		return "", fmt.Errorf("chatgpt_account_id is empty in JWT")
	}

	return auth.ChatGPTAccountID, nil
}
