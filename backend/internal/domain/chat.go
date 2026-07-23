package domain

import (
	"context"
	"time"
)

type ChatMessage struct {
	ID        string    `json:"id" bson:"_id"`
	SenderID  string    `json:"sender_id" bson:"sender_id"`
	RecvID    string    `json:"recv_id,omitempty" bson:"recv_id,omitempty"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`

	// DisputeID marks this as belonging to a dispute's 3-party (client +
	// musician + moderator) chat room instead of a normal 1:1 conversation —
	// when set, RecvID is unused and the message fans out to every
	// participant on that dispute instead of a single recipient.
	DisputeID string `json:"dispute_id,omitempty" bson:"dispute_id,omitempty"`
	// IsSystem marks an automatic message (e.g. "a moderator joined this
	// dispute") rather than something a participant typed.
	IsSystem bool `json:"is_system,omitempty" bson:"is_system,omitempty"`
	// MentionedUserID is set when the moderator taps "Tag" on a participant
	// (e.g. to ask them for proof) — that user gets a push notification.
	// Deliberately not free-text @mention parsing: a dispute room only ever
	// has two non-moderator participants, so a single explicit target is
	// simpler and unambiguous.
	MentionedUserID string `json:"mentioned_user_id,omitempty" bson:"mentioned_user_id,omitempty"`

	// AttachmentURL/AttachmentType carry an image or voice note — uploaded
	// first via POST /media/upload (already fast: local disk, no processing),
	// then sent as a normal message carrying just the resulting URL, so
	// delivery over the realtime socket is exactly as fast as a text message.
	AttachmentURL  string `json:"attachment_url,omitempty" bson:"attachment_url,omitempty"`
	AttachmentType string `json:"attachment_type,omitempty" bson:"attachment_type,omitempty"` // "image" or "audio"

	// ContractID/MilestoneID mark this as a milestone system message — when
	// both are set, the frontend renders an inline actionable milestone card
	// (accept/reject/counter/fund/release) right in the chat bubble instead
	// of plain text, so responding to a proposal doesn't require leaving
	// the thread to find the milestones panel.
	ContractID  string `json:"contract_id,omitempty" bson:"contract_id,omitempty"`
	MilestoneID string `json:"milestone_id,omitempty" bson:"milestone_id,omitempty"`
}

type ChatRepository interface {
	SaveMessage(ctx context.Context, msg *ChatMessage) error
	GetChatHistory(ctx context.Context, user1, user2 string) ([]*ChatMessage, error)
	GetRecentChats(ctx context.Context, userID string) ([]*ChatMessage, error)
	ListByDispute(ctx context.Context, disputeID string) ([]*ChatMessage, error)
}

type ChatUsecase interface {
	SendMessage(ctx context.Context, senderID, recvID, content string) (*ChatMessage, error)
	GetChatHistory(ctx context.Context, user1, user2 string) ([]*ChatMessage, error)
	GetRecentChats(ctx context.Context, userID string) ([]*ChatMessage, error)
}
