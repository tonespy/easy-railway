// Package api contains Railway API clients, request types, and query definitions.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tonespy/easy-railway/internal/log"
)

const defaultEndpoint = "https://backboard.railway.com/graphql/v2"

const redactedValue = "[REDACTED]"

// Client executes Railway GraphQL requests.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *log.Logger
}

// New returns a Client using the Railway production endpoint.
func New() *Client {
	return &Client{
		baseURL: defaultEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.Default,
	}
}

// NewWithEndpoint returns a Client pointed at a custom URL (e.g. for testing).
func NewWithEndpoint(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: log.Default,
	}
}

// SetLogger overrides the logger used for debug and trace output.
func (c *Client) SetLogger(l *log.Logger) {
	c.logger = l
}

func (c *Client) do(ctx context.Context, token, query string, variables map[string]any, dest any) error {
	return c.doWithHeaders(ctx, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}, query, variables, dest)
}

func (c *Client) doWithHeaders(ctx context.Context, headers map[string]string, query string, variables map[string]any, dest any) error {
	body, err := json.Marshal(GraphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	c.logger.Debug("POST %s", c.baseURL)
	c.traceHeaders(headers)
	c.logger.Trace("request body:\n%s", prettyJSON(body))

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	elapsed := time.Since(start)
	c.logger.Debug("HTTP %d (%s)", resp.StatusCode, elapsed.Round(time.Millisecond))
	c.logger.Trace("response body:\n%s", prettyJSON(respBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (c *Client) traceHeaders(headers map[string]string) {
	if len(headers) == 0 {
		return
	}

	raw, err := json.Marshal(redactHeaders(headers))
	if err != nil {
		c.logger.Trace("request headers: %v", redactHeaders(headers))

		return
	}

	c.logger.Trace("request headers:\n%s", prettyJSON(raw))
}

func redactHeaders(headers map[string]string) map[string]string {
	redacted := make(map[string]string, len(headers))

	for key, value := range headers {
		if isSensitiveHeader(key) {
			redacted[key] = redactHeaderValue(key, value)
			continue
		}

		redacted[key] = value
	}

	return redacted
}

func isSensitiveHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "proxy-authorization", "project-access-token", "x-api-key", "x-auth-token":
		return true
	default:
		return false
	}
}

func redactHeaderValue(name, value string) string {
	if strings.EqualFold(name, "authorization") || strings.EqualFold(name, "proxy-authorization") {
		parts := strings.Fields(value)
		if len(parts) >= 2 {
			return parts[0] + " " + redactedValue
		}
	}

	return redactedValue
}

// prettyJSON formats raw JSON bytes with indentation for readable trace output.
// Falls back to the raw input if it is not valid JSON.
func prettyJSON(data []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return string(data)
	}

	return buf.String()
}
