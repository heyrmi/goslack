package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/service"
)

// @Summary Upload File
// @Description Upload a file to a workspace, channel, or for direct messaging
// @Tags files
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param workspace_id formData int true "Workspace ID"
// @Param channel_id formData int false "Channel ID (for channel files)"
// @Param receiver_id formData int false "Receiver User ID (for direct message files)"
// @Success 201 {object} map[string]interface{} "File uploaded successfully"
// @Failure 400 {object} map[string]string "Invalid request, file too large, or validation error"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Access denied - workspace/channel membership required"
// @Failure 413 {object} map[string]string "File too large"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /files/upload [post]
func (server *Server) uploadFile(ctx *gin.Context) {
	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Parse multipart form
	if err := ctx.Request.ParseMultipartForm(server.config.FileMaxSize); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("failed to parse multipart form: %w", err)))
		return
	}

	// Parse form data
	var req service.FileUploadRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Validate that user belongs to the workspace
	if !server.userService.UserBelongsToWorkspace(user.ID, req.WorkspaceID) {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("access denied: user does not belong to workspace")))
		return
	}

	// If channel_id is provided, validate user has access to the channel
	if req.ChannelID != nil {
		if !server.channelService.UserHasChannelAccess(user.ID, *req.ChannelID) {
			ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("access denied: user does not have access to channel")))
			return
		}
	}

	// If receiver_id is provided, validate it's a direct message scenario
	if req.ReceiverID != nil {
		if req.ChannelID != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("cannot specify both channel_id and receiver_id")))
			return
		}

		// Validate receiver belongs to the same workspace
		if !server.userService.UserBelongsToWorkspace(*req.ReceiverID, req.WorkspaceID) {
			ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("receiver does not belong to the workspace")))
			return
		}
	}

	// Upload file
	fileResponse, err := server.fileService.UploadFile(req, user.ID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "File uploaded successfully",
		"file":    fileResponse,
	})
}

// @Summary Download File
// @Description Download a file by ID (requires appropriate access permissions)
// @Tags files
// @Security BearerAuth
// @Produce application/octet-stream
// @Param id path int true "File ID"
// @Success 200 {file} file "File content"
// @Failure 400 {object} map[string]string "Invalid file ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 404 {object} map[string]string "File not found or access denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /files/{id}/download [get]
func (server *Server) downloadFile(ctx *gin.Context) {
	// Get file ID from URL
	fileIDStr := ctx.Param("id")
	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid file ID")))
		return
	}

	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Get file content with permission check
	fileContent, fileInfo, err := server.fileService.GetFileContent(fileID, user.ID)
	if err != nil {
		if err.Error() == "file not found" || err.Error() == "access denied: you don't have permission to download this file" {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}
	defer fileContent.Close()

	// Set appropriate headers
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Type", fileInfo.MimeType)
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.OriginalFilename))
	ctx.Header("Content-Length", fmt.Sprintf("%d", fileInfo.FileSize))
	ctx.Header("Cache-Control", "must-revalidate")

	// Stream file content
	if _, err := io.Copy(ctx.Writer, fileContent); err != nil {
		// Log error but can't change response at this point
		fmt.Printf("Error streaming file: %v\n", err)
	}
}

// @Summary Get File Metadata
// @Description Retrieve file metadata by ID (requires appropriate access permissions)
// @Tags files
// @Security BearerAuth
// @Produce json
// @Param id path int true "File ID"
// @Param workspace_id query int true "Workspace ID"
// @Success 200 {object} map[string]interface{} "File metadata"
// @Failure 400 {object} map[string]string "Invalid file ID or workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Access denied - workspace membership required"
// @Failure 404 {object} map[string]string "File not found or access denied"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /files/{id} [get]
func (server *Server) getFile(ctx *gin.Context) {
	// Get file ID from URL
	fileIDStr := ctx.Param("id")
	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid file ID")))
		return
	}

	// Get workspace ID from query parameter
	workspaceIDStr := ctx.Query("workspace_id")
	if workspaceIDStr == "" {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("workspace_id is required")))
		return
	}

	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid workspace_id")))
		return
	}

	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Validate that user belongs to the workspace
	if !server.userService.UserBelongsToWorkspace(user.ID, workspaceID) {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("access denied: user does not belong to workspace")))
		return
	}

	// Get file with permission check
	fileResponse, err := server.fileService.GetFile(fileID, user.ID, workspaceID)
	if err != nil {
		if err.Error() == "file not found" || err.Error() == "access denied: you don't have permission to access this file" {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"file": fileResponse,
	})
}

// @Summary List Workspace Files
// @Description List all files in a workspace (requires workspace membership)
// @Tags files
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param limit query int false "Files per page (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} map[string]interface{} "List of files with pagination"
// @Failure 400 {object} map[string]string "Invalid workspace ID or pagination parameters"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/files [get]
func (server *Server) listWorkspaceFiles(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid workspace ID")))
		return
	}

	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Validate that user belongs to the workspace
	if !server.userService.UserBelongsToWorkspace(user.ID, workspaceID) {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("access denied: user does not belong to workspace")))
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// List files
	files, err := server.fileService.ListWorkspaceFiles(workspaceID, int32(limit), int32(offset))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"files": files,
		"pagination": gin.H{
			"page":   page,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// @Summary Delete File
// @Description Delete a file (only file uploader can delete)
// @Tags files
// @Security BearerAuth
// @Produce json
// @Param id path int true "File ID"
// @Success 200 {object} map[string]string "File deleted successfully"
// @Failure 400 {object} map[string]string "Invalid file ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Only file uploader can delete"
// @Failure 404 {object} map[string]string "File not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /files/{id} [delete]
func (server *Server) deleteFile(ctx *gin.Context) {
	// Get file ID from URL
	fileIDStr := ctx.Param("id")
	fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid file ID")))
		return
	}

	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Delete file
	if err := server.fileService.DeleteFile(fileID, user.ID); err != nil {
		if err.Error() == "file not found" {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
		} else if err.Error() == "access denied: only the file uploader can delete this file" {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
		} else {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "File deleted successfully",
	})
}

// @Summary Get File Statistics
// @Description Get file statistics for a workspace (requires workspace membership)
// @Tags files
// @Security BearerAuth
// @Produce json
// @Param id path int true "Workspace ID"
// @Success 200 {object} map[string]interface{} "File statistics"
// @Failure 400 {object} map[string]string "Invalid workspace ID"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /workspaces/{id}/files/stats [get]
func (server *Server) getFileStats(ctx *gin.Context) {
	// Get workspace ID from URL
	workspaceIDStr := ctx.Param("id")
	workspaceID, err := strconv.ParseInt(workspaceIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid workspace ID")))
		return
	}

	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Validate that user belongs to the workspace
	if !server.userService.UserBelongsToWorkspace(user.ID, workspaceID) {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("access denied: user does not belong to workspace")))
		return
	}

	// Get file statistics
	stats, err := server.fileService.GetFileStats(workspaceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// @Summary Send File Message
// @Description Send a message with file attachment to channel or direct message
// @Tags files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param message body map[string]interface{} true "File message details" example({"workspace_id": 1, "channel_id": 2, "file_id": 3, "content": "Check this out!"})
// @Success 201 {object} map[string]interface{} "File message sent successfully"
// @Failure 400 {object} map[string]string "Invalid request or file access denied"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 403 {object} map[string]string "Workspace membership required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /files/message [post]
func (server *Server) sendFileMessage(ctx *gin.Context) {
	var req struct {
		WorkspaceID int64  `json:"workspace_id" binding:"required"`
		ChannelID   *int64 `json:"channel_id"`
		ReceiverID  *int64 `json:"receiver_id"`
		FileID      int64  `json:"file_id" binding:"required"`
		Content     string `json:"content"` // Optional text content with the file
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(*db.User)

	// Validate that user belongs to the workspace
	if !server.userService.UserBelongsToWorkspace(user.ID, req.WorkspaceID) {
		ctx.JSON(http.StatusForbidden, errorResponse(fmt.Errorf("access denied: user does not belong to workspace")))
		return
	}

	// Validate channel or receiver
	if req.ChannelID == nil && req.ReceiverID == nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("either channel_id or receiver_id must be specified")))
		return
	}

	if req.ChannelID != nil && req.ReceiverID != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("cannot specify both channel_id and receiver_id")))
		return
	}

	// Check file access
	_, err := server.fileService.GetFile(req.FileID, user.ID, req.WorkspaceID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("invalid file or access denied: %w", err)))
		return
	}

	// Create file message using message service
	var messageResponse *service.MessageResponse

	if req.ChannelID != nil {
		// Channel message
		messageReq := service.CreateChannelMessageRequest{
			WorkspaceID: req.WorkspaceID,
			ChannelID:   *req.ChannelID,
			Content:     req.Content,
			ContentType: "file",
			FileID:      &req.FileID,
		}
		messageResponse, err = server.messageService.CreateChannelMessage(messageReq, user.ID)
	} else {
		// Direct message
		messageReq := service.CreateDirectMessageRequest{
			WorkspaceID: req.WorkspaceID,
			ReceiverID:  *req.ReceiverID,
			Content:     req.Content,
			ContentType: "file",
			FileID:      &req.FileID,
		}
		messageResponse, err = server.messageService.CreateDirectMessage(messageReq, user.ID)
	}

	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "File message sent successfully",
		"data":    messageResponse,
	})
}
