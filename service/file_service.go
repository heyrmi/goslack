package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/util"
)

// FileService handles file upload, download, and management operations
type FileService struct {
	store  db.Store
	config util.Config
}

// NewFileService creates a new file service instance
func NewFileService(store db.Store, config util.Config) *FileService {
	return &FileService{
		store:  store,
		config: config,
	}
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	WorkspaceID int64                 `form:"workspace_id" binding:"required"`
	ChannelID   *int64                `form:"channel_id"`
	ReceiverID  *int64                `form:"receiver_id"`
	File        *multipart.FileHeader `form:"file" binding:"required"`
	IsPublic    bool                  `form:"is_public"`
}

// FileResponse represents a file response
type FileResponse struct {
	ID               int64        `json:"id"`
	OriginalFilename string       `json:"original_filename"`
	FileSize         int64        `json:"file_size"`
	MimeType         string       `json:"mime_type"`
	DownloadURL      string       `json:"download_url"`
	ThumbnailURL     string       `json:"thumbnail_url,omitempty"`
	Uploader         UserResponse `json:"uploader"`
	CreatedAt        time.Time    `json:"created_at"`
	IsPublic         bool         `json:"is_public"`
}

// FileUploadProgress represents file upload progress for WebSocket
type FileUploadProgress struct {
	FileID   int64   `json:"file_id"`
	Progress float64 `json:"progress"`
	Status   string  `json:"status"`
	Error    string  `json:"error,omitempty"`
}

// AllowedFileTypes defines the allowed MIME types for file uploads
var AllowedFileTypes = map[string]bool{
	"image/jpeg":                   true,
	"image/png":                    true,
	"image/gif":                    true,
	"image/webp":                   true,
	"image/svg+xml":                true,
	"application/pdf":              true,
	"text/plain":                   true,
	"application/zip":              true,
	"application/x-zip-compressed": true,
	"application/json":             true,
	"text/csv":                     true,
	"application/vnd.ms-excel":     true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
}

// ValidateFile validates the uploaded file
func (s *FileService) ValidateFile(header *multipart.FileHeader) error {
	// Check file size
	if header.Size > s.config.FileMaxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size of %d bytes", header.Size, s.config.FileMaxSize)
	}

	if header.Size == 0 {
		return errors.New("file cannot be empty")
	}

	// Parse allowed types from config
	allowedTypes := make(map[string]bool)
	for _, mimeType := range strings.Split(s.config.FileAllowedTypes, ",") {
		allowedTypes[strings.TrimSpace(mimeType)] = true
	}

	// Check MIME type from header
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Try to detect from filename extension
		ext := strings.ToLower(filepath.Ext(header.Filename))
		contentType = s.getMimeTypeFromExtension(ext)
	}

	if !allowedTypes[contentType] {
		return fmt.Errorf("file type '%s' is not allowed", contentType)
	}

	// Validate filename
	if header.Filename == "" {
		return errors.New("filename cannot be empty")
	}

	if len(header.Filename) > 255 {
		return errors.New("filename too long (maximum 255 characters)")
	}

	return nil
}

// getMimeTypeFromExtension returns MIME type based on file extension
func (s *FileService) getMimeTypeFromExtension(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".zip":
		return "application/zip"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}

// CalculateFileHash calculates SHA-256 hash of the file content
func (s *FileService) CalculateFileHash(file multipart.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Reset file position for subsequent reads
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to reset file position: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GenerateUniqueFilename generates a unique filename to prevent conflicts
func (s *FileService) GenerateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	name := strings.TrimSuffix(originalFilename, ext)

	// Sanitize filename
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	// Add UUID to ensure uniqueness
	uuid := uuid.New().String()
	return fmt.Sprintf("%s_%s%s", name, uuid, ext)
}

// EnsureUploadDirectory creates the upload directory if it doesn't exist
func (s *FileService) EnsureUploadDirectory() error {
	return os.MkdirAll(s.config.FileStoragePath, 0755)
}

// CheckDuplicateFile checks if a file with the same hash already exists in the workspace
func (s *FileService) CheckDuplicateFile(hash string, workspaceID int64) (*db.File, error) {
	if !s.config.EnableFileDeduplication {
		return nil, nil
	}

	ctx := context.Background()
	file, err := s.store.GetFileByHash(ctx, db.GetFileByHashParams{
		FileHash:    hash,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check duplicate file: %w", err)
	}

	return &file, nil
}

// UploadFile handles the complete file upload process
func (s *FileService) UploadFile(req FileUploadRequest, uploaderID int64) (*FileResponse, error) {
	// Validate file
	if err := s.ValidateFile(req.File); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Open uploaded file
	src, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Calculate file hash for deduplication
	hash, err := s.CalculateFileHash(src)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Check for duplicate files
	if duplicate, err := s.CheckDuplicateFile(hash, req.WorkspaceID); err != nil {
		return nil, fmt.Errorf("failed to check for duplicates: %w", err)
	} else if duplicate != nil {
		// Return existing file if deduplication is enabled
		return s.convertToFileResponse(*duplicate)
	}

	// Ensure upload directory exists
	if err := s.EnsureUploadDirectory(); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename
	storedFilename := s.GenerateUniqueFilename(req.File.Filename)
	filePath := filepath.Join(s.config.FileStoragePath, storedFilename)

	// Create database record first (with upload_completed = false)
	contentType := req.File.Header.Get("Content-Type")
	if contentType == "" {
		contentType = s.getMimeTypeFromExtension(filepath.Ext(req.File.Filename))
	}

	createFileParams := db.CreateFileParams{
		WorkspaceID:      req.WorkspaceID,
		UploaderID:       uploaderID,
		OriginalFilename: req.File.Filename,
		StoredFilename:   storedFilename,
		FilePath:         filePath,
		FileSize:         req.File.Size,
		MimeType:         contentType,
		FileHash:         hash,
		IsPublic:         req.IsPublic,
		UploadCompleted:  false,
		ThumbnailPath:    sql.NullString{Valid: false},
	}

	ctx := context.Background()
	file, err := s.store.CreateFile(ctx, createFileParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		// Clean up file and database record on failure
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Update file as completed
	if err := s.store.UpdateFileUploadStatus(ctx, db.UpdateFileUploadStatusParams{
		ID:              file.ID,
		UploadCompleted: true,
	}); err != nil {
		// Clean up file on database update failure
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to mark file upload as completed: %w", err)
	}

	// Generate thumbnail for images if enabled
	if s.config.EnableThumbnails && s.isImageFile(contentType) {
		if thumbnailPath, err := s.GenerateThumbnail(filePath); err == nil {
			s.store.UpdateFileThumbnail(ctx, db.UpdateFileThumbnailParams{
				ID:            file.ID,
				ThumbnailPath: sql.NullString{String: thumbnailPath, Valid: true},
			})
		}
		// Don't fail upload if thumbnail generation fails
	}

	// Update file record with completion status
	file.UploadCompleted = true

	return s.convertToFileResponse(file)
}

// isImageFile checks if the MIME type is an image
func (s *FileService) isImageFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// GenerateThumbnail generates a thumbnail for image files
func (s *FileService) GenerateThumbnail(filePath string) (string, error) {
	// This is a placeholder - you would implement actual thumbnail generation here
	// For now, we'll just return the original file path
	// In a real implementation, you'd use a library like github.com/disintegration/imaging

	ext := filepath.Ext(filePath)
	thumbnailPath := strings.TrimSuffix(filePath, ext) + "_thumb" + ext

	// For now, just copy the original file as thumbnail
	// In production, you'd resize it to a smaller dimension
	src, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(thumbnailPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		os.Remove(thumbnailPath)
		return "", err
	}

	return thumbnailPath, nil
}

// GetFile retrieves a file by ID with permission check
func (s *FileService) GetFile(fileID, userID, workspaceID int64) (*FileResponse, error) {
	// Check file access permissions
	ctx := context.Background()
	hasAccess, err := s.store.CheckFileAccess(ctx, db.CheckFileAccessParams{
		FileID:     fileID,
		UploaderID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check file access: %w", err)
	}

	if !hasAccess {
		return nil, errors.New("access denied: you don't have permission to access this file")
	}

	// Get file with permission check
	file, err := s.store.GetFileWithPermissionCheck(ctx, db.GetFileWithPermissionCheckParams{
		ID:          fileID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("file not found")
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return s.convertToFileResponseWithUploader(file)
}

// convertToFileResponse converts a database File to FileResponse
func (s *FileService) convertToFileResponse(file db.File) (*FileResponse, error) {
	// Get uploader info
	ctx := context.Background()
	uploader, err := s.store.GetUser(ctx, file.UploaderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get uploader info: %w", err)
	}

	response := &FileResponse{
		ID:               file.ID,
		OriginalFilename: file.OriginalFilename,
		FileSize:         file.FileSize,
		MimeType:         file.MimeType,
		DownloadURL:      fmt.Sprintf("/api/files/%d/download", file.ID),
		CreatedAt:        file.CreatedAt,
		IsPublic:         file.IsPublic,
		Uploader: UserResponse{
			ID:        uploader.ID,
			Email:     uploader.Email,
			FirstName: uploader.FirstName,
			LastName:  uploader.LastName,
		},
	}

	// Add thumbnail URL if available
	if file.ThumbnailPath.Valid {
		response.ThumbnailURL = fmt.Sprintf("/api/files/%d/thumbnail", file.ID)
	}

	return response, nil
}

// convertToFileResponseWithUploader converts a GetFileWithPermissionCheckRow to FileResponse
func (s *FileService) convertToFileResponseWithUploader(row db.GetFileWithPermissionCheckRow) (*FileResponse, error) {
	response := &FileResponse{
		ID:               row.ID,
		OriginalFilename: row.OriginalFilename,
		FileSize:         row.FileSize,
		MimeType:         row.MimeType,
		DownloadURL:      fmt.Sprintf("/api/files/%d/download", row.ID),
		CreatedAt:        row.CreatedAt,
		IsPublic:         row.IsPublic,
		Uploader: UserResponse{
			ID:        row.UploaderID,
			Email:     row.UploaderEmail,
			FirstName: row.UploaderFirstName,
			LastName:  row.UploaderLastName,
		},
	}

	// Add thumbnail URL if available
	if row.ThumbnailPath.Valid {
		response.ThumbnailURL = fmt.Sprintf("/api/files/%d/thumbnail", row.ID)
	}

	return response, nil
}

// ListWorkspaceFiles lists files in a workspace with pagination
func (s *FileService) ListWorkspaceFiles(workspaceID int64, limit, offset int32) ([]*FileResponse, error) {
	ctx := context.Background()
	files, err := s.store.ListWorkspaceFiles(ctx, db.ListWorkspaceFilesParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace files: %w", err)
	}

	responses := make([]*FileResponse, len(files))
	for i, file := range files {
		responses[i] = &FileResponse{
			ID:               file.ID,
			OriginalFilename: file.OriginalFilename,
			FileSize:         file.FileSize,
			MimeType:         file.MimeType,
			DownloadURL:      fmt.Sprintf("/api/files/%d/download", file.ID),
			CreatedAt:        file.CreatedAt,
			IsPublic:         file.IsPublic,
			Uploader: UserResponse{
				ID:        file.UploaderID,
				Email:     file.UploaderEmail,
				FirstName: file.UploaderFirstName,
				LastName:  file.UploaderLastName,
			},
		}

		if file.ThumbnailPath.Valid {
			responses[i].ThumbnailURL = fmt.Sprintf("/api/files/%d/thumbnail", file.ID)
		}
	}

	return responses, nil
}

// DeleteFile deletes a file (only by the uploader)
func (s *FileService) DeleteFile(fileID, userID int64) error {
	// Get file to check ownership and get file path
	ctx := context.Background()
	file, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("file not found")
		}
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Check if user is the uploader
	if file.UploaderID != userID {
		return errors.New("access denied: only the file uploader can delete this file")
	}

	// Delete file from database
	if err := s.store.DeleteFile(ctx, db.DeleteFileParams{
		ID:         fileID,
		UploaderID: userID,
	}); err != nil {
		return fmt.Errorf("failed to delete file from database: %w", err)
	}

	// Delete file from disk
	if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
		// Log error but don't fail the operation
		// In production, you might want to queue this for cleanup
		fmt.Printf("Warning: failed to delete file from disk: %v\n", err)
	}

	// Delete thumbnail if exists
	if file.ThumbnailPath.Valid {
		if err := os.Remove(file.ThumbnailPath.String); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to delete thumbnail from disk: %v\n", err)
		}
	}

	return nil
}

// GetFileContent returns the file content for download
func (s *FileService) GetFileContent(fileID, userID int64) (*os.File, *db.File, error) {
	// Check file access permissions
	ctx := context.Background()
	hasAccess, err := s.store.CheckFileAccess(ctx, db.CheckFileAccessParams{
		FileID:     fileID,
		UploaderID: userID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check file access: %w", err)
	}

	if !hasAccess {
		return nil, nil, errors.New("access denied: you don't have permission to download this file")
	}

	// Get file info
	file, err := s.store.GetFile(ctx, fileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("file not found")
		}
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	}

	if !file.UploadCompleted {
		return nil, nil, errors.New("file upload not completed")
	}

	// Open file for reading
	fileContent, err := os.Open(file.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errors.New("file not found on disk")
		}
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	return fileContent, &file, nil
}

// CleanupIncompleteUploads removes incomplete uploads older than 1 hour
func (s *FileService) CleanupIncompleteUploads() error {
	ctx := context.Background()
	return s.store.CleanupIncompleteUploads(ctx)
}

// GetFileStats returns file statistics for a workspace
func (s *FileService) GetFileStats(workspaceID int64) (*db.GetFileStatsRow, error) {
	ctx := context.Background()
	stats, err := s.store.GetFileStats(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	return &stats, nil
}
