package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/zerodha/logf"
)

const (
	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second
	// BaseURL for Meta Graph API
	BaseURL = "https://graph.facebook.com"
)

// Client is the Instagram Messaging API client
type Client struct {
	HTTPClient *http.Client
	Log        logf.Logger
	baseURL    string // For testing with mock servers
}

// New creates a new Instagram client
func New(log logf.Logger) *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		Log:     log,
		baseURL: BaseURL,
	}
}

// NewWithTimeout creates a new Instagram client with custom timeout
func NewWithTimeout(log logf.Logger, timeout time.Duration) *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Log:     log,
		baseURL: BaseURL,
	}
}

// NewWithBaseURL creates a new Instagram client with a custom base URL (for testing)
func NewWithBaseURL(log logf.Logger, baseURL string) *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		Log:     log,
		baseURL: baseURL,
	}
}

// getBaseURL returns the base URL for API requests
func (c *Client) getBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return BaseURL
}

// doRequest performs an HTTP request to the Meta API
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}, accessToken string) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr MetaAPIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("API error %d: %s (code: %d)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Code)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// buildMessagesURL builds the messages endpoint URL for Instagram
// Instagram uses the Page ID to send messages
func (c *Client) buildMessagesURL(account *Account) string {
	return fmt.Sprintf("%s/%s/%s/messages", c.getBaseURL(), account.APIVersion, account.PageID)
}

// GetUserProfile retrieves a user's profile information
func (c *Client) GetUserProfile(ctx context.Context, account *Account, userID string) (*UserProfile, error) {
	// Trim whitespace from access token
	cleanToken := strings.TrimSpace(account.AccessToken)

	// Determine the correct API base URL based on token type
	// Instagram User Access Tokens (start with "IG") use graph.instagram.com
	// Page Access Tokens (start with "EAA") use graph.facebook.com
	apiBase := c.getBaseURL() // defaults to graph.facebook.com
	if strings.HasPrefix(cleanToken, "IG") {
		apiBase = "https://graph.instagram.com"
	}

	// Build URL with proper query encoding
	// Use only basic fields that are available without extra permissions
	baseURL := fmt.Sprintf("%s/%s/%s", apiBase, account.APIVersion, userID)
	params := neturl.Values{}
	params.Set("fields", "id,name,username")
	params.Set("access_token", cleanToken)
	fullURL := baseURL + "?" + params.Encode()

	c.Log.Info("Fetching Instagram user profile", "user_id", userID, "api_version", account.APIVersion, "api_base", apiBase, "token_length", len(cleanToken))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Log.Error("Failed to fetch Instagram user profile", "error", err)
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.Log.Info("Instagram user profile response", "status", resp.StatusCode, "response", string(respBody))

	if resp.StatusCode != http.StatusOK {
		var apiErr MetaAPIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			c.Log.Error("Instagram API error", "code", apiErr.Error.Code, "message", apiErr.Error.Message)
			return nil, fmt.Errorf("API error %d: %s (code: %d)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Code)
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var profile UserProfile
	if err := json.Unmarshal(respBody, &profile); err != nil {
		c.Log.Error("Failed to parse Instagram user profile", "error", err, "response", string(respBody))
		return nil, fmt.Errorf("failed to parse user profile: %w", err)
	}

	c.Log.Info("Parsed Instagram user profile", "id", profile.ID, "name", profile.Name, "username", profile.Username)

	return &profile, nil
}

// DownloadMedia downloads media content from a URL
func (c *Client) DownloadMedia(ctx context.Context, mediaURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("media download failed with status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read media content: %w", err)
	}

	return data, contentType, nil
}
