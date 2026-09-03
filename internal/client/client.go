// Package client provides a minimal HTTP client for the Dub API
// (https://dub.co/docs/api-reference), scoped to the endpoints required to
// manage domains on a Dub workspace.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	defaultBaseURL        = "https://api.dub.co"
	defaultWebhookBaseURL = "https://app.dub.co/api"
)

// Client is a small wrapper around http.Client that knows how to talk to the
// Dub API for domain resources.
type Client struct {
	BaseURL     string
	APIKey      string
	WorkspaceID string
	HTTPClient  *http.Client
	UserAgent   string
	Webhooks    *WebhookClient
}

// New creates a new Dub API client. baseURL may be empty, in which case the
// default production API URL is used.
func New(baseURL, apiKey, workspaceID string, httpClient *http.Client) *Client {
	client := newClient(baseURL, apiKey, workspaceID, httpClient)
	client.Webhooks = NewWebhookClient(apiKey, workspaceID, httpClient)
	return client
}

func newClient(baseURL, apiKey, workspaceID string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		APIKey:      apiKey,
		WorkspaceID: workspaceID,
		HTTPClient:  httpClient,
		UserAgent:   "terraform-provider-dub",
	}
}

// WebhookClient is a dedicated client for Dub's app API webhook endpoints.
// These endpoints are distinct from the public API used by Client.
type WebhookClient struct {
	client *Client
}

// NewWebhookClient creates a client for Dub's app API webhook endpoints.
func NewWebhookClient(apiKey, workspaceID string, httpClient *http.Client) *WebhookClient {
	return NewWebhookClientWithBaseURL(defaultWebhookBaseURL, apiKey, workspaceID, httpClient)
}

// NewWebhookClientWithBaseURL creates a webhook client with a custom base URL.
// It is primarily useful for testing the app API integration.
func NewWebhookClientWithBaseURL(baseURL, apiKey, workspaceID string, httpClient *http.Client) *WebhookClient {
	return &WebhookClient{client: newClient(baseURL, apiKey, workspaceID, httpClient)}
}

// APIError represents an error response returned by the Dub API.
type APIError struct {
	StatusCode int
	Code       string `json:"code"`
	Message    string `json:"message"`
	Body       string `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("dub api error (status %d, code %s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("dub api error (status %d): %s", e.StatusCode, e.Body)
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Domain represents a Dub domain resource as returned by the Dub API.
type Domain struct {
	ID          string `json:"id,omitempty"`
	Slug        string `json:"slug"`
	Verified    bool   `json:"verified,omitempty"`
	Primary     bool   `json:"primary,omitempty"`
	Archived    bool   `json:"archived"`
	Placeholder string `json:"placeholder,omitempty"`
	ExpiredURL  string `json:"expiredUrl,omitempty"`
	NotFoundURL string `json:"notFoundUrl,omitempty"`
	Logo        string `json:"logo,omitempty"`
	LinksCount  int    `json:"linksCount,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// CreateDomainRequest is the request body used to create a new domain.
type CreateDomainRequest struct {
	Slug        string  `json:"slug"`
	ExpiredURL  *string `json:"expiredUrl,omitempty"`
	NotFoundURL *string `json:"notFoundUrl,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
	Placeholder *string `json:"placeholder,omitempty"`
	Logo        *string `json:"logo,omitempty"`
}

// UpdateDomainRequest is the request body used to update an existing domain.
// The domain slug cannot be changed after creation and is instead supplied
// as part of the request path.
type UpdateDomainRequest struct {
	Slug        *string `json:"slug,omitempty"`
	ExpiredURL  *string `json:"expiredUrl,omitempty"`
	NotFoundURL *string `json:"notFoundUrl,omitempty"`
	Archived    *bool   `json:"archived,omitempty"`
	Placeholder *string `json:"placeholder,omitempty"`
	Logo        *string `json:"logo,omitempty"`
}

// ListDomainsParams contains the optional query parameters supported by the
// list domains endpoint.
type ListDomainsParams struct {
	Search   string
	Archived *bool
	Page     int
	PageSize int
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Request, error) {
	u := c.BaseURL + path
	if len(query) > 0 {
		if c.WorkspaceID != "" && query.Get("workspaceId") == "" {
			query.Set("workspaceId", c.WorkspaceID)
		}
		u += "?" + query.Encode()
	} else if c.WorkspaceID != "" {
		u += "?" + url.Values{"workspaceId": []string{c.WorkspaceID}}.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		var env errorEnvelope
		if jsonErr := json.Unmarshal(respBody, &env); jsonErr == nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return apiErr
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decoding response body: %w", err)
	}

	return nil
}

// IsNotFound returns true if err represents a 404 Not Found response from
// the Dub API.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if !asAPIError(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound
}

// IsForbidden returns true if err represents a 403 Forbidden response from
// the Dub API.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if !asAPIError(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusForbidden
}

func asAPIError(err error, target **APIError) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	*target = apiErr
	return true
}

// CreateDomain creates a new domain in the configured workspace.
func (c *Client) CreateDomain(ctx context.Context, in CreateDomainRequest) (*Domain, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/domains", nil, in)
	if err != nil {
		return nil, err
	}

	var out Domain
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDomain retrieves a domain by its slug.
func (c *Client) GetDomain(ctx context.Context, slug string) (*Domain, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/domains/"+url.PathEscape(slug), nil, nil)
	if err != nil {
		return nil, err
	}

	var out Domain
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDomain updates an existing domain identified by slug.
func (c *Client) UpdateDomain(ctx context.Context, slug string, in UpdateDomainRequest) (*Domain, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, "/domains/"+url.PathEscape(slug), nil, in)
	if err != nil {
		return nil, err
	}

	var out Domain
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDomain deletes a domain identified by slug.
func (c *Client) DeleteDomain(ctx context.Context, slug string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/domains/"+url.PathEscape(slug), nil, nil)
	if err != nil {
		return err
	}

	return c.do(req, nil)
}

// ListDomains returns the list of domains in the configured workspace,
// optionally filtered/paginated via params.
func (c *Client) ListDomains(ctx context.Context, params ListDomainsParams) ([]Domain, error) {
	query := url.Values{}
	if params.Search != "" {
		query.Set("search", params.Search)
	}
	if params.Archived != nil {
		query.Set("archived", strconv.FormatBool(*params.Archived))
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.PageSize > 0 {
		query.Set("pageSize", strconv.Itoa(params.PageSize))
	}

	req, err := c.newRequest(ctx, http.MethodGet, "/domains", query, nil)
	if err != nil {
		return nil, err
	}

	var out []Domain
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Webhook represents a webhook returned by Dub's app API.
type Webhook struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Secret         string   `json:"secret"`
	Triggers       []string `json:"triggers"`
	LinkScope      *string  `json:"linkScope"`
	DisabledAt     *string  `json:"disabledAt"`
	InstallationID *string  `json:"installationId"`
}

// WebhookRequest is the request body used to create or update a webhook.
type WebhookRequest struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Triggers  []string `json:"triggers"`
	LinkScope *string  `json:"linkScope"`
	LinkIDs   []string `json:"linkIds,omitempty"`
	FolderIDs []string `json:"folderIds,omitempty"`
}

// WorkspaceID returns the workspace used by the webhook client.
func (c *WebhookClient) WorkspaceID() string {
	return c.client.WorkspaceID
}

func (c *WebhookClient) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	req, err := c.client.newRequest(ctx, method, path, nil, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// CreateWebhook creates a webhook in the configured workspace.
func (c *WebhookClient) CreateWebhook(ctx context.Context, in WebhookRequest) (*Webhook, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/webhooks", in)
	if err != nil {
		return nil, err
	}

	var out Webhook
	if err := c.client.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetWebhook retrieves a webhook by ID from the configured workspace.
func (c *WebhookClient) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/webhooks/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	var out Webhook
	if err := c.client.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateWebhook updates a webhook by ID in the configured workspace.
func (c *WebhookClient) UpdateWebhook(ctx context.Context, id string, in WebhookRequest) (*Webhook, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, "/webhooks/"+url.PathEscape(id), in)
	if err != nil {
		return nil, err
	}

	var out Webhook
	if err := c.client.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteWebhook deletes a webhook by ID from the configured workspace.
func (c *WebhookClient) DeleteWebhook(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, "/webhooks/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}

	return c.client.do(req, nil)
}
