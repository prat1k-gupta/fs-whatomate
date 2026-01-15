package instagram

import (
	"encoding/json"
	"fmt"
	"time"
)

// ParseWebhook parses the incoming Instagram webhook payload
func ParseWebhook(payload []byte) (*WebhookPayload, error) {
	var webhookPayload WebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Validate it's an Instagram webhook
	if webhookPayload.Object != "instagram" {
		return nil, fmt.Errorf("invalid webhook object: expected 'instagram', got '%s'", webhookPayload.Object)
	}

	return &webhookPayload, nil
}

// ExtractMessages extracts all messages from a webhook payload
func ExtractMessages(payload *WebhookPayload) []ParsedMessage {
	var messages []ParsedMessage

	for _, entry := range payload.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Message == nil {
				continue
			}

			msg := messaging.Message

			// Skip echo messages (messages sent by us)
			if msg.IsEcho {
				continue
			}

			// Skip deleted messages
			if msg.IsDeleted {
				continue
			}

			parsed := ParsedMessage{
				SenderID:      messaging.Sender.ID,
				RecipientID:   messaging.Recipient.ID,
				MessageID:     msg.Mid,
				Timestamp:     time.Unix(messaging.Timestamp/1000, 0),
				IsEcho:        msg.IsEcho,
				IsDeleted:     msg.IsDeleted,
				IsUnsupported: msg.IsUnsupported,
			}

			// Handle text messages
			if msg.Text != "" {
				parsed.Type = "text"
				parsed.Text = msg.Text
			}

			// Handle attachments
			if len(msg.Attachments) > 0 {
				attachment := msg.Attachments[0] // Primary attachment
				parsed.Type = attachment.Type
				parsed.MediaURL = attachment.Payload.URL
				parsed.MediaType = attachment.Type
			}

			// Handle reply context
			if msg.ReplyTo != nil {
				if msg.ReplyTo.Mid != "" {
					parsed.ReplyToMid = msg.ReplyTo.Mid
				}
				if msg.ReplyTo.Story != nil {
					parsed.Type = "story_reply"
					parsed.StoryURL = msg.ReplyTo.Story.URL
					parsed.StoryID = msg.ReplyTo.Story.ID
				}
			}

			messages = append(messages, parsed)
		}
	}

	return messages
}

// ExtractReactions extracts all reaction events from a webhook payload
func ExtractReactions(payload *WebhookPayload) []ParsedReaction {
	var reactions []ParsedReaction

	for _, entry := range payload.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Reaction == nil {
				continue
			}

			reaction := messaging.Reaction
			parsed := ParsedReaction{
				SenderID:  messaging.Sender.ID,
				MessageID: reaction.Mid,
				Action:    reaction.Action,
				Reaction:  reaction.Reaction,
				Emoji:     reaction.Emoji,
				Timestamp: time.Unix(messaging.Timestamp/1000, 0),
			}

			reactions = append(reactions, parsed)
		}
	}

	return reactions
}

// ExtractReadReceipts extracts read receipt events
func ExtractReadReceipts(payload *WebhookPayload) []struct {
	SenderID  string
	Watermark time.Time
} {
	var receipts []struct {
		SenderID  string
		Watermark time.Time
	}

	for _, entry := range payload.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Read == nil {
				continue
			}

			receipts = append(receipts, struct {
				SenderID  string
				Watermark time.Time
			}{
				SenderID:  messaging.Sender.ID,
				Watermark: time.Unix(messaging.Read.Watermark/1000, 0),
			})
		}
	}

	return receipts
}

// ExtractDeliveryReceipts extracts delivery receipt events
func ExtractDeliveryReceipts(payload *WebhookPayload) []struct {
	SenderID   string
	MessageIDs []string
	Watermark  time.Time
} {
	var receipts []struct {
		SenderID   string
		MessageIDs []string
		Watermark  time.Time
	}

	for _, entry := range payload.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Delivery == nil {
				continue
			}

			receipts = append(receipts, struct {
				SenderID   string
				MessageIDs []string
				Watermark  time.Time
			}{
				SenderID:   messaging.Sender.ID,
				MessageIDs: messaging.Delivery.Mids,
				Watermark:  time.Unix(messaging.Delivery.Watermark/1000, 0),
			})
		}
	}

	return receipts
}

// GetMessageType returns a standardized message type from attachment type
func GetMessageType(attachmentType string) string {
	switch attachmentType {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "file":
		return "document"
	case "share":
		return "share"
	case "story_mention":
		return "story_mention"
	default:
		return "unknown"
	}
}

// IsValidWebhook checks if the webhook payload is valid for Instagram
func IsValidWebhook(payload *WebhookPayload) bool {
	return payload != nil && payload.Object == "instagram" && len(payload.Entry) > 0
}

// GetAccountIDFromEntry extracts the Instagram account ID from a webhook entry
func GetAccountIDFromEntry(entry *WebhookEntry) string {
	return entry.ID
}

// HasMessages checks if the webhook contains any messages
func HasMessages(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Message != nil {
				return true
			}
		}
	}
	return false
}

// HasReactions checks if the webhook contains any reaction events
func HasReactions(payload *WebhookPayload) bool {
	for _, entry := range payload.Entry {
		for _, messaging := range entry.Messaging {
			if messaging.Reaction != nil {
				return true
			}
		}
	}
	return false
}
