package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// @Summary Send Typing Indicator
// @Description Send typing indicator to a channel (requires workspace membership)
// @Tags realtime
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param channel_id path int true "Channel ID"
// @Success 200 {object} map[string]string "Typing indicator sent"
// @Failure 400 {object} map[string]string "Invalid workspace ID or channel ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Router /workspaces/{id}/channels/{channel_id}/typing [post]
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
