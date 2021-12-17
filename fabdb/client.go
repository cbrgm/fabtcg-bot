package fabdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Version is current version of this client.
const Version = "1.0.0"

const (
	apiEndpoint = "https://api.fabdb.net"
)

type APIError struct {
	// StatusCode is the HTTP response status code
	StatusCode int `json:"-"`
	message    string
}

func (a APIError) Error() string {
	if len(a.message) > 0 {
		return a.message
	}
	return fmt.Sprintf("HTTP Requests failed with statuscode %d", a.StatusCode)
}

func newDefaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

var defaultHTTPClient HTTPClient = newDefaultHTTPClient()

// Client wraps http client
type Client struct {
	apiEndpoint string

	// HTTPClient is the HTTP client used for making requests against the
	// FabDB.net API. You can use either *http.Client here, or your own
	// implementation.
	HTTPClient HTTPClient
}

// ClientOptions allows for options to be passed into the Client for customization
type ClientOptions func(*Client)

// NewClient creates an API client
func NewClient(options ...ClientOptions) *Client {
	client := Client{
		apiEndpoint: apiEndpoint,
		HTTPClient:  defaultHTTPClient,
	}

	for _, opt := range options {
		opt(&client)
	}

	return &client
}

// WithAPIEndpoint allows for a custom API endpoint to be passed into the the client
func WithAPIEndpoint(endpoint string) ClientOptions {
	return func(c *Client) {
		c.apiEndpoint = endpoint
	}
}

// Do sets some headers on the request, before actioning it using the internal
// HTTPClient. This also assumes any request body is in JSON format and sets the Content-Type to application/json.
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	c.prepRequest(r, nil)

	return c.HTTPClient.Do(r)
}

func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) put(ctx context.Context, path string, payload interface{}, headers map[string]string) (*http.Response, error) {
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return c.do(ctx, http.MethodPut, path, bytes.NewBuffer(data), headers)
	}
	return c.do(ctx, http.MethodPut, path, nil, headers)
}

func (c *Client) post(ctx context.Context, path string, payload interface{}, headers map[string]string) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, path, bytes.NewBuffer(data), headers)
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, nil)
}

const (
	userAgentHeader   = "fabtcg-bot/" + Version
	contentTypeHeader = "application/json"
)

func (c *Client) prepRequest(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("User-Agent", userAgentHeader)
	req.Header.Set("Content-Type", contentTypeHeader)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	return c.doWithEndpoint(ctx, c.apiEndpoint, method, path, body, headers)
}

func (c *Client) doWithEndpoint(ctx context.Context, endpoint, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	var resp *http.Response
	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	c.prepRequest(req, headers)

	resp, err = c.HTTPClient.Do(req)

	return c.checkResponse(resp, err)
}

func (c *Client) checkResponse(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, fmt.Errorf("Error calling the API endpoint: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp, c.getErrorFromResponse(resp)
	}

	return resp, nil
}

func (c *Client) decodeJSON(resp *http.Response, payload interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	return json.Unmarshal(body, payload)
}

func (c *Client) getErrorFromResponse(resp *http.Response) APIError {
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		apierror := APIError{
			StatusCode: resp.StatusCode,
			message:    fmt.Sprintf("HTTP response with status code %d does not contain Content-Type: application/json", resp.StatusCode),
		}
		return apierror
	}
	return APIError{
		StatusCode: resp.StatusCode,
		message:    fmt.Sprintf("HTTP response with status code %d", resp.StatusCode),
	}
}
