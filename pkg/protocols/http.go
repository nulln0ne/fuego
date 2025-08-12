package protocols

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nulln0ne/fuego/pkg/scenario"
)

type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

type HTTPResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	BodyText   string              `json:"body_text"`
	Duration   time.Duration       `json:"duration"`
	Size       int64               `json:"size"`
}

func NewHTTPClient(config HTTPClientConfig) *HTTPClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.VerifySSL,
		},
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &HTTPClient{
		client:  client,
		baseURL: config.BaseURL,
		headers: config.Headers,
	}
}

type HTTPClientConfig struct {
	BaseURL         string
	Headers         map[string]string
	Timeout         time.Duration
	VerifySSL       bool
	FollowRedirects bool
}

func (c *HTTPClient) Execute(step *scenario.Step) (*HTTPResponse, error) {
	if step.Type != "http" {
		return nil, fmt.Errorf("step type %s is not supported by HTTP client", step.Type)
	}

	startTime := time.Now()

	// Build request
	req, err := c.buildRequest(step)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	httpResp := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		BodyText:   string(body),
		Duration:   duration,
		Size:       int64(len(body)),
	}

	return httpResp, nil
}

func (c *HTTPClient) buildRequest(step *scenario.Step) (*http.Request, error) {
	// Build URL
	requestURL := step.Request.URL
	if !strings.HasPrefix(requestURL, "http") && c.baseURL != "" {
		requestURL = strings.TrimSuffix(c.baseURL, "/") + "/" + strings.TrimPrefix(requestURL, "/")
	}

	// Add query parameters
	if len(step.Request.Query) > 0 {
		u, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		q := u.Query()
		for key, value := range step.Request.Query {
			q.Add(key, value)
		}
		u.RawQuery = q.Encode()
		requestURL = u.String()
	}

	// Build request body
	var body io.Reader
	if step.Request.Body != nil {
		bodyBytes, err := c.buildRequestBody(step.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to build request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequest(step.Request.Method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	c.addHeaders(req, step)

	// Add authentication
	if err := c.addAuth(req, step.Request.Auth); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Add cookies
	for name, value := range step.Request.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	return req, nil
}

func (c *HTTPClient) buildRequestBody(body interface{}) ([]byte, error) {
	switch v := body.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case map[string]interface{}, []interface{}:
		return json.Marshal(v)
	default:
		return json.Marshal(v)
	}
}

func (c *HTTPClient) addHeaders(req *http.Request, step *scenario.Step) {
	// Add global headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Add step-specific headers
	for key, value := range step.Request.Headers {
		req.Header.Set(key, value)
	}

	// Set content type for JSON body if not specified
	if req.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (c *HTTPClient) addAuth(req *http.Request, auth *scenario.AuthConfig) error {
	if auth == nil {
		return nil
	}

	switch auth.Type {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "api_key":
		// Default to Authorization header, can be customized in config
		headerName := "Authorization"
		if name, ok := auth.Config["header"].(string); ok {
			headerName = name
		}
		req.Header.Set(headerName, auth.Token)
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}

	return nil
}
