package service

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	db "github.com/heyrmi/goslack/db/sqlc"
)

// MessageEnhancedService handles advanced message features like threading, reactions, mentions, etc.
type MessageEnhancedService struct {
	store db.Store
	hub   MessageHub // Interface for WebSocket broadcasting
}

// MessageHub interface for WebSocket broadcasting
type MessageHub interface {
	BroadcastToChannel(workspaceID, channelID int64, message interface{})
	BroadcastToUser(userID int64, message interface{})
}

type ThreadReplyRequest struct {
	WorkspaceID int64  `json:"workspace_id" binding:"required"`
	ThreadID    int64  `json:"thread_id" binding:"required"`
	ChannelID   *int64 `json:"channel_id"`
	ReceiverID  *int64 `json:"receiver_id"`
	Content     string `json:"content" binding:"required"`
	ContentType string `json:"content_type"`
}

type MessageReactionRequest struct {
	MessageID int64  `json:"message_id" binding:"required"`
	Emoji     string `json:"emoji" binding:"required"`
}

type MessageMentionRequest struct {
	MessageID       int64  `json:"message_id" binding:"required"`
	MentionedUserID *int64 `json:"mentioned_user_id"`
	MentionType     string `json:"mention_type" binding:"required"`
}

type SearchMessagesRequest struct {
	WorkspaceID int64  `json:"workspace_id" binding:"required"`
	ChannelID   *int64 `json:"channel_id"`
	UserID      *int64 `json:"user_id"`
	Query       string `json:"query" binding:"required"`
	Limit       int32  `json:"limit"`
	Offset      int32  `json:"offset"`
}

type PinMessageRequest struct {
	MessageID int64 `json:"message_id" binding:"required"`
	ChannelID int64 `json:"channel_id" binding:"required"`
	PinnedBy  int64 `json:"pinned_by" binding:"required"`
}

type SaveDraftRequest struct {
	UserID      int64  `json:"user_id" binding:"required"`
	WorkspaceID int64  `json:"workspace_id" binding:"required"`
	ChannelID   *int64 `json:"channel_id"`
	ReceiverID  *int64 `json:"receiver_id"`
	ThreadID    *int64 `json:"thread_id"`
	Content     string `json:"content" binding:"required"`
}

type MarkAsReadRequest struct {
	UserID            int64  `json:"user_id" binding:"required"`
	WorkspaceID       int64  `json:"workspace_id" binding:"required"`
	ChannelID         *int64 `json:"channel_id"`
	LastReadMessageID int64  `json:"last_read_message_id" binding:"required"`
}

func NewMessageEnhancedService(store db.Store, hub MessageHub) *MessageEnhancedService {
	return &MessageEnhancedService{
		store: store,
		hub:   hub,
	}
}

// Thread Operations

func (s *MessageEnhancedService) CreateThreadReply(ctx context.Context, req ThreadReplyRequest, senderID int64) (*db.Message, error) {
	// Validate thread exists
	threadInfo, err := s.store.GetThreadInfo(ctx, req.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}

	// Determine message type
	messageType := "channel"
	if req.ReceiverID != nil {
		messageType = "direct"
	}

	// Set default content type
	contentType := req.ContentType
	if contentType == "" {
		contentType = "text"
	}

	// Create thread reply
	message, err := s.store.CreateThreadReply(ctx, db.CreateThreadReplyParams{
		WorkspaceID: req.WorkspaceID,
		ChannelID:   sql.NullInt64{Int64: *req.ChannelID, Valid: req.ChannelID != nil},
		SenderID:    senderID,
		ReceiverID:  sql.NullInt64{Int64: *req.ReceiverID, Valid: req.ReceiverID != nil},
		Content:     req.Content,
		ContentType: contentType,
		MessageType: messageType,
		ThreadID:    sql.NullInt64{Int64: req.ThreadID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create thread reply: %w", err)
	}

	// Parse and create mentions
	err = s.processMentions(ctx, message.ID, req.Content, req.WorkspaceID)
	if err != nil {
		// Log error but don't fail the message creation
		fmt.Printf("Failed to process mentions: %v\n", err)
	}

	// Broadcast to channel or user
	if req.ChannelID != nil {
		s.hub.BroadcastToChannel(req.WorkspaceID, *req.ChannelID, map[string]interface{}{
			"type":    "thread_reply",
			"message": message,
			"thread":  threadInfo,
		})
	} else if req.ReceiverID != nil {
		s.hub.BroadcastToUser(*req.ReceiverID, map[string]interface{}{
			"type":    "thread_reply",
			"message": message,
			"thread":  threadInfo,
		})
	}

	return &message, nil
}

func (s *MessageEnhancedService) GetThreadMessages(ctx context.Context, threadID int64) ([]db.GetThreadMessagesRow, error) {
	return s.store.GetThreadMessages(ctx, threadID)
}

func (s *MessageEnhancedService) GetThreadReplies(ctx context.Context, threadID int64, limit, offset int32) ([]db.GetThreadRepliesRow, error) {
	return s.store.GetThreadReplies(ctx, db.GetThreadRepliesParams{
		ThreadID: sql.NullInt64{Int64: threadID, Valid: true},
		Limit:    limit,
		Offset:   offset,
	})
}

func (s *MessageEnhancedService) GetThreadInfo(ctx context.Context, threadID int64) (*db.GetThreadInfoRow, error) {
	threadInfo, err := s.store.GetThreadInfo(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}
	return &threadInfo, nil
}

// Reaction Operations

func (s *MessageEnhancedService) AddReaction(ctx context.Context, req MessageReactionRequest, userID int64) error {
	_, err := s.store.AddMessageReaction(ctx, db.AddMessageReactionParams{
		MessageID: req.MessageID,
		UserID:    userID,
		Emoji:     req.Emoji,
	})
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	// Get message to determine broadcast target
	message, err := s.store.GetMessageByID(ctx, req.MessageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	// Broadcast reaction update
	reactionCounts, err := s.store.GetMessageReactionCounts(ctx, req.MessageID)
	if err == nil {
		broadcastData := map[string]interface{}{
			"type":       "reaction_added",
			"message_id": req.MessageID,
			"emoji":      req.Emoji,
			"user_id":    userID,
			"counts":     reactionCounts,
		}

		if message.ChannelID.Valid {
			s.hub.BroadcastToChannel(message.WorkspaceID, message.ChannelID.Int64, broadcastData)
		} else if message.ReceiverID.Valid {
			s.hub.BroadcastToUser(message.ReceiverID.Int64, broadcastData)
			s.hub.BroadcastToUser(message.SenderID, broadcastData)
		}
	}

	return nil
}

func (s *MessageEnhancedService) RemoveReaction(ctx context.Context, req MessageReactionRequest, userID int64) error {
	err := s.store.RemoveMessageReaction(ctx, db.RemoveMessageReactionParams{
		MessageID: req.MessageID,
		UserID:    userID,
		Emoji:     req.Emoji,
	})
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}

	// Get message to determine broadcast target
	message, err := s.store.GetMessageByID(ctx, req.MessageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	// Broadcast reaction update
	reactionCounts, err := s.store.GetMessageReactionCounts(ctx, req.MessageID)
	if err == nil {
		broadcastData := map[string]interface{}{
			"type":       "reaction_removed",
			"message_id": req.MessageID,
			"emoji":      req.Emoji,
			"user_id":    userID,
			"counts":     reactionCounts,
		}

		if message.ChannelID.Valid {
			s.hub.BroadcastToChannel(message.WorkspaceID, message.ChannelID.Int64, broadcastData)
		} else if message.ReceiverID.Valid {
			s.hub.BroadcastToUser(message.ReceiverID.Int64, broadcastData)
			s.hub.BroadcastToUser(message.SenderID, broadcastData)
		}
	}

	return nil
}

func (s *MessageEnhancedService) GetMessageReactions(ctx context.Context, messageID int64) ([]db.GetMessageReactionsRow, error) {
	return s.store.GetMessageReactions(ctx, messageID)
}

func (s *MessageEnhancedService) GetMessageReactionCounts(ctx context.Context, messageID int64) ([]db.GetMessageReactionCountsRow, error) {
	return s.store.GetMessageReactionCounts(ctx, messageID)
}

// Mention Operations

func (s *MessageEnhancedService) processMentions(ctx context.Context, messageID int64, content string, workspaceID int64) error {
	// Regular expressions for different mention types
	userMentionRegex := regexp.MustCompile(`@(\w+)`)
	channelMentionRegex := regexp.MustCompile(`@channel`)
	hereMentionRegex := regexp.MustCompile(`@here`)
	everyoneMentionRegex := regexp.MustCompile(`@everyone`)

	// Find @channel mentions
	if channelMentionRegex.MatchString(content) {
		_, err := s.store.CreateMessageMention(ctx, db.CreateMessageMentionParams{
			MessageID:       messageID,
			MentionedUserID: sql.NullInt64{Valid: false},
			MentionType:     "channel",
		})
		if err != nil {
			return fmt.Errorf("failed to create channel mention: %w", err)
		}
	}

	// Find @here mentions
	if hereMentionRegex.MatchString(content) {
		_, err := s.store.CreateMessageMention(ctx, db.CreateMessageMentionParams{
			MessageID:       messageID,
			MentionedUserID: sql.NullInt64{Valid: false},
			MentionType:     "here",
		})
		if err != nil {
			return fmt.Errorf("failed to create here mention: %w", err)
		}
	}

	// Find @everyone mentions
	if everyoneMentionRegex.MatchString(content) {
		_, err := s.store.CreateMessageMention(ctx, db.CreateMessageMentionParams{
			MessageID:       messageID,
			MentionedUserID: sql.NullInt64{Valid: false},
			MentionType:     "everyone",
		})
		if err != nil {
			return fmt.Errorf("failed to create everyone mention: %w", err)
		}
	}

	// Find user mentions (@username)
	userMatches := userMentionRegex.FindAllStringSubmatch(content, -1)
	for _, match := range userMatches {
		if len(match) > 1 {
			username := match[1]
			// For simplicity, we'll assume username is the email prefix
			// In a real implementation, you'd have a proper username system
			email := username + "@example.com"

			user, err := s.store.GetUserByEmail(ctx, email)
			if err == nil {
				_, err = s.store.CreateMessageMention(ctx, db.CreateMessageMentionParams{
					MessageID:       messageID,
					MentionedUserID: sql.NullInt64{Int64: user.ID, Valid: true},
					MentionType:     "user",
				})
				if err != nil {
					return fmt.Errorf("failed to create user mention: %w", err)
				}
			}
		}
	}

	return nil
}

func (s *MessageEnhancedService) GetUserMentions(ctx context.Context, userID, workspaceID int64, limit, offset int32) ([]db.GetUserMentionsRow, error) {
	return s.store.GetUserMentions(ctx, db.GetUserMentionsParams{
		MentionedUserID: sql.NullInt64{Int64: userID, Valid: true},
		WorkspaceID:     workspaceID,
		Limit:           limit,
		Offset:          offset,
	})
}

func (s *MessageEnhancedService) GetUnreadMentions(ctx context.Context, userID, workspaceID int64, limit int32) ([]db.GetUnreadMentionsRow, error) {
	return s.store.GetUnreadMentions(ctx, db.GetUnreadMentionsParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Limit:       limit,
	})
}

// Search Operations

func (s *MessageEnhancedService) SearchMessages(ctx context.Context, req SearchMessagesRequest) ([]db.SearchMessagesRow, error) {
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	var channelID, userID int64
	if req.ChannelID != nil {
		channelID = *req.ChannelID
	}
	if req.UserID != nil {
		userID = *req.UserID
	}

	return s.store.SearchMessages(ctx, db.SearchMessagesParams{
		WorkspaceID:    req.WorkspaceID,
		Column2:        channelID,
		Column3:        userID,
		PlaintoTsquery: req.Query,
		Limit:          limit,
		Offset:         req.Offset,
	})
}

func (s *MessageEnhancedService) SearchMessagesInThread(ctx context.Context, threadID, workspaceID int64, query string, limit, offset int32) ([]db.SearchMessagesInThreadRow, error) {
	if limit == 0 {
		limit = 10
	}

	return s.store.SearchMessagesInThread(ctx, db.SearchMessagesInThreadParams{
		ID:             threadID,
		WorkspaceID:    workspaceID,
		PlaintoTsquery: query,
		Limit:          limit,
		Offset:         offset,
	})
}

// Pinning Operations

func (s *MessageEnhancedService) PinMessage(ctx context.Context, req PinMessageRequest) error {
	_, err := s.store.PinMessage(ctx, db.PinMessageParams{
		MessageID: req.MessageID,
		ChannelID: req.ChannelID,
		PinnedBy:  req.PinnedBy,
	})
	if err != nil {
		return fmt.Errorf("failed to pin message: %w", err)
	}

	// Get message to determine workspace
	message, err := s.store.GetMessageByID(ctx, req.MessageID)
	if err == nil {
		// Broadcast pin update
		s.hub.BroadcastToChannel(message.WorkspaceID, req.ChannelID, map[string]interface{}{
			"type":       "message_pinned",
			"message_id": req.MessageID,
			"pinned_by":  req.PinnedBy,
		})
	}

	return nil
}

func (s *MessageEnhancedService) UnpinMessage(ctx context.Context, messageID, channelID int64) error {
	// Get message to determine workspace
	message, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get message: %w", err)
	}

	err = s.store.UnpinMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to unpin message: %w", err)
	}

	// Broadcast unpin update
	s.hub.BroadcastToChannel(message.WorkspaceID, channelID, map[string]interface{}{
		"type":       "message_unpinned",
		"message_id": messageID,
	})

	return nil
}

func (s *MessageEnhancedService) GetPinnedMessages(ctx context.Context, channelID int64) ([]db.GetPinnedMessagesRow, error) {
	return s.store.GetPinnedMessages(ctx, channelID)
}

// Draft Operations

func (s *MessageEnhancedService) SaveDraft(ctx context.Context, req SaveDraftRequest) (*db.MessageDraft, error) {
	draft, err := s.store.SaveMessageDraft(ctx, db.SaveMessageDraftParams{
		UserID:      req.UserID,
		WorkspaceID: req.WorkspaceID,
		ChannelID:   sql.NullInt64{Int64: *req.ChannelID, Valid: req.ChannelID != nil},
		ReceiverID:  sql.NullInt64{Int64: *req.ReceiverID, Valid: req.ReceiverID != nil},
		ThreadID:    sql.NullInt64{Int64: *req.ThreadID, Valid: req.ThreadID != nil},
		Content:     req.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save draft: %w", err)
	}

	return &draft, nil
}

func (s *MessageEnhancedService) GetDraft(ctx context.Context, userID int64, channelID, receiverID, threadID *int64) (*db.MessageDraft, error) {
	draft, err := s.store.GetMessageDraft(ctx, db.GetMessageDraftParams{
		UserID:     userID,
		ChannelID:  sql.NullInt64{Int64: *channelID, Valid: channelID != nil},
		ReceiverID: sql.NullInt64{Int64: *receiverID, Valid: receiverID != nil},
		ThreadID:   sql.NullInt64{Int64: *threadID, Valid: threadID != nil},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	return &draft, nil
}

func (s *MessageEnhancedService) DeleteDraft(ctx context.Context, userID int64, channelID, receiverID, threadID *int64) error {
	return s.store.DeleteMessageDraft(ctx, db.DeleteMessageDraftParams{
		UserID:     userID,
		ChannelID:  sql.NullInt64{Int64: *channelID, Valid: channelID != nil},
		ReceiverID: sql.NullInt64{Int64: *receiverID, Valid: receiverID != nil},
		ThreadID:   sql.NullInt64{Int64: *threadID, Valid: threadID != nil},
	})
}

func (s *MessageEnhancedService) GetUserDrafts(ctx context.Context, userID, workspaceID int64) ([]db.GetUserDraftsRow, error) {
	return s.store.GetUserDrafts(ctx, db.GetUserDraftsParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
}

// Unread Operations

func (s *MessageEnhancedService) GetUnreadMessages(ctx context.Context, userID, workspaceID int64) ([]db.UnreadMessage, error) {
	return s.store.GetUnreadMessages(ctx, db.GetUnreadMessagesParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
}

func (s *MessageEnhancedService) MarkChannelAsRead(ctx context.Context, req MarkAsReadRequest) error {
	return s.store.MarkChannelAsRead(ctx, db.MarkChannelAsReadParams{
		UserID:            req.UserID,
		WorkspaceID:       req.WorkspaceID,
		ChannelID:         sql.NullInt64{Int64: *req.ChannelID, Valid: req.ChannelID != nil},
		LastReadMessageID: sql.NullInt64{Int64: req.LastReadMessageID, Valid: true},
	})
}

func (s *MessageEnhancedService) GetChannelUnreadCount(ctx context.Context, userID, channelID int64) (int64, error) {
	result, err := s.store.GetChannelUnreadCount(ctx, db.GetChannelUnreadCountParams{
		UserID:    userID,
		ChannelID: sql.NullInt64{Int64: channelID, Valid: true},
	})
	if err != nil {
		return 0, err
	}
	return int64(result), nil
}

func (s *MessageEnhancedService) GetWorkspaceUnreadCount(ctx context.Context, userID, workspaceID int64) (int64, error) {
	result, err := s.store.GetWorkspaceUnreadCount(ctx, db.GetWorkspaceUnreadCountParams{
		UserID:      userID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return 0, err
	}
	// Convert interface{} to int64
	if totalUnread, ok := result.(int64); ok {
		return totalUnread, nil
	}
	return 0, nil
}

// Cleanup Operations

func (s *MessageEnhancedService) CleanupOldDrafts(ctx context.Context, olderThan time.Time) error {
	return s.store.CleanupOldDrafts(ctx, olderThan)
}
