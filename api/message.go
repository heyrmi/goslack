package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// sendChannelMessage handles POST /workspace/:id/channels/:channel_id/messages
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

// sendDirectMessage handles POST /workspace/:id/messages/direct
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

// getChannelMessages handles GET /workspace/:id/channels/:channel_id/messages
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

// getDirectMessages handles GET /workspace/:id/messages/direct/:user_id
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

// editMessage handles PUT /messages/:message_id
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

// deleteMessage handles DELETE /messages/:message_id
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

// getMessage handles GET /messages/:message_id
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

// Helper function to get current user from context
func getCurrentUser(ctx *gin.Context) service.UserResponse {
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		panic("user not found in context")
	}
	return currentUser.(service.UserResponse)
}
