package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// handleTyping handles POST /workspaces/:id/channels/:channel_id/typing
func (server *Server) handleTyping(ctx *gin.Context) {
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

	// Broadcast typing indicator
	wsMessage := &service.WSMessage{
		Type: "user_typing",
		Data: map[string]interface{}{
			"user_id": currentUser.ID,
			"user":    currentUser,
			"typing":  true,
		},
		WorkspaceID: workspaceID,
		ChannelID:   &channelID,
		UserID:      currentUser.ID,
		Timestamp:   time.Now(),
	}

	server.hub.BroadcastToChannel(workspaceID, channelID, wsMessage)

	ctx.JSON(http.StatusOK, gin.H{"message": "Typing indicator sent"})
}
