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
	_, err = u.userRepo.GetByID(ctx, recvID)
	if err != nil {
		return nil, fmt.Errorf("receiver validation failed: %w", err)
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
	return u.chatRepo.GetChatHistory(ctx, user1, user2)
}

func (u *chatUsecase) GetRecentChats(ctx context.Context, userID string) ([]*domain.ChatMessage, error) {
	return u.chatRepo.GetRecentChats(ctx, userID)
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
