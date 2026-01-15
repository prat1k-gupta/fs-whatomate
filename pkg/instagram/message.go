package instagram

import (
	"context"
	"encoding/json"
	"fmt"
)

// SendTextMessage sends a text message to an Instagram user
func (c *Client) SendTextMessage(ctx context.Context, account *Account, recipientID, text string) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]string{
			"text": text,
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Sending Instagram text message", "recipient", recipientID, "url", url)

	respBody, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to send Instagram text message", "error", err, "recipient", recipientID)
		return "", fmt.Errorf("failed to send text message: %w", err)
	}

	var resp MetaAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.MessageID == "" {
		return "", fmt.Errorf("no message ID in response")
	}

	c.Log.Info("Instagram text message sent", "message_id", resp.MessageID, "recipient", recipientID)
	return resp.MessageID, nil
}

// SendImageMessage sends an image message using a URL
func (c *Client) SendImageMessage(ctx context.Context, account *Account, recipientID, imageURL string) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "image",
				"payload": map[string]interface{}{
					"url":         imageURL,
					"is_reusable": true,
				},
			},
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Sending Instagram image message", "recipient", recipientID, "image_url", imageURL)

	respBody, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to send Instagram image message", "error", err, "recipient", recipientID)
		return "", fmt.Errorf("failed to send image message: %w", err)
	}

	var resp MetaAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.MessageID == "" {
		return "", fmt.Errorf("no message ID in response")
	}

	c.Log.Info("Instagram image message sent", "message_id", resp.MessageID, "recipient", recipientID)
	return resp.MessageID, nil
}

// SendVideoMessage sends a video message using a URL
func (c *Client) SendVideoMessage(ctx context.Context, account *Account, recipientID, videoURL string) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "video",
				"payload": map[string]interface{}{
					"url":         videoURL,
					"is_reusable": true,
				},
			},
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Sending Instagram video message", "recipient", recipientID, "video_url", videoURL)

	respBody, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to send Instagram video message", "error", err, "recipient", recipientID)
		return "", fmt.Errorf("failed to send video message: %w", err)
	}

	var resp MetaAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.MessageID == "" {
		return "", fmt.Errorf("no message ID in response")
	}

	c.Log.Info("Instagram video message sent", "message_id", resp.MessageID, "recipient", recipientID)
	return resp.MessageID, nil
}

// SendAudioMessage sends an audio message using a URL
func (c *Client) SendAudioMessage(ctx context.Context, account *Account, recipientID, audioURL string) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "audio",
				"payload": map[string]interface{}{
					"url":         audioURL,
					"is_reusable": true,
				},
			},
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Sending Instagram audio message", "recipient", recipientID, "audio_url", audioURL)

	respBody, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to send Instagram audio message", "error", err, "recipient", recipientID)
		return "", fmt.Errorf("failed to send audio message: %w", err)
	}

	var resp MetaAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.MessageID == "" {
		return "", fmt.Errorf("no message ID in response")
	}

	c.Log.Info("Instagram audio message sent", "message_id", resp.MessageID, "recipient", recipientID)
	return resp.MessageID, nil
}

// SendFileMessage sends a file message using a URL
func (c *Client) SendFileMessage(ctx context.Context, account *Account, recipientID, fileURL string) (string, error) {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "file",
				"payload": map[string]interface{}{
					"url":         fileURL,
					"is_reusable": true,
				},
			},
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Sending Instagram file message", "recipient", recipientID, "file_url", fileURL)

	respBody, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to send Instagram file message", "error", err, "recipient", recipientID)
		return "", fmt.Errorf("failed to send file message: %w", err)
	}

	var resp MetaAPIResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.MessageID == "" {
		return "", fmt.Errorf("no message ID in response")
	}

	c.Log.Info("Instagram file message sent", "message_id", resp.MessageID, "recipient", recipientID)
	return resp.MessageID, nil
}

// SendReaction sends a reaction to a message
// Note: Instagram Messaging API supports reactions via a specific endpoint
func (c *Client) SendReaction(ctx context.Context, account *Account, recipientID, messageID, reaction string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"sender_action": "react",
		"payload": map[string]string{
			"message_id": messageID,
			"reaction":   reaction,
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Sending Instagram reaction", "recipient", recipientID, "message_id", messageID, "reaction", reaction)

	_, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to send Instagram reaction", "error", err, "recipient", recipientID)
		return fmt.Errorf("failed to send reaction: %w", err)
	}

	c.Log.Info("Instagram reaction sent", "message_id", messageID, "reaction", reaction)
	return nil
}

// RemoveReaction removes a reaction from a message
func (c *Client) RemoveReaction(ctx context.Context, account *Account, recipientID, messageID string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"sender_action": "unreact",
		"payload": map[string]string{
			"message_id": messageID,
		},
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Removing Instagram reaction", "recipient", recipientID, "message_id", messageID)

	_, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to remove Instagram reaction", "error", err, "recipient", recipientID)
		return fmt.Errorf("failed to remove reaction: %w", err)
	}

	c.Log.Info("Instagram reaction removed", "message_id", messageID)
	return nil
}

// MarkSeen marks messages as seen (sends read receipt)
func (c *Client) MarkSeen(ctx context.Context, account *Account, recipientID string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"sender_action": "mark_seen",
	}

	url := c.buildMessagesURL(account)
	c.Log.Debug("Marking messages as seen", "recipient", recipientID)

	_, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		c.Log.Error("Failed to mark messages as seen", "error", err, "recipient", recipientID)
		return fmt.Errorf("failed to mark as seen: %w", err)
	}

	c.Log.Debug("Messages marked as seen", "recipient", recipientID)
	return nil
}

// SendTypingOn sends typing indicator (shows "typing..." to user)
func (c *Client) SendTypingOn(ctx context.Context, account *Account, recipientID string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"sender_action": "typing_on",
	}

	url := c.buildMessagesURL(account)

	_, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to send typing indicator: %w", err)
	}

	return nil
}

// SendTypingOff turns off typing indicator
func (c *Client) SendTypingOff(ctx context.Context, account *Account, recipientID string) error {
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientID,
		},
		"sender_action": "typing_off",
	}

	url := c.buildMessagesURL(account)

	_, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to turn off typing indicator: %w", err)
	}

	return nil
}

// IceBreaker represents a conversation starter/ice breaker
type IceBreaker struct {
	Question string `json:"question"`
	Payload  string `json:"payload"`
}

// SetIceBreakers sets the ice breakers (conversation starters) for the Instagram account
func (c *Client) SetIceBreakers(ctx context.Context, account *Account, iceBreakers []IceBreaker) error {
	url := fmt.Sprintf("%s/%s/%s/messenger_profile", c.getBaseURL(), account.APIVersion, account.PageID)

	payload := map[string]interface{}{
		"platform":    "instagram",
		"ice_breakers": iceBreakers,
	}

	_, err := c.doRequest(ctx, "POST", url, payload, account.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to set ice breakers: %w", err)
	}

	c.Log.Info("Ice breakers set successfully", "count", len(iceBreakers))
	return nil
}

// DeleteIceBreakers removes all ice breakers
func (c *Client) DeleteIceBreakers(ctx context.Context, account *Account) error {
	url := fmt.Sprintf("%s/%s/%s/messenger_profile?fields=ice_breakers&platform=instagram",
		c.getBaseURL(), account.APIVersion, account.PageID)

	_, err := c.doRequest(ctx, "DELETE", url, nil, account.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to delete ice breakers: %w", err)
	}

	c.Log.Info("Ice breakers deleted successfully")
	return nil
}
