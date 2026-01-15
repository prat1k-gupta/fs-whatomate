package instagram

import "time"

// Account represents Instagram Business Account credentials
type Account struct {
	InstagramAccountID string
	PageID             string
	APIVersion         string
	AccessToken        string
}

// MetaAPIResponse represents a successful API response from Meta
type MetaAPIResponse struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
}

// MetaAPIError represents an error response from Meta API
type MetaAPIError struct {
	Error struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		FBTraceID    string `json:"fbtrace_id"`
	} `json:"error"`
}

// WebhookPayload represents the incoming webhook from Meta for Instagram
type WebhookPayload struct {
	Object string         `json:"object"` // "instagram"
	Entry  []WebhookEntry `json:"entry"`
}

// WebhookEntry represents an entry in the webhook payload
type WebhookEntry struct {
	ID        string            `json:"id"` // Instagram Business Account ID or Page ID
	Time      int64             `json:"time"`
	Messaging []WebhookMessaging `json:"messaging,omitempty"`
}

// WebhookMessaging represents a messaging event
type WebhookMessaging struct {
	Sender    WebhookParticipant `json:"sender"`
	Recipient WebhookParticipant `json:"recipient"`
	Timestamp int64              `json:"timestamp"`
	Message   *WebhookMessage    `json:"message,omitempty"`
	Reaction  *WebhookReaction   `json:"reaction,omitempty"`
	Read      *WebhookRead       `json:"read,omitempty"`
	Delivery  *WebhookDelivery   `json:"delivery,omitempty"`
}

// WebhookParticipant represents sender or recipient
type WebhookParticipant struct {
	ID string `json:"id"` // Instagram-scoped ID (IGSID)
}

// WebhookMessage represents an incoming message
type WebhookMessage struct {
	Mid         string              `json:"mid"` // Message ID
	Text        string              `json:"text,omitempty"`
	Attachments []WebhookAttachment `json:"attachments,omitempty"`
	ReplyTo     *WebhookReplyTo     `json:"reply_to,omitempty"`
	IsDeleted   bool                `json:"is_deleted,omitempty"`
	IsEcho      bool                `json:"is_echo,omitempty"`
	IsUnsupported bool              `json:"is_unsupported,omitempty"`
}

// WebhookAttachment represents media attachment in a message
type WebhookAttachment struct {
	Type    string                 `json:"type"` // image, video, audio, file, share, story_mention
	Payload WebhookAttachmentPayload `json:"payload"`
}

// WebhookAttachmentPayload contains attachment details
type WebhookAttachmentPayload struct {
	URL       string `json:"url,omitempty"`
	StickerID string `json:"sticker_id,omitempty"`
}

// WebhookReplyTo represents reply context
type WebhookReplyTo struct {
	Mid   string             `json:"mid,omitempty"`   // Message ID being replied to
	Story *WebhookStoryReply `json:"story,omitempty"` // If replying to a story
}

// WebhookStoryReply represents a story reply
type WebhookStoryReply struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

// WebhookReaction represents a reaction to a message
type WebhookReaction struct {
	Mid      string `json:"mid"`      // Message ID being reacted to
	Action   string `json:"action"`   // "react" or "unreact"
	Reaction string `json:"reaction"` // Emoji (e.g., "love", "like", etc.)
	Emoji    string `json:"emoji"`    // Actual emoji character
}

// WebhookRead represents read receipt
type WebhookRead struct {
	Watermark int64 `json:"watermark"` // Timestamp of last read message
}

// WebhookDelivery represents delivery receipt
type WebhookDelivery struct {
	Mids      []string `json:"mids"`      // Message IDs that were delivered
	Watermark int64    `json:"watermark"` // Timestamp
}

// ParsedMessage represents a parsed incoming message
type ParsedMessage struct {
	SenderID       string
	RecipientID    string
	MessageID      string
	Timestamp      time.Time
	Type           string // text, image, video, audio, file, story_mention, story_reply
	Text           string
	MediaURL       string
	MediaType      string
	IsEcho         bool
	IsDeleted      bool
	IsUnsupported  bool
	ReplyToMid     string
	StoryURL       string
	StoryID        string
}

// ParsedReaction represents a parsed reaction event
type ParsedReaction struct {
	SenderID     string
	MessageID    string
	Action       string // react or unreact
	Reaction     string
	Emoji        string
	Timestamp    time.Time
}

// UserProfile represents an Instagram user profile
type UserProfile struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ProfilePic      string `json:"profile_pic"`
	Username        string `json:"username,omitempty"`
	FollowerCount   int    `json:"follower_count,omitempty"`
	IsVerifiedUser  bool   `json:"is_verified_user,omitempty"`
	IsUserFollowing bool   `json:"is_user_following,omitempty"`
}
