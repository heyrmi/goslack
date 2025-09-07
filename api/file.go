package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/heyrmi/goslack/service"
)

// uploadFile handles file upload requests
func (server *Server) uploadFile(ctx *gin.Context) {
	// Get current user
	currentUser, exists := ctx.Get(currentUserKey)
	if !exists {
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("user not found in context")))
		return
	}
	user := currentUser.(service.UserResponse)

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

// downloadFile handles file download requests
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
	user := currentUser.(service.UserResponse)

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

// getFile retrieves file metadata
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
	user := currentUser.(service.UserResponse)

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

// listWorkspaceFiles lists files in a workspace
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
	user := currentUser.(service.UserResponse)

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

// deleteFile handles file deletion
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
	user := currentUser.(service.UserResponse)

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

// getFileStats returns file statistics for a workspace
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
	user := currentUser.(service.UserResponse)

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

// sendFileMessage creates a message with file attachment
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
	user := currentUser.(service.UserResponse)

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
