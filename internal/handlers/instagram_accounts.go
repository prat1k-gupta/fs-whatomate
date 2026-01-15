package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// InstagramAccountRequest represents the request body for creating/updating an Instagram account
type InstagramAccountRequest struct {
	Name               string `json:"name" validate:"required"`
	InstagramAccountID string `json:"instagram_account_id" validate:"required"`
	PageID             string `json:"page_id" validate:"required"`
	AccessToken        string `json:"access_token" validate:"required"`
	WebhookVerifyToken string `json:"webhook_verify_token"`
	APIVersion         string `json:"api_version"`
	IsDefaultIncoming  bool   `json:"is_default_incoming"`
	IsDefaultOutgoing  bool   `json:"is_default_outgoing"`
}

// InstagramAccountResponse represents the response for an Instagram account (without sensitive data)
type InstagramAccountResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	InstagramAccountID string    `json:"instagram_account_id"`
	PageID             string    `json:"page_id"`
	WebhookVerifyToken string    `json:"webhook_verify_token"`
	APIVersion         string    `json:"api_version"`
	IsDefaultIncoming  bool      `json:"is_default_incoming"`
	IsDefaultOutgoing  bool      `json:"is_default_outgoing"`
	Status             string    `json:"status"`
	HasAccessToken     bool      `json:"has_access_token"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

// ListInstagramAccounts returns all Instagram accounts for the organization
func (a *App) ListInstagramAccounts(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var accounts []models.InstagramAccount
	if err := a.DB.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&accounts).Error; err != nil {
		a.Log.Error("Failed to list Instagram accounts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list accounts", nil, "")
	}

	// Convert to response format (hide sensitive data)
	response := make([]InstagramAccountResponse, len(accounts))
	for i, acc := range accounts {
		response[i] = instagramAccountToResponse(acc)
	}

	return r.SendEnvelope(map[string]interface{}{
		"accounts": response,
	})
}

// CreateInstagramAccount creates a new Instagram account
func (a *App) CreateInstagramAccount(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	var req InstagramAccountRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Validate required fields
	if req.Name == "" || req.InstagramAccountID == "" || req.PageID == "" || req.AccessToken == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Name, instagram_account_id, page_id, and access_token are required", nil, "")
	}

	// Generate webhook verify token if not provided
	webhookVerifyToken := req.WebhookVerifyToken
	if webhookVerifyToken == "" {
		webhookVerifyToken = generateInstagramVerifyToken()
	}

	// Set default API version
	apiVersion := req.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}

	account := models.InstagramAccount{
		OrganizationID:     orgID,
		Name:               req.Name,
		InstagramAccountID: req.InstagramAccountID,
		PageID:             req.PageID,
		AccessToken:        req.AccessToken, // TODO: encrypt before storing
		WebhookVerifyToken: webhookVerifyToken,
		APIVersion:         apiVersion,
		IsDefaultIncoming:  req.IsDefaultIncoming,
		IsDefaultOutgoing:  req.IsDefaultOutgoing,
		Status:             "active",
	}

	// If this is set as default, unset other defaults
	if req.IsDefaultIncoming {
		a.DB.Model(&models.InstagramAccount{}).
			Where("organization_id = ? AND is_default_incoming = ?", orgID, true).
			Update("is_default_incoming", false)
	}
	if req.IsDefaultOutgoing {
		a.DB.Model(&models.InstagramAccount{}).
			Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).
			Update("is_default_outgoing", false)
	}

	if err := a.DB.Create(&account).Error; err != nil {
		a.Log.Error("Failed to create Instagram account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create account", nil, "")
	}

	return r.SendEnvelope(instagramAccountToResponse(account))
}

// GetInstagramAccount returns a single Instagram account
func (a *App) GetInstagramAccount(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account ID", nil, "")
	}

	var account models.InstagramAccount
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&account).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
	}

	return r.SendEnvelope(instagramAccountToResponse(account))
}

// UpdateInstagramAccount updates an Instagram account
func (a *App) UpdateInstagramAccount(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr, ok := r.RequestCtx.UserValue("id").(string)
	if !ok || idStr == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing account ID", nil, "")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account ID", nil, "")
	}

	var account models.InstagramAccount
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&account).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
	}

	var req InstagramAccountRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	// Update fields if provided
	if req.Name != "" {
		account.Name = req.Name
	}
	if req.InstagramAccountID != "" {
		account.InstagramAccountID = req.InstagramAccountID
	}
	if req.PageID != "" {
		account.PageID = req.PageID
	}
	if req.AccessToken != "" {
		account.AccessToken = req.AccessToken // TODO: encrypt
	}
	if req.WebhookVerifyToken != "" {
		account.WebhookVerifyToken = req.WebhookVerifyToken
	}
	if req.APIVersion != "" {
		account.APIVersion = req.APIVersion
	}

	// Handle default flags
	if req.IsDefaultIncoming && !account.IsDefaultIncoming {
		a.DB.Model(&models.InstagramAccount{}).
			Where("organization_id = ? AND is_default_incoming = ?", orgID, true).
			Update("is_default_incoming", false)
	}
	if req.IsDefaultOutgoing && !account.IsDefaultOutgoing {
		a.DB.Model(&models.InstagramAccount{}).
			Where("organization_id = ? AND is_default_outgoing = ?", orgID, true).
			Update("is_default_outgoing", false)
	}
	account.IsDefaultIncoming = req.IsDefaultIncoming
	account.IsDefaultOutgoing = req.IsDefaultOutgoing

	if err := a.DB.Save(&account).Error; err != nil {
		a.Log.Error("Failed to update Instagram account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update account", nil, "")
	}

	// Invalidate cache
	a.InvalidateInstagramAccountCache(account.InstagramAccountID)

	return r.SendEnvelope(instagramAccountToResponse(account))
}

// DeleteInstagramAccount deletes an Instagram account
func (a *App) DeleteInstagramAccount(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account ID", nil, "")
	}

	// Get account first for cache invalidation
	var account models.InstagramAccount
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&account).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
	}

	if err := a.DB.Delete(&account).Error; err != nil {
		a.Log.Error("Failed to delete Instagram account", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete account", nil, "")
	}

	// Invalidate cache
	a.InvalidateInstagramAccountCache(account.InstagramAccountID)

	return r.SendEnvelope(map[string]string{"message": "Account deleted successfully"})
}

// TestInstagramAccountConnection tests the Instagram API connection
func (a *App) TestInstagramAccountConnection(r *fastglue.Request) error {
	orgID, err := getOrganizationID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid account ID", nil, "")
	}

	var account models.InstagramAccount
	if err := a.DB.Where("id = ? AND organization_id = ?", id, orgID).First(&account).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Account not found", nil, "")
	}

	// Test the connection by fetching Instagram account details from Meta API
	url := fmt.Sprintf("%s/%s/%s?fields=id,name,username,profile_picture_url,followers_count",
		a.Config.Instagram.BaseURL, account.APIVersion, account.InstagramAccountID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return r.SendEnvelope(map[string]interface{}{
			"success": false,
			"error":   "Failed to connect to Instagram API: " + err.Error(),
		})
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		var errorResp map[string]interface{}
		_ = json.Unmarshal(body, &errorResp)
		return r.SendEnvelope(map[string]interface{}{
			"success": false,
			"error":   "API error",
			"details": errorResp,
		})
	}

	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)

	return r.SendEnvelope(map[string]interface{}{
		"success":            true,
		"id":                 result["id"],
		"name":               result["name"],
		"username":           result["username"],
		"profile_picture":    result["profile_picture_url"],
		"followers_count":    result["followers_count"],
	})
}

// Helper functions

func instagramAccountToResponse(acc models.InstagramAccount) InstagramAccountResponse {
	return InstagramAccountResponse{
		ID:                 acc.ID,
		Name:               acc.Name,
		InstagramAccountID: acc.InstagramAccountID,
		PageID:             acc.PageID,
		WebhookVerifyToken: acc.WebhookVerifyToken,
		APIVersion:         acc.APIVersion,
		IsDefaultIncoming:  acc.IsDefaultIncoming,
		IsDefaultOutgoing:  acc.IsDefaultOutgoing,
		Status:             acc.Status,
		HasAccessToken:     acc.AccessToken != "",
		CreatedAt:          acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          acc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func generateInstagramVerifyToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// InvalidateInstagramAccountCache invalidates the cache for an Instagram account
func (a *App) InvalidateInstagramAccountCache(instagramAccountID string) {
	// Similar to WhatsApp cache invalidation
	cacheKey := fmt.Sprintf("instagram_account:%s", instagramAccountID)
	if a.Redis != nil {
		ctx := context.Background()
		_ = a.Redis.Del(ctx, cacheKey).Err()
	}
}

// getInstagramAccountCached retrieves an Instagram account from cache or database
func (a *App) getInstagramAccountCached(instagramAccountID string) (*models.InstagramAccount, error) {
	cacheKey := fmt.Sprintf("instagram_account:%s", instagramAccountID)
	ctx := context.Background()

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var account models.InstagramAccount
			if err := json.Unmarshal([]byte(cached), &account); err == nil {
				return &account, nil
			}
		}
	}

	// Query database
	var account models.InstagramAccount
	if err := a.DB.Where("instagram_account_id = ?", instagramAccountID).First(&account).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(account); err == nil {
			_ = a.Redis.Set(ctx, cacheKey, data, 0).Err()
		}
	}

	return &account, nil
}

// getInstagramAccountByPageIDCached retrieves an Instagram account by Page ID from cache or database
func (a *App) getInstagramAccountByPageIDCached(pageID string) (*models.InstagramAccount, error) {
	cacheKey := fmt.Sprintf("instagram_account_page:%s", pageID)
	ctx := context.Background()

	// Try cache first
	if a.Redis != nil {
		cached, err := a.Redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var account models.InstagramAccount
			if err := json.Unmarshal([]byte(cached), &account); err == nil {
				return &account, nil
			}
		}
	}

	// Query database
	var account models.InstagramAccount
	if err := a.DB.Where("page_id = ?", pageID).First(&account).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if a.Redis != nil {
		if data, err := json.Marshal(account); err == nil {
			_ = a.Redis.Set(ctx, cacheKey, data, 0).Err()
		}
	}

	return &account, nil
}
