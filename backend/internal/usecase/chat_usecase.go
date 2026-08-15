package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gigpurse/internal/domain"
)

type chatUsecase struct {
	chatRepo  domain.ChatRepository
	userRepo  domain.UserRepository
	notifRepo domain.NotificationRepository
}

func NewChatUsecase(chatRepo domain.ChatRepository, userRepo domain.UserRepository, notifRepo domain.NotificationRepository) domain.ChatUsecase {
	return &chatUsecase{
		chatRepo:  chatRepo,
		userRepo:  userRepo,
		notifRepo: notifRepo,
	}
}

func (u *chatUsecase) SendMessage(ctx context.Context, senderID, recvID, content string) (*domain.ChatMessage, error) {
	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}
	if senderID == recvID {
		return nil, errors.New("cannot send a message to yourself")
	}

	// Validate sender exists
	sender, err := u.userRepo.GetByID(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("sender validation failed: %w", err)
	}

	// Validate receiver exists
	receiver, err := u.userRepo.GetByID(ctx, recvID)
	if err != nil {
		return nil, fmt.Errorf("receiver validation failed: %w", err)
	}
	if receiver.Disabled {
		return nil, errors.New("this account is currently disabled and won't receive your message")
	}

	filteredContent := filterContent(content)

	msg := &domain.ChatMessage{
		SenderID:  senderID,
		RecvID:    recvID,
		Content:   filteredContent,
		Timestamp: time.Now(),
	}

	if err := u.chatRepo.SaveMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	preview := filteredContent
	if len(preview) > 80 {
		preview = preview[:80] + "…"
	}
	notif := &domain.Notification{
		UserID:    recvID,
		Title:     "New message from " + sender.Name,
		Message:   preview,
		Link:      "/messages?with=" + senderID,
		CreatedAt: time.Now(),
	}
	_ = u.notifRepo.Create(ctx, notif)

	return msg, nil
}

func (u *chatUsecase) GetChatHistory(ctx context.Context, user1, user2 string) ([]*domain.ChatMessage, error) {
	// user1 is the requester loading this thread — mark whatever user2 sent
	// them as read now that they're actually looking at it. Fire-and-forget,
	// same convention as every other notify/audit side effect in this
	// codebase; a failed mark-as-read shouldn't block showing the messages.
	_ = u.chatRepo.MarkConversationRead(ctx, user1, user2)
	return u.chatRepo.GetChatHistory(ctx, user1, user2)
}

func (u *chatUsecase) GetRecentChats(ctx context.Context, userID string) ([]*domain.ChatMessage, error) {
	return u.chatRepo.GetRecentChats(ctx, userID)
}

// StartUnreadEmailScanner mirrors milestoneUsecase.StartReminderScanner's
// shape exactly — a ticker firing sendUnreadDigests, stopped when ctx is
// cancelled.
func (u *chatUsecase) StartUnreadEmailScanner(ctx context.Context, checkInterval, staleAfter time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.sendUnreadDigests(ctx, staleAfter)
			}
		}
	}()
}

// sendUnreadDigests emails anyone with a message that's sat unread longer
// than staleAfter — one email per recipient covering every sender they
// haven't replied to, not one email per message, so five unread messages
// from the same person don't produce five emails.
func (u *chatUsecase) sendUnreadDigests(ctx context.Context, staleAfter time.Duration) {
	stale, err := u.chatRepo.ListUnreadOlderThan(ctx, time.Now().Add(-staleAfter))
	if err != nil || len(stale) == 0 {
		return
	}

	byRecipient := make(map[string][]*domain.ChatMessage)
	for _, msg := range stale {
		byRecipient[msg.RecvID] = append(byRecipient[msg.RecvID], msg)
	}

	for recvID, msgs := range byRecipient {
		recipient, err := u.userRepo.GetByID(ctx, recvID)
		if err != nil || recipient.Email == "" {
			continue
		}

		senderNames := make(map[string]bool)
		var ids []string
		for _, msg := range msgs {
			ids = append(ids, msg.ID)
			if sender, err := u.userRepo.GetByID(ctx, msg.SenderID); err == nil && sender.Name != "" {
				senderNames[sender.Name] = true
			}
		}
		names := make([]string, 0, len(senderNames))
		for name := range senderNames {
			names = append(names, name)
		}

		subject := "You have unread messages on GigPurse"
		body := fmt.Sprintf("You have unread messages from %s. Log in to GigPurse to read them.", strings.Join(names, ", "))
		if err := sendEmailFn(recipient.Email, subject, body); err != nil {
			continue
		}
		_ = u.chatRepo.MarkReminderEmailSent(ctx, ids)
	}
}

// Simple Words Filtering System (profanity + bypass prevention)
func filterContent(input string) string {
	bypassWords := []string{
		"paypal", "cashapp", "venmo", "zelle", "whatsapp", "telegram", "e-mail",
		"phone number", "direct deposit", "pay me directly", "pay outside",
	}

	return applyFilters(input, bypassWords)
}

func applyFilters(input string, words []string) string {
	output := input
	for _, word := range words {
		lowerInput := strings.ToLower(output)
		lowerWord := strings.ToLower(word)
		startIdx := 0
		for {
			idx := strings.Index(lowerInput[startIdx:], lowerWord)
			if idx == -1 {
				break
			}
			absIdx := startIdx + idx
			censor := strings.Repeat("*", len(word))
			output = output[:absIdx] + censor + output[absIdx+len(word):]
			lowerInput = lowerInput[:absIdx] + censor + lowerInput[absIdx+len(word):]
			startIdx = absIdx + len(word)
		}
	}
	return output
}
