package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// @Summary Create Thread Reply
// @Description Create a reply to a message thread
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.ThreadReplyRequest true "Thread reply details"
// @Success 200 {object} db.Message "Thread reply created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/thread/reply [post]
func (server *Server) createThreadReply(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.ThreadReplyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	message, err := server.messageEnhancedService.CreateThreadReply(ctx, req, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, message)
}

// @Summary Get Thread Messages
// @Description Get all messages in a thread
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param thread_id path int true "Thread ID"
// @Success 200 {array} db.GetThreadMessagesRow "Thread messages"
// @Failure 400 {object} map[string]string "Invalid thread ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/thread/{thread_id} [get]
func (server *Server) getThreadMessages(ctx *gin.Context) {
	threadIDStr := ctx.Param("thread_id")
	threadID, err := strconv.ParseInt(threadIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	messages, err := server.messageEnhancedService.GetThreadMessages(ctx, threadID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, messages)
}

// @Summary Get Thread Info
// @Description Get information about a thread
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param thread_id path int true "Thread ID"
// @Success 200 {object} db.GetThreadInfoRow "Thread information"
// @Failure 400 {object} map[string]string "Invalid thread ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/thread/{thread_id}/info [get]
func (server *Server) getThreadInfo(ctx *gin.Context) {
	threadIDStr := ctx.Param("thread_id")
	threadID, err := strconv.ParseInt(threadIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	threadInfo, err := server.messageEnhancedService.GetThreadInfo(ctx, threadID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, threadInfo)
}

// @Summary Add Message Reaction
// @Description Add a reaction to a message
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.MessageReactionRequest true "Reaction details"
// @Success 200 {object} map[string]string "Reaction added successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/reactions/add [post]
func (server *Server) addMessageReaction(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.MessageReactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.messageEnhancedService.AddReaction(ctx, req, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Reaction added successfully"})
}

// @Summary Remove Message Reaction
// @Description Remove a reaction from a message
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.MessageReactionRequest true "Reaction details"
// @Success 200 {object} map[string]string "Reaction removed successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/reactions/remove [post]
func (server *Server) removeMessageReaction(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.MessageReactionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.messageEnhancedService.RemoveReaction(ctx, req, user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Reaction removed successfully"})
}

// @Summary Get Message Reactions
// @Description Get all reactions for a message
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param message_id path int true "Message ID"
// @Success 200 {array} db.GetMessageReactionsRow "Message reactions"
// @Failure 400 {object} map[string]string "Invalid message ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/{message_id}/reactions [get]
func (server *Server) getMessageReactions(ctx *gin.Context) {
	messageIDStr := ctx.Param("message_id")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	reactions, err := server.messageEnhancedService.GetMessageReactions(ctx, messageID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, reactions)
}

// @Summary Search Messages
// @Description Search for messages in a workspace
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.SearchMessagesRequest true "Search parameters"
// @Success 200 {array} db.SearchMessagesRow "Search results"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/search [post]
func (server *Server) searchMessages(ctx *gin.Context) {
	var req service.SearchMessagesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	results, err := server.messageEnhancedService.SearchMessages(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, results)
}

// @Summary Pin Message
// @Description Pin a message in a channel
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.PinMessageRequest true "Pin message details"
// @Success 200 {object} map[string]string "Message pinned successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/pin [post]
func (server *Server) pinMessage(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.PinMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.PinnedBy = user.ID

	err := server.messageEnhancedService.PinMessage(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Message pinned successfully"})
}

// @Summary Unpin Message
// @Description Unpin a message
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param message_id path int true "Message ID"
// @Param channel_id query int true "Channel ID"
// @Success 200 {object} map[string]string "Message unpinned successfully"
// @Failure 400 {object} map[string]string "Invalid message ID or channel ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/{message_id}/unpin [post]
func (server *Server) unpinMessage(ctx *gin.Context) {
	messageIDStr := ctx.Param("message_id")
	messageID, err := strconv.ParseInt(messageIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	channelIDStr := ctx.Query("channel_id")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err = server.messageEnhancedService.UnpinMessage(ctx, messageID, channelID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Message unpinned successfully"})
}

// @Summary Get Pinned Messages
// @Description Get all pinned messages in a channel
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param id path int true "Channel ID"
// @Success 200 {array} db.GetPinnedMessagesRow "Pinned messages"
// @Failure 400 {object} map[string]string "Invalid channel ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /channels/{id}/pinned [get]
func (server *Server) getPinnedMessages(ctx *gin.Context) {
	channelIDStr := ctx.Param("id")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	pinnedMessages, err := server.messageEnhancedService.GetPinnedMessages(ctx, channelID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, pinnedMessages)
}

// @Summary Save Message Draft
// @Description Save a message draft
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.SaveDraftRequest true "Draft details"
// @Success 200 {object} db.MessageDraft "Draft saved successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/drafts [post]
func (server *Server) saveDraft(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.SaveDraftRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.UserID = user.ID

	draft, err := server.messageEnhancedService.SaveDraft(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, draft)
}

// @Summary Get User Drafts
// @Description Get all drafts for a user in a workspace
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Success 200 {array} db.GetUserDraftsRow "User drafts"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/drafts [get]
func (server *Server) getUserDrafts(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	drafts, err := server.messageEnhancedService.GetUserDrafts(ctx, user.ID, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, drafts)
}

// @Summary Get User Mentions
// @Description Get all mentions for a user
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param limit query int false "Number of mentions to return" default(20)
// @Param offset query int false "Number of mentions to skip" default(0)
// @Success 200 {array} db.GetUserMentionsRow "User mentions"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/mentions [get]
func (server *Server) getUserMentions(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	limit := int32(20)
	offset := int32(0)

	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	if offsetStr := ctx.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = int32(o)
		}
	}

	mentions, err := server.messageEnhancedService.GetUserMentions(ctx, user.ID, workspaceID, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, mentions)
}

// @Summary Get Unread Messages
// @Description Get unread message counts for a user
// @Tags messages
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Success 200 {array} db.UnreadMessage "Unread messages"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/unread [get]
func (server *Server) getUnreadMessages(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	unreadMessages, err := server.messageEnhancedService.GetUnreadMessages(ctx, user.ID, workspaceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, unreadMessages)
}

// @Summary Mark Channel as Read
// @Description Mark a channel as read for the current user
// @Tags messages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.MarkAsReadRequest true "Mark as read details"
// @Success 200 {object} map[string]string "Channel marked as read"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /messages/mark-read [post]
func (server *Server) markAsRead(ctx *gin.Context) {
	user := getCurrentUser(ctx)

	var req service.MarkAsReadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	req.UserID = user.ID

	err := server.messageEnhancedService.MarkChannelAsRead(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Marked as read successfully"})
}
