package llmtrace

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is an HTTP client for an LLM observability provider (Langfuse-compatible API).
type Client struct {
	host      string
	publicKey string
	secretKey string
	http      *http.Client
}

// NewClient creates a new LLM trace API client.
func NewClient(host, publicKey, secretKey string) (*Client, error) {
	host = strings.TrimRight(host, "/")
	if host == "" {
		return nil, fmt.Errorf("llmtrace host is required")
	}
	if publicKey == "" {
		return nil, fmt.Errorf("llmtrace public key is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("llmtrace secret key is required")
	}
	return &Client{
		host:      host,
		publicKey: publicKey,
		secretKey: secretKey,
		http:      &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) authHeader() string {
	creds := c.publicKey + ":" + c.secretKey
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	url := c.host + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// HealthCheck tests connectivity to the LLM trace provider.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, status, err := c.doRequest(ctx, http.MethodGet, "/api/public/health", nil)
	if err != nil {
		return fmt.Errorf("llmtrace health check: %w", err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("llmtrace health check failed (HTTP %d)", status)
	}
	return nil
}

// FetchTraces retrieves a page of traces, optionally filtered by timestamp.
func (c *Client) FetchTraces(ctx context.Context, page, limit int, fromTimestamp *time.Time) (*TracesResponse, error) {
	path := fmt.Sprintf("/api/public/traces?page=%d&limit=%d&orderBy=timestamp.asc", page, limit)
	if fromTimestamp != nil {
		path += "&fromTimestamp=" + fromTimestamp.UTC().Format(time.RFC3339)
	}

	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch traces error (HTTP %d): %s", status, strings.TrimSpace(string(body)))
	}

	var resp TracesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse traces response: %w", err)
	}
	return &resp, nil
}

// FetchTrace retrieves a single trace by ID.
func (c *Client) FetchTrace(ctx context.Context, traceID string) (*Trace, error) {
	path := "/api/public/traces/" + traceID
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch trace error (HTTP %d): %s", status, strings.TrimSpace(string(body)))
	}

	var trace Trace
	if err := json.Unmarshal(body, &trace); err != nil {
		return nil, fmt.Errorf("parse trace response: %w", err)
	}
	return &trace, nil
}

// FetchObservations retrieves all observations for a trace.
func (c *Client) FetchObservations(ctx context.Context, traceID string) ([]Observation, error) {
	path := "/api/public/observations?traceId=" + traceID
	body, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch observations error (HTTP %d): %s", status, strings.TrimSpace(string(body)))
	}

	var resp ObservationsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse observations response: %w", err)
	}
	return resp.Data, nil
}

// PostScore writes a security score back to the LLM trace provider.
func (c *Client) PostScore(ctx context.Context, score ScoreRequest) error {
	body, status, err := c.doRequest(ctx, http.MethodPost, "/api/public/scores", score)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("post score error (HTTP %d): %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}
