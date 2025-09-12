package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// @Summary Send Channel Message
// @Description Send a message to a specific channel (requires workspace membership)
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param channel_id path int true "Channel ID"
// @Param message body service.SendChannelMessageRequest true "Message content"
// @Success 201 {object} service.MessageResponse "Message sent successfully"
// @Failure 400 {object} map[string]string "Invalid request or IDs"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/channels/{channel_id}/messages [post]
func (server *Server) sendChannelMessage(ctx *gin.Context) {
	var req service.SendChannelMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get channel ID from URL
	channelIDStr := ctx.Param("channel_id")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid channel ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Send message
	message, err := server.messageService.SendChannelMessage(ctx, workspaceID, channelID, currentUser.ID, req.Content)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, message)
}

// @Summary Send Direct Message
// @Description Send a direct message to another user (requires workspace membership)
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Workspace ID"
// @Param message body service.SendDirectMessageRequest true "Direct message content"
// @Success 201 {object} service.MessageResponse "Direct message sent successfully"
// @Failure 400 {object} map[string]string "Invalid request or workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/messages/direct [post]
func (server *Server) sendDirectMessage(ctx *gin.Context) {
	var req service.SendDirectMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Send message
	message, err := server.messageService.SendDirectMessage(ctx, workspaceID, currentUser.ID, req.ReceiverID, req.Content)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, message)
}

// @Summary Get Channel Messages
// @Description Retrieve messages from a specific channel (requires workspace membership)
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param channel_id path int true "Channel ID"
// @Param limit query int false "Number of messages to retrieve (default: 50, max: 100)" minimum(1) maximum(100)
// @Param offset query int false "Number of messages to skip (default: 0)" minimum(0)
// @Success 200 {object} map[string]interface{} "Channel messages"
// @Failure 400 {object} map[string]string "Invalid request or IDs"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/channels/{channel_id}/messages [get]
func (server *Server) getChannelMessages(ctx *gin.Context) {
	var req service.GetMessagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Set default values if not provided
	if req.Limit == 0 {
		req.Limit = 50 // Default to 50 messages
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get channel ID from URL
	channelIDStr := ctx.Param("channel_id")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid channel ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Get messages
	messages, err := server.messageService.GetChannelMessages(ctx, workspaceID, channelID, currentUser.ID, req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"messages": messages})
}

// @Summary Get Direct Messages
// @Description Retrieve direct messages with another user (requires workspace membership)
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param user_id path int true "Other User ID"
// @Param limit query int false "Number of messages to retrieve (default: 50, max: 100)" minimum(1) maximum(100)
// @Param offset query int false "Number of messages to skip (default: 0)" minimum(0)
// @Success 200 {object} map[string]interface{} "Direct messages"
// @Failure 400 {object} map[string]string "Invalid request or IDs"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspace/{id}/messages/direct/{user_id} [get]
func (server *Server) getDirectMessages(ctx *gin.Context) {
	var req service.GetMessagesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Set default values if not provided
	if req.Limit == 0 {
		req.Limit = 50 // Default to 50 messages
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid workspace ID")))
		return
	}

	// Get other user ID from URL
	otherUserIDStr := ctx.Param("user_id")
	otherUserID, err := strconv.ParseInt(otherUserIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid user ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Get messages
	messages, err := server.messageService.GetDirectMessages(ctx, workspaceID, currentUser.ID, otherUserID, req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"messages": messages})
}

// @Summary Edit Message
// @Description Edit a message (only message sender can edit)
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param message_id path int true "Message ID"
// @Param message body service.EditMessageRequest true "Updated message content"
// @Success 200 {object} service.MessageResponse "Message edited successfully"
// @Failure 400 {object} map[string]string "Invalid request or message ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Only message sender can edit"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/{message_id} [put]
func (server *Server) editMessage(ctx *gin.Context) {
	var req service.EditMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get message ID from URL
	messageIDStr := ctx.Param("message_id")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid message ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Edit message
	message, err := server.messageService.EditMessage(ctx, messageID, currentUser.ID, req.Content)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, message)
}

// @Summary Delete Message
// @Description Delete a message (only message sender can delete)
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param message_id path int true "Message ID"
// @Success 200 {object} map[string]string "Message deleted successfully"
// @Failure 400 {object} map[string]string "Invalid message ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Only message sender can delete"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/{message_id} [delete]
func (server *Server) deleteMessage(ctx *gin.Context) {
	// Get message ID from URL
	messageIDStr := ctx.Param("message_id")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid message ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Delete message
	err = server.messageService.DeleteMessage(ctx, messageID, currentUser.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}

// @Summary Get Message
// @Description Retrieve a specific message by ID
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param message_id path int true "Message ID"
// @Success 200 {object} service.MessageResponse "Message details"
// @Failure 400 {object} map[string]string "Invalid message ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/{message_id} [get]
func (server *Server) getMessage(ctx *gin.Context) {
	// Get message ID from URL
	messageIDStr := ctx.Param("message_id")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid message ID")))
		return
	}

	// Get current user
	currentUser := getCurrentUser(ctx)

	// Get message
	message, err := server.messageService.GetMessage(ctx, messageID, currentUser.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, message)
}
