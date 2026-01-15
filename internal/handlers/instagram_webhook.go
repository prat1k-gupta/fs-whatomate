package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/internal/websocket"
	"github.com/shridarpatil/whatomate/pkg/instagram"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// InstagramWebhookVerify handles Meta's webhook verification challenge for Instagram
func (a *App) InstagramWebhookVerify(r *fastglue.Request) error {
	mode := string(r.RequestCtx.QueryArgs().Peek("hub.mode"))
	token := string(r.RequestCtx.QueryArgs().Peek("hub.verify_token"))
	challenge := string(r.RequestCtx.QueryArgs().Peek("hub.challenge"))

	if mode != "subscribe" {
		a.Log.Warn("Instagram webhook verification failed - invalid mode", "mode", mode)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Verification failed", nil, "")
	}

	// First check against global config token
	if token == a.Config.Instagram.WebhookVerifyToken && token != "" {
		a.Log.Info("Instagram webhook verified successfully (global token)")
		r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
		r.RequestCtx.SetBodyString(challenge)
		return nil
	}

	// Then check against tokens stored in Instagram accounts
	var account models.InstagramAccount
	result := a.DB.Where("webhook_verify_token = ?", token).First(&account)
	if result.Error == nil {
		a.Log.Info("Instagram webhook verified successfully (account token)", "account", account.Name)
		r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
		r.RequestCtx.SetBodyString(challenge)
		return nil
	}

	a.Log.Warn("Instagram webhook verification failed - token not found", "token", token)
	return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Verification failed", nil, "")
}

// InstagramWebhookHandler processes incoming webhook events from Meta for Instagram
func (a *App) InstagramWebhookHandler(r *fastglue.Request) error {
	payload, err := instagram.ParseWebhook(r.RequestCtx.PostBody())
	if err != nil {
		a.Log.Error("Failed to parse Instagram webhook payload", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid payload", nil, "")
	}

	// Process each entry
	for _, entry := range payload.Entry {
		pageID := entry.ID // The entry ID is the Page ID

		// Find the Instagram account by page ID
		account, err := a.getInstagramAccountByPageIDCached(pageID)
		if err != nil {
			a.Log.Error("Instagram account not found for page", "page_id", pageID, "error", err)
			continue
		}

		// Process messaging events
		for _, messaging := range entry.Messaging {
			// Handle incoming messages
			if messaging.Message != nil && !messaging.Message.IsEcho {
				a.Log.Info("Received Instagram message",
					"from", messaging.Sender.ID,
					"type", getInstagramMessageType(messaging.Message),
					"page_id", pageID,
				)

				// Process message asynchronously
				go a.processInstagramMessage(account, messaging)
			}

			// Handle reactions
			if messaging.Reaction != nil {
				a.Log.Info("Received Instagram reaction",
					"from", messaging.Sender.ID,
					"message_id", messaging.Reaction.Mid,
					"action", messaging.Reaction.Action,
				)

				go a.processInstagramReaction(account, messaging)
			}

			// Handle read receipts
			if messaging.Read != nil {
				a.Log.Debug("Received Instagram read receipt",
					"from", messaging.Sender.ID,
					"watermark", messaging.Read.Watermark,
				)
				// Could update message status to "read" here
			}

			// Handle delivery receipts
			if messaging.Delivery != nil {
				a.Log.Debug("Received Instagram delivery receipt",
					"from", messaging.Sender.ID,
					"message_ids", messaging.Delivery.Mids,
				)
				// Could update message status to "delivered" here
			}
		}
	}

	// Always respond with 200 to acknowledge receipt
	return r.SendEnvelope(map[string]string{"status": "ok"})
}

// processInstagramMessage processes an incoming Instagram message
func (a *App) processInstagramMessage(account *models.InstagramAccount, messaging instagram.WebhookMessaging) {
	msg := messaging.Message
	senderID := messaging.Sender.ID

	// Check for duplicate message
	if msg.Mid != "" {
		var existingMsg models.Message
		if err := a.DB.Where("instagram_message_id = ?", msg.Mid).First(&existingMsg).Error; err == nil {
			a.Log.Debug("Duplicate Instagram message detected, skipping", "message_id", msg.Mid)
			return
		}
	}

	// Get or create contact
	contact, isNewContact := a.getOrCreateInstagramContact(account.OrganizationID, senderID, account.Name)

	// Dispatch webhook if new contact was created
	if isNewContact {
		a.DispatchWebhook(account.OrganizationID, models.WebhookEventContactCreated, ContactEventData{
			ContactID:       contact.ID.String(),
			ContactPhone:    senderID, // Using IGSID as identifier
			ContactName:     contact.ProfileName,
			WhatsAppAccount: "", // Not applicable
		})
	}

	// Determine message type and content
	messageType := models.MessageTypeText
	content := msg.Text
	var mediaURL, mediaMimeType string

	if len(msg.Attachments) > 0 {
		attachment := msg.Attachments[0]
		switch attachment.Type {
		case "image":
			messageType = models.MessageTypeImage
			mediaURL = attachment.Payload.URL
			mediaMimeType = "image/*"
		case "video":
			messageType = models.MessageTypeVideo
			mediaURL = attachment.Payload.URL
			mediaMimeType = "video/*"
		case "audio":
			messageType = models.MessageTypeAudio
			mediaURL = attachment.Payload.URL
			mediaMimeType = "audio/*"
		case "file":
			messageType = models.MessageTypeDocument
			mediaURL = attachment.Payload.URL
		case "share", "story_mention":
			messageType = models.MessageTypeText
			content = "[Shared content]"
			if attachment.Payload.URL != "" {
				mediaURL = attachment.Payload.URL
			}
		}
	}

	// Handle story replies
	if msg.ReplyTo != nil && msg.ReplyTo.Story != nil {
		messageType = models.MessageTypeText
		content = "[Story reply] " + content
	}

	// Create message record
	message := models.Message{
		BaseModel:          models.BaseModel{ID: uuid.New()},
		OrganizationID:     account.OrganizationID,
		Channel:            models.ChannelInstagram,
		InstagramAccount:   account.Name,
		ContactID:          contact.ID,
		InstagramMessageID: msg.Mid,
		Direction:          models.DirectionIncoming,
		MessageType:        messageType,
		Content:            content,
		MediaURL:           mediaURL,
		MediaMimeType:      mediaMimeType,
		Status:             models.MessageStatusReceived,
	}

	// Handle reply context
	if msg.ReplyTo != nil && msg.ReplyTo.Mid != "" {
		var replyToMsg models.Message
		if err := a.DB.Where("instagram_message_id = ?", msg.ReplyTo.Mid).First(&replyToMsg).Error; err == nil {
			message.IsReply = true
			message.ReplyToMessageID = &replyToMsg.ID
		}
	}

	if err := a.DB.Create(&message).Error; err != nil {
		a.Log.Error("Failed to create Instagram message", "error", err)
		return
	}

	// Update contact's last message
	now := time.Now()
	preview := truncateString(content, 100)
	if preview == "" && messageType != models.MessageTypeText {
		preview = "[" + string(messageType) + "]"
	}
	a.DB.Model(contact).Updates(map[string]interface{}{
		"last_message_at":      now,
		"last_message_preview": preview,
		"is_read":              false,
	})

	// Broadcast via WebSocket
	if a.WSHub != nil {
		a.broadcastInstagramMessage(account.OrganizationID, &message, contact)
	}

	// Dispatch incoming message webhook
	a.DispatchWebhook(account.OrganizationID, models.WebhookEventMessageIncoming, MessageEventData{
		MessageID:       message.ID.String(),
		ContactID:       contact.ID.String(),
		ContactPhone:    senderID,
		ContactName:     contact.ProfileName,
		MessageType:     message.MessageType,
		Content:         content,
		WhatsAppAccount: "", // Not applicable
		Direction:       models.DirectionIncoming,
		Channel:         "instagram",
	})

	a.Log.Info("Instagram message processed",
		"message_id", message.ID,
		"ig_message_id", msg.Mid,
		"contact_id", contact.ID,
		"type", messageType,
	)
}

// processInstagramReaction processes an Instagram reaction event
func (a *App) processInstagramReaction(account *models.InstagramAccount, messaging instagram.WebhookMessaging) {
	reaction := messaging.Reaction
	senderID := messaging.Sender.ID

	// Find the message being reacted to
	var message models.Message
	if err := a.DB.Where("instagram_message_id = ?", reaction.Mid).First(&message).Error; err != nil {
		a.Log.Warn("Message not found for Instagram reaction", "message_id", reaction.Mid)
		return
	}

	// Get or create contact (for reaction sender)
	contact, _ := a.getOrCreateInstagramContact(account.OrganizationID, senderID, account.Name)

	// Update message metadata with reaction
	if message.Metadata == nil {
		message.Metadata = models.JSONB{}
	}

	reactions, ok := message.Metadata["reactions"].([]interface{})
	if !ok {
		reactions = []interface{}{}
	}

	if reaction.Action == "react" {
		// Add reaction
		newReaction := map[string]interface{}{
			"emoji":    reaction.Emoji,
			"reaction": reaction.Reaction,
			"from":     senderID,
		}
		reactions = append(reactions, newReaction)
	} else if reaction.Action == "unreact" {
		// Remove reaction
		newReactions := []interface{}{}
		for _, r := range reactions {
			if rm, ok := r.(map[string]interface{}); ok {
				if rm["from"] != senderID {
					newReactions = append(newReactions, r)
				}
			}
		}
		reactions = newReactions
	}

	message.Metadata["reactions"] = reactions
	a.DB.Save(&message)

	// Broadcast reaction update via WebSocket
	if a.WSHub != nil {
		a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
			Type: "instagram_reaction",
			Payload: map[string]interface{}{
				"message_id": message.ID.String(),
				"contact_id": contact.ID.String(),
				"action":     reaction.Action,
				"emoji":      reaction.Emoji,
				"from":       senderID,
			},
		})
	}
}

// getOrCreateInstagramContact gets or creates an Instagram contact
func (a *App) getOrCreateInstagramContact(orgID uuid.UUID, igsID, accountName string) (*models.Contact, bool) {
	var contact models.Contact
	
	// Try to find existing contact by IGSID and account
	err := a.DB.Where(
		"organization_id = ? AND channel = ? AND channel_identifier = ? AND instagram_account = ?",
		orgID, models.ChannelInstagram, igsID, accountName,
	).First(&contact).Error

	if err == nil {
		return &contact, false
	}

	// Create new contact
	contact = models.Contact{
		BaseModel:         models.BaseModel{ID: uuid.New()},
		OrganizationID:    orgID,
		Channel:           models.ChannelInstagram,
		ChannelIdentifier: igsID,
		InstagramAccount:  accountName,
		ProfileName:       "", // Will be populated when we fetch user profile
		IsRead:            false,
	}

	// Try to fetch user profile for name
	if a.Instagram != nil {
		igAccount, err := a.getInstagramAccountCached(accountName)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			account := &instagram.Account{
				InstagramAccountID: igAccount.InstagramAccountID,
				PageID:             igAccount.PageID,
				APIVersion:         igAccount.APIVersion,
				AccessToken:        igAccount.AccessToken,
			}

			profile, err := a.Instagram.GetUserProfile(ctx, account, igsID)
			if err == nil && profile != nil {
				contact.ProfileName = profile.Name
				if contact.ProfileName == "" && profile.Username != "" {
					contact.ProfileName = "@" + profile.Username
				}
			}
		}
	}

	if err := a.DB.Create(&contact).Error; err != nil {
		a.Log.Error("Failed to create Instagram contact", "error", err)
		// Try to fetch again in case of race condition
		a.DB.Where(
			"organization_id = ? AND channel = ? AND channel_identifier = ?",
			orgID, models.ChannelInstagram, igsID,
		).First(&contact)
		return &contact, false
	}

	return &contact, true
}

// broadcastInstagramMessage broadcasts a new Instagram message via WebSocket
func (a *App) broadcastInstagramMessage(orgID uuid.UUID, msg *models.Message, contact *models.Contact) {
	payload := map[string]interface{}{
		"id":           msg.ID,
		"contact_id":   contact.ID.String(),
		"channel":      "instagram",
		"direction":    msg.Direction,
		"message_type": msg.MessageType,
		"content":      map[string]string{"body": msg.Content},
		"status":       msg.Status,
		"created_at":   msg.CreatedAt,
		"updated_at":   msg.UpdatedAt,
	}

	if contact.AssignedUserID != nil {
		payload["assigned_user_id"] = contact.AssignedUserID.String()
	}
	payload["profile_name"] = contact.ProfileName

	if msg.MediaURL != "" {
		payload["media_url"] = msg.MediaURL
		payload["media_mime_type"] = msg.MediaMimeType
	}

	a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
		Type:    websocket.TypeNewMessage,
		Payload: payload,
	})
}

// getInstagramMessageType returns the message type string
func getInstagramMessageType(msg *instagram.WebhookMessage) string {
	if msg.Text != "" {
		return "text"
	}
	if len(msg.Attachments) > 0 {
		return msg.Attachments[0].Type
	}
	return "unknown"
}

// MessageEventData extended with Channel field (for webhooks)
type MessageEventDataWithChannel struct {
	MessageEventData
	Channel string `json:"channel,omitempty"`
}
