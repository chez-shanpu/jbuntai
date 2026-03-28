// Package codex implements LLM backends using the ChatGPT Responses API with OAuth PKCE authentication.
package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	responsesEndpoint = "https://chatgpt.com/backend-api/wham/responses"
	userAgent         = "codex/0.1.0 (jbuntai)"
)

// client calls the ChatGPT Responses API.
type client struct {
	httpClient *http.Client
	oauth      *oauthClient
}

// newClient creates a new Responses API client.
func newClient() (*client, error) {
	oauth, err := newOAuthClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}
	return &client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		oauth:      oauth,
	}, nil
}

// responsesRequest is the request body for the Responses API.
type responsesRequest struct {
	Model        string      `json:"model"`
	Instructions string      `json:"instructions"`
	Input        []inputItem `json:"input"`
	Store        bool        `json:"store"`
	Stream       bool        `json:"stream"`
}

type inputItem struct {
	Type    string        `json:"type"`
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// run sends a prompt to the Responses API and returns the full text response.
func (c *client) run(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	token, err := c.oauth.GetValidToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	accountID, err := extractAccountID(token)
	if err != nil {
		return "", fmt.Errorf("failed to extract account ID: %w", err)
	}

	reqBody := responsesRequest{
		Model:        model,
		Instructions: systemPrompt,
		Input: []inputItem{
			{
				Type: "message",
				Role: "user",
				Content: []contentPart{
					{Type: "input_text", Text: userPrompt},
				},
			},
		},
		Store:  false,
		Stream: true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("ChatGPT-Account-Id", accountID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return parseSSEResponse(resp.Body)
}

// parseSSEResponse reads SSE events and accumulates text deltas.
func parseSSEResponse(body io.Reader) (string, error) {
	scanner := bufio.NewScanner(body)
	var text strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type     string `json:"type"`
			Delta    string `json:"delta"`
			Response struct {
				StatusCode int `json:"status_code"`
			} `json:"response"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "response.output_text.delta":
			text.WriteString(event.Delta)
		case "response.failed":
			return "", fmt.Errorf("API response failed: status=%d", event.Response.StatusCode)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading SSE stream: %w", err)
	}

	result := strings.TrimSpace(text.String())
	if result == "" {
		return "", fmt.Errorf("no text response from API")
	}

	return result, nil
}
