package typesafeai

import (
	"fmt"
	"net/http"
	"os"
	"time"

	typesafeaigo "github.com/chez-shanpu/typesafeai-go"
)

const (
	apiKeyEnvVar = "TYPESAFE_API_KEY"

	// requestTimeout bounds a single System One request so that a stalled
	// API cannot hang the CLI.
	requestTimeout = 60 * time.Second
)

// client calls the TypeSafe AI System One API.
type client struct {
	systemOne *typesafeaigo.SystemOneClient
}

// newClientFromEnv creates a new client, reading the API key from the
// TYPESAFE_API_KEY environment variable.
func newClientFromEnv() (*client, error) {
	apiKey := os.Getenv(apiKeyEnvVar)
	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set", apiKeyEnvVar)
	}
	return newClient(apiKey, &http.Client{Timeout: requestTimeout}), nil
}

// newClient creates a new client with the given API key and HTTP client.
func newClient(apiKey string, hc *http.Client) *client {
	return &client{
		systemOne: typesafeaigo.NewSystemOneClient(apiKey, hc),
	}
}
