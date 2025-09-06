package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/heyrmi/goslack/db/sqlc"
)

// MessageService handles message-related business logic
type MessageService struct {
	store       db.Store
	userService *UserService
}

// NewMessageService creates a new message service
func NewMessageService(store db.Store, userService *UserService) *MessageService {
	return &MessageService{
		store:       store,
		userService: userService,
	}
}

// SendChannelMessage sends a message to a channel
func (s *MessageService) SendChannelMessage(ctx context.Context, workspaceID, channelID, senderID int64, content string) (*MessageResponse, error) {
	// Verify sender is a workspace member
	isMember, err := s.userService.IsWorkspaceMember(ctx, senderID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check workspace membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("sender is not a member of the workspace")
	}

	// Create the message
	arg := db.CreateChannelMessageParams{
		WorkspaceID: workspaceID,
		ChannelID:   sql.NullInt64{Int64: channelID, Valid: true},
		SenderID:    senderID,
		Content:     content,
	}

	message, err := s.store.CreateChannelMessage(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel message: %w", err)
	}

	return s.toMessageResponse(ctx, message)
}

// SendDirectMessage sends a direct message between two users
func (s *MessageService) SendDirectMessage(ctx context.Context, workspaceID, senderID, receiverID int64, content string) (*MessageResponse, error) {
	// Verify both users are workspace members
	isSenderMember, err := s.userService.IsWorkspaceMember(ctx, senderID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check sender workspace membership: %w", err)
	}
	if !isSenderMember {
		return nil, errors.New("sender is not a member of the workspace")
	}

	isReceiverMember, err := s.userService.IsWorkspaceMember(ctx, receiverID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check receiver workspace membership: %w", err)
	}
	if !isReceiverMember {
		return nil, errors.New("receiver is not a member of the workspace")
	}

	// Create the message
	arg := db.CreateDirectMessageParams{
		WorkspaceID: workspaceID,
		SenderID:    senderID,
		ReceiverID:  sql.NullInt64{Int64: receiverID, Valid: true},
		Content:     content,
	}

	message, err := s.store.CreateDirectMessage(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct message: %w", err)
	}

	return s.toMessageResponse(ctx, message)
}

// GetChannelMessages retrieves messages from a channel with pagination
func (s *MessageService) GetChannelMessages(ctx context.Context, workspaceID, channelID, userID int64, limit, offset int32) ([]*MessageResponse, error) {
	// Verify user is a workspace member
	isMember, err := s.userService.IsWorkspaceMember(ctx, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check workspace membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("user is not a member of the workspace")
	}

	// Get messages
	arg := db.GetChannelMessagesParams{
		ChannelID:   sql.NullInt64{Int64: channelID, Valid: true},
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	}

	messages, err := s.store.GetChannelMessages(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel messages: %w", err)
	}

	return s.toChannelMessageResponses(messages), nil
}

// GetDirectMessages retrieves direct messages between two users
func (s *MessageService) GetDirectMessages(ctx context.Context, workspaceID, userID, otherUserID int64, limit, offset int32) ([]*MessageResponse, error) {
	// Verify both users are workspace members
	isMember, err := s.userService.IsWorkspaceMember(ctx, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check user workspace membership: %w", err)
	}
	if !isMember {
		return nil, errors.New("user is not a member of the workspace")
	}

	isOtherMember, err := s.userService.IsWorkspaceMember(ctx, otherUserID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check other user workspace membership: %w", err)
	}
	if !isOtherMember {
		return nil, errors.New("other user is not a member of the workspace")
	}

	// Get messages
	arg := db.GetDirectMessagesBetweenUsersParams{
		WorkspaceID: workspaceID,
		SenderID:    userID,
		ReceiverID:  sql.NullInt64{Int64: otherUserID, Valid: true},
		Limit:       limit,
		Offset:      offset,
	}

	messages, err := s.store.GetDirectMessagesBetweenUsers(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to get direct messages: %w", err)
	}

	return s.toDirectMessageResponses(messages), nil
}

// EditMessage edits a message (only by the author)
func (s *MessageService) EditMessage(ctx context.Context, messageID, userID int64, newContent string) (*MessageResponse, error) {
	// Check if user is the author
	authorID, err := s.store.CheckMessageAuthor(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("message not found")
		}
		return nil, fmt.Errorf("failed to check message author: %w", err)
	}

	if authorID != userID {
		return nil, errors.New("only the message author can edit the message")
	}

	// Update the message
	arg := db.UpdateMessageContentParams{
		ID:      messageID,
		Content: newContent,
	}

	message, err := s.store.UpdateMessageContent(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	return s.toMessageResponse(ctx, message)
}

// DeleteMessage soft deletes a message (by author or workspace admin)
func (s *MessageService) DeleteMessage(ctx context.Context, messageID, userID int64) error {
	// Get the message to check author and workspace
	message, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("message not found")
		}
		return fmt.Errorf("failed to get message: %w", err)
	}

	// Check if user is the author or workspace admin
	isAuthor := message.SenderID == userID
	isAdmin := false

	if !isAuthor {
		// Check if user is workspace admin
		var workspaceAdminErr error
		isAdmin, workspaceAdminErr = s.userService.IsWorkspaceAdmin(ctx, userID, message.WorkspaceID)
		if workspaceAdminErr != nil {
			return fmt.Errorf("failed to check workspace admin status: %w", workspaceAdminErr)
		}
	}

	if !isAuthor && !isAdmin {
		return errors.New("only the message author or workspace admin can delete the message")
	}

	// Soft delete the message
	err = s.store.SoftDeleteMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// GetMessage retrieves a single message
func (s *MessageService) GetMessage(ctx context.Context, messageID, userID int64) (*MessageResponse, error) {
	message, err := s.store.GetMessageByID(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Check if user has access to this message
	if message.MessageType == "direct" {
		// For direct messages, user must be sender or receiver
		receiverID := int64(0)
		if message.ReceiverID.Valid {
			receiverID = message.ReceiverID.Int64
		}

		if message.SenderID != userID && receiverID != userID {
			return nil, errors.New("access denied: user is not part of this conversation")
		}
	} else {
		// For channel messages, user must be workspace member
		isMember, err := s.userService.IsWorkspaceMember(ctx, userID, message.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to check workspace membership: %w", err)
		}
		if !isMember {
			return nil, errors.New("access denied: user is not a member of the workspace")
		}
	}

	return s.toMessageByIDResponse(message), nil
}

// Helper function to convert db message to response with sender info
func (s *MessageService) toMessageResponse(ctx context.Context, message db.Message) (*MessageResponse, error) {
	sender, err := s.userService.GetUser(ctx, message.SenderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender info: %w", err)
	}

	response := &MessageResponse{
		ID:          message.ID,
		WorkspaceID: message.WorkspaceID,
		SenderID:    message.SenderID,
		Content:     message.Content,
		MessageType: message.MessageType,
		Sender:      sender,
		CreatedAt:   message.CreatedAt,
	}

	if message.ChannelID.Valid {
		response.ChannelID = &message.ChannelID.Int64
	}

	if message.ReceiverID.Valid {
		response.ReceiverID = &message.ReceiverID.Int64
	}

	if message.ThreadID.Valid {
		response.ThreadID = &message.ThreadID.Int64
	}

	if message.EditedAt.Valid {
		response.EditedAt = &message.EditedAt.Time
	}

	return response, nil
}

// Helper function to convert channel message rows to responses
func (s *MessageService) toChannelMessageResponses(messages []db.GetChannelMessagesRow) []*MessageResponse {
	responses := make([]*MessageResponse, len(messages))
	for i, message := range messages {
		response := &MessageResponse{
			ID:          message.ID,
			WorkspaceID: message.WorkspaceID,
			SenderID:    message.SenderID,
			Content:     message.Content,
			MessageType: message.MessageType,
			Sender: UserResponse{
				ID:        message.SenderID,
				Email:     message.SenderEmail,
				FirstName: message.SenderFirstName,
				LastName:  message.SenderLastName,
			},
			CreatedAt: message.CreatedAt,
		}

		if message.ChannelID.Valid {
			response.ChannelID = &message.ChannelID.Int64
		}

		if message.ReceiverID.Valid {
			response.ReceiverID = &message.ReceiverID.Int64
		}

		if message.ThreadID.Valid {
			response.ThreadID = &message.ThreadID.Int64
		}

		if message.EditedAt.Valid {
			response.EditedAt = &message.EditedAt.Time
		}

		responses[i] = response
	}
	return responses
}

// Helper function to convert direct message rows to responses
func (s *MessageService) toDirectMessageResponses(messages []db.GetDirectMessagesBetweenUsersRow) []*MessageResponse {
	responses := make([]*MessageResponse, len(messages))
	for i, message := range messages {
		response := &MessageResponse{
			ID:          message.ID,
			WorkspaceID: message.WorkspaceID,
			SenderID:    message.SenderID,
			Content:     message.Content,
			MessageType: message.MessageType,
			Sender: UserResponse{
				ID:        message.SenderID,
				Email:     message.SenderEmail,
				FirstName: message.SenderFirstName,
				LastName:  message.SenderLastName,
			},
			CreatedAt: message.CreatedAt,
		}

		if message.ChannelID.Valid {
			response.ChannelID = &message.ChannelID.Int64
		}

		if message.ReceiverID.Valid {
			response.ReceiverID = &message.ReceiverID.Int64
		}

		if message.ThreadID.Valid {
			response.ThreadID = &message.ThreadID.Int64
		}

		if message.EditedAt.Valid {
			response.EditedAt = &message.EditedAt.Time
		}

		responses[i] = response
	}
	return responses
}

// Helper function to convert message by ID row to response
func (s *MessageService) toMessageByIDResponse(message db.GetMessageByIDRow) *MessageResponse {
	response := &MessageResponse{
		ID:          message.ID,
		WorkspaceID: message.WorkspaceID,
		SenderID:    message.SenderID,
		Content:     message.Content,
		MessageType: message.MessageType,
		Sender: UserResponse{
			ID:        message.SenderID,
			Email:     message.SenderEmail,
			FirstName: message.SenderFirstName,
			LastName:  message.SenderLastName,
		},
		CreatedAt: message.CreatedAt,
	}

	if message.ChannelID.Valid {
		response.ChannelID = &message.ChannelID.Int64
	}

	if message.ReceiverID.Valid {
		response.ReceiverID = &message.ReceiverID.Int64
	}

	if message.ThreadID.Valid {
		response.ThreadID = &message.ThreadID.Int64
	}

	if message.EditedAt.Valid {
		response.EditedAt = &message.EditedAt.Time
	}

	return response
}
