package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomFile(t *testing.T) File {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	return createRandomFileForWorkspace(t, workspace.ID, user.ID)
}

func createRandomFileForWorkspace(t *testing.T, workspaceID, uploaderID int64) File {
	arg := CreateFileParams{
		WorkspaceID:      workspaceID,
		UploaderID:       uploaderID,
		OriginalFilename: util.RandomString(10) + ".jpg",
		StoredFilename:   util.RandomString(15) + ".jpg",
		FilePath:         "/uploads/" + util.RandomString(20) + ".jpg",
		FileSize:         int64(util.RandomInt(1000, 10000000)), // 1KB to 10MB
		MimeType:         "image/jpeg",
		FileHash:         util.RandomString(64), // SHA-256 hash
		IsPublic:         util.RandomBool(),
		UploadCompleted:  true,
		ThumbnailPath: func() sql.NullString {
			if util.RandomBool() {
				return sql.NullString{String: "/thumbs/" + util.RandomString(20) + ".jpg", Valid: true}
			}
			return sql.NullString{Valid: false}
		}(),
	}

	file, err := testQueries.CreateFile(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, file)

	require.Equal(t, arg.WorkspaceID, file.WorkspaceID)
	require.Equal(t, arg.UploaderID, file.UploaderID)
	require.Equal(t, arg.OriginalFilename, file.OriginalFilename)
	require.Equal(t, arg.StoredFilename, file.StoredFilename)
	require.Equal(t, arg.FilePath, file.FilePath)
	require.Equal(t, arg.FileSize, file.FileSize)
	require.Equal(t, arg.MimeType, file.MimeType)
	require.Equal(t, arg.FileHash, file.FileHash)
	require.Equal(t, arg.IsPublic, file.IsPublic)
	require.Equal(t, arg.UploadCompleted, file.UploadCompleted)
	require.Equal(t, arg.ThumbnailPath, file.ThumbnailPath)

	require.NotZero(t, file.ID)
	require.NotZero(t, file.CreatedAt)
	require.NotZero(t, file.UpdatedAt)

	return file
}

func TestCreateFile(t *testing.T) {
	createRandomFile(t)
}

func TestGetFile(t *testing.T) {
	file1 := createRandomFile(t)

	file2, err := testQueries.GetFile(context.Background(), file1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, file2)

	require.Equal(t, file1.ID, file2.ID)
	require.Equal(t, file1.WorkspaceID, file2.WorkspaceID)
	require.Equal(t, file1.UploaderID, file2.UploaderID)
	require.Equal(t, file1.OriginalFilename, file2.OriginalFilename)
	require.Equal(t, file1.StoredFilename, file2.StoredFilename)
	require.Equal(t, file1.FilePath, file2.FilePath)
	require.Equal(t, file1.FileSize, file2.FileSize)
	require.Equal(t, file1.MimeType, file2.MimeType)
	require.Equal(t, file1.FileHash, file2.FileHash)
	require.Equal(t, file1.IsPublic, file2.IsPublic)
	require.Equal(t, file1.UploadCompleted, file2.UploadCompleted)
	require.Equal(t, file1.ThumbnailPath, file2.ThumbnailPath)
	require.WithinDuration(t, file1.CreatedAt, file2.CreatedAt, time.Second)
	require.WithinDuration(t, file1.UpdatedAt, file2.UpdatedAt, time.Second)
}

func TestGetFileByHash(t *testing.T) {
	file1 := createRandomFile(t)

	arg := GetFileByHashParams{
		FileHash:    file1.FileHash,
		WorkspaceID: file1.WorkspaceID,
	}

	file2, err := testQueries.GetFileByHash(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, file2)

	require.Equal(t, file1.ID, file2.ID)
	require.Equal(t, file1.FileHash, file2.FileHash)
	require.Equal(t, file1.WorkspaceID, file2.WorkspaceID)
}

func TestUpdateFileUploadStatus(t *testing.T) {
	file1 := createRandomFile(t)

	arg := UpdateFileUploadStatusParams{
		ID:              file1.ID,
		UploadCompleted: !file1.UploadCompleted,
	}

	err := testQueries.UpdateFileUploadStatus(context.Background(), arg)
	require.NoError(t, err)

	file2, err := testQueries.GetFile(context.Background(), file1.ID)
	require.NoError(t, err)
	require.Equal(t, arg.UploadCompleted, file2.UploadCompleted)
	require.True(t, file2.UpdatedAt.After(file1.UpdatedAt))
}

func TestUpdateFileThumbnail(t *testing.T) {
	file1 := createRandomFile(t)

	newThumbnailPath := "/new/thumbnail/path.jpg"
	arg := UpdateFileThumbnailParams{
		ID:            file1.ID,
		ThumbnailPath: sql.NullString{String: newThumbnailPath, Valid: true},
	}

	err := testQueries.UpdateFileThumbnail(context.Background(), arg)
	require.NoError(t, err)

	file2, err := testQueries.GetFile(context.Background(), file1.ID)
	require.NoError(t, err)
	require.Equal(t, arg.ThumbnailPath, file2.ThumbnailPath)
	require.True(t, file2.UpdatedAt.After(file1.UpdatedAt))
}

func TestListWorkspaceFiles(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)

	// Create multiple files
	for i := 0; i < 10; i++ {
		createRandomFileForWorkspace(t, workspace.ID, user.ID)
	}

	arg := ListWorkspaceFilesParams{
		WorkspaceID: workspace.ID,
		Limit:       5,
		Offset:      0,
	}

	files, err := testQueries.ListWorkspaceFiles(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, files, 5)

	for _, file := range files {
		require.Equal(t, workspace.ID, file.WorkspaceID)
		require.NotEmpty(t, file.UploaderFirstName)
		require.NotEmpty(t, file.UploaderLastName)
		require.NotEmpty(t, file.UploaderEmail)
	}
}

func TestListUserFiles(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)

	// Create multiple files for the user
	for i := 0; i < 5; i++ {
		createRandomFileForWorkspace(t, workspace.ID, user.ID)
	}

	arg := ListUserFilesParams{
		UploaderID:  user.ID,
		WorkspaceID: workspace.ID,
		Limit:       3,
		Offset:      0,
	}

	files, err := testQueries.ListUserFiles(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, files, 3)

	for _, file := range files {
		require.Equal(t, user.ID, file.UploaderID)
		require.Equal(t, workspace.ID, file.WorkspaceID)
	}
}

func TestDeleteFile(t *testing.T) {
	file1 := createRandomFile(t)

	arg := DeleteFileParams{
		ID:         file1.ID,
		UploaderID: file1.UploaderID,
	}

	err := testQueries.DeleteFile(context.Background(), arg)
	require.NoError(t, err)

	file2, err := testQueries.GetFile(context.Background(), file1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, file2)
}

func TestGetFileWithPermissionCheck(t *testing.T) {
	file1 := createRandomFile(t)

	arg := GetFileWithPermissionCheckParams{
		ID:          file1.ID,
		WorkspaceID: file1.WorkspaceID,
	}

	file2, err := testQueries.GetFileWithPermissionCheck(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, file2)

	require.Equal(t, file1.ID, file2.ID)
	require.Equal(t, file1.WorkspaceID, file2.WorkspaceID)
	require.NotEmpty(t, file2.UploaderFirstName)
	require.NotEmpty(t, file2.UploaderLastName)
	require.NotEmpty(t, file2.UploaderEmail)
}

func TestCheckFileAccess(t *testing.T) {
	file1 := createRandomFile(t)

	// Test file access for uploader (should have access)
	arg := CheckFileAccessParams{
		FileID:     file1.ID,
		UploaderID: file1.UploaderID,
	}

	hasAccess, err := testQueries.CheckFileAccess(context.Background(), arg)
	require.NoError(t, err)
	require.True(t, hasAccess)

	// Test file access for different user (should not have access unless public)
	otherUser := createRandomUser(t)
	arg2 := CheckFileAccessParams{
		FileID:     file1.ID,
		UploaderID: otherUser.ID,
	}

	hasAccess2, err := testQueries.CheckFileAccess(context.Background(), arg2)
	require.NoError(t, err)

	if file1.IsPublic {
		require.True(t, hasAccess2)
	} else {
		require.False(t, hasAccess2)
	}
}

func TestGetFileStats(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)

	// Create files with known properties
	var totalSize int64
	imageCount := 0
	pdfCount := 0
	totalFiles := 5

	for i := 0; i < totalFiles; i++ {
		var mimeType string
		var fileSize int64 = int64(util.RandomInt(1000, 5000))

		if i%2 == 0 {
			mimeType = "image/jpeg"
			imageCount++
		} else if i%3 == 0 {
			mimeType = "application/pdf"
			pdfCount++
		} else {
			mimeType = "text/plain"
		}

		totalSize += fileSize

		arg := CreateFileParams{
			WorkspaceID:      workspace.ID,
			UploaderID:       user.ID,
			OriginalFilename: util.RandomString(10) + ".file",
			StoredFilename:   util.RandomString(15) + ".file",
			FilePath:         "/uploads/" + util.RandomString(20) + ".file",
			FileSize:         fileSize,
			MimeType:         mimeType,
			FileHash:         util.RandomString(64),
			IsPublic:         false,
			UploadCompleted:  true,
			ThumbnailPath:    sql.NullString{Valid: false},
		}

		_, err := testQueries.CreateFile(context.Background(), arg)
		require.NoError(t, err)
	}

	stats, err := testQueries.GetFileStats(context.Background(), workspace.ID)
	require.NoError(t, err)

	require.Equal(t, int64(totalFiles), stats.TotalFiles)

	// Handle TotalSize which might be returned as different types
	if totalSizeBytes, ok := stats.TotalSize.([]uint8); ok {
		// Convert byte slice to string and then to int64
		totalSizeStr := string(totalSizeBytes)
		require.Equal(t, fmt.Sprintf("%d", totalSize), totalSizeStr)
	} else if totalSizeInt, ok := stats.TotalSize.(int64); ok {
		require.Equal(t, totalSize, totalSizeInt)
	} else {
		t.Fatalf("Unexpected type for TotalSize: %T", stats.TotalSize)
	}

	require.Equal(t, int64(imageCount), stats.ImageCount)
	require.Equal(t, int64(pdfCount), stats.PdfCount)
}

func TestCleanupIncompleteUploads(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)

	// Create an incomplete upload
	arg := CreateFileParams{
		WorkspaceID:      workspace.ID,
		UploaderID:       user.ID,
		OriginalFilename: "incomplete.jpg",
		StoredFilename:   util.RandomString(20) + "_incomplete_stored.jpg", // Make unique
		FilePath:         "/uploads/incomplete_" + util.RandomString(10) + ".jpg",
		FileSize:         1000,
		MimeType:         "image/jpeg",
		FileHash:         util.RandomString(64),
		IsPublic:         false,
		UploadCompleted:  false, // Incomplete
		ThumbnailPath:    sql.NullString{Valid: false},
	}

	file, err := testQueries.CreateFile(context.Background(), arg)
	require.NoError(t, err)

	// Run cleanup (this would normally clean files older than 1 hour)
	err = testQueries.CleanupIncompleteUploads(context.Background())
	require.NoError(t, err)

	// File should still exist since it was just created
	_, err = testQueries.GetFile(context.Background(), file.ID)
	require.NoError(t, err)
}

// Message Files Tests

func TestCreateMessageFile(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	channel := createRandomChannelForWorkspace(t, workspace.ID, user.ID)

	// Create message
	messageArg := CreateChannelMessageParams{
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    user.ID,
		Content:     "Test message with file",
		ContentType: "file",
	}

	message, err := testQueries.CreateChannelMessage(context.Background(), messageArg)
	require.NoError(t, err)

	// Create file
	file := createRandomFileForWorkspace(t, workspace.ID, user.ID)

	// Link message and file
	arg := CreateMessageFileParams{
		MessageID: message.ID,
		FileID:    file.ID,
	}

	messageFile, err := testQueries.CreateMessageFile(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, messageFile)

	require.Equal(t, arg.MessageID, messageFile.MessageID)
	require.Equal(t, arg.FileID, messageFile.FileID)
	require.NotZero(t, messageFile.ID)
	require.NotZero(t, messageFile.CreatedAt)
}

func TestGetMessageFiles(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	channel := createRandomChannelForWorkspace(t, workspace.ID, user.ID)

	// Create message
	messageArg := CreateChannelMessageParams{
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    user.ID,
		Content:     "Test message with multiple files",
		ContentType: "file",
	}

	message, err := testQueries.CreateChannelMessage(context.Background(), messageArg)
	require.NoError(t, err)

	// Create multiple files and link them
	fileCount := 3
	for i := 0; i < fileCount; i++ {
		file := createRandomFileForWorkspace(t, workspace.ID, user.ID)

		arg := CreateMessageFileParams{
			MessageID: message.ID,
			FileID:    file.ID,
		}

		_, err := testQueries.CreateMessageFile(context.Background(), arg)
		require.NoError(t, err)
	}

	files, err := testQueries.GetMessageFiles(context.Background(), message.ID)
	require.NoError(t, err)
	require.Len(t, files, fileCount)

	for _, file := range files {
		require.Equal(t, workspace.ID, file.WorkspaceID)
		require.NotEmpty(t, file.UploaderFirstName)
		require.NotEmpty(t, file.UploaderLastName)
		require.NotEmpty(t, file.UploaderEmail)
	}
}

func TestGetFileMessages(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	channel := createRandomChannelForWorkspace(t, workspace.ID, user.ID)

	// Create file
	file := createRandomFileForWorkspace(t, workspace.ID, user.ID)

	// Create multiple messages and link them to the file
	messageCount := 2
	for i := 0; i < messageCount; i++ {
		messageArg := CreateChannelMessageParams{
			WorkspaceID: workspace.ID,
			ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
			SenderID:    user.ID,
			Content:     util.RandomString(20),
			ContentType: "file",
		}

		message, err := testQueries.CreateChannelMessage(context.Background(), messageArg)
		require.NoError(t, err)

		arg := CreateMessageFileParams{
			MessageID: message.ID,
			FileID:    file.ID,
		}

		_, err = testQueries.CreateMessageFile(context.Background(), arg)
		require.NoError(t, err)
	}

	messages, err := testQueries.GetFileMessages(context.Background(), file.ID)
	require.NoError(t, err)
	require.Len(t, messages, messageCount)

	for _, message := range messages {
		require.Equal(t, workspace.ID, message.WorkspaceID)
		require.Equal(t, user.ID, message.SenderID)
		require.NotEmpty(t, message.SenderFirstName)
		require.NotEmpty(t, message.SenderLastName)
		require.NotEmpty(t, message.SenderEmail)
	}
}

func TestDeleteMessageFile(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	channel := createRandomChannelForWorkspace(t, workspace.ID, user.ID)

	// Create message and file
	messageArg := CreateChannelMessageParams{
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    user.ID,
		Content:     "Test message",
		ContentType: "file",
	}

	message, err := testQueries.CreateChannelMessage(context.Background(), messageArg)
	require.NoError(t, err)

	file := createRandomFileForWorkspace(t, workspace.ID, user.ID)

	// Link message and file
	linkArg := CreateMessageFileParams{
		MessageID: message.ID,
		FileID:    file.ID,
	}

	_, err = testQueries.CreateMessageFile(context.Background(), linkArg)
	require.NoError(t, err)

	// Delete the link
	deleteArg := DeleteMessageFileParams{
		MessageID: message.ID,
		FileID:    file.ID,
	}

	err = testQueries.DeleteMessageFile(context.Background(), deleteArg)
	require.NoError(t, err)

	// Verify link is deleted
	files, err := testQueries.GetMessageFiles(context.Background(), message.ID)
	require.NoError(t, err)
	require.Empty(t, files)
}

// File Shares Tests

func TestCreateFileShare(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	channel := createRandomChannelForWorkspace(t, workspace.ID, user.ID)
	file := createRandomFileForWorkspace(t, workspace.ID, user.ID)

	// Test channel share
	arg := CreateFileShareParams{
		FileID:           file.ID,
		SharedBy:         user.ID,
		ChannelID:        sql.NullInt64{Int64: channel.ID, Valid: true},
		SharedWithUserID: sql.NullInt64{Valid: false},
		Permission:       "view",
		ExpiresAt:        sql.NullTime{Time: time.Now().UTC().Add(24 * time.Hour), Valid: true},
	}

	share, err := testQueries.CreateFileShare(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, share)

	require.Equal(t, arg.FileID, share.FileID)
	require.Equal(t, arg.SharedBy, share.SharedBy)
	require.Equal(t, arg.ChannelID, share.ChannelID)
	require.Equal(t, arg.SharedWithUserID, share.SharedWithUserID)
	require.Equal(t, arg.Permission, share.Permission)
	require.True(t, share.ExpiresAt.Valid)
	require.NotZero(t, share.ID)
	require.NotZero(t, share.CreatedAt)
}

func TestCreateFileShareWithUser(t *testing.T) {
	user1 := createRandomUser(t)
	user2 := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user1.ID)
	file := createRandomFileForWorkspace(t, workspace.ID, user1.ID)

	// Test direct user share
	arg := CreateFileShareParams{
		FileID:           file.ID,
		SharedBy:         user1.ID,
		ChannelID:        sql.NullInt64{Valid: false},
		SharedWithUserID: sql.NullInt64{Int64: user2.ID, Valid: true},
		Permission:       "download",
		ExpiresAt:        sql.NullTime{Valid: false}, // No expiration
	}

	share, err := testQueries.CreateFileShare(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, share)

	require.Equal(t, arg.FileID, share.FileID)
	require.Equal(t, arg.SharedBy, share.SharedBy)
	require.Equal(t, arg.ChannelID, share.ChannelID)
	require.Equal(t, arg.SharedWithUserID, share.SharedWithUserID)
	require.Equal(t, arg.Permission, share.Permission)
	require.Equal(t, arg.ExpiresAt, share.ExpiresAt)
}

func TestGetFileShares(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)
	channel := createRandomChannelForWorkspace(t, workspace.ID, user.ID)
	file := createRandomFileForWorkspace(t, workspace.ID, user.ID)

	// Create multiple shares
	shareCount := 3
	for i := 0; i < shareCount; i++ {
		arg := CreateFileShareParams{
			FileID:           file.ID,
			SharedBy:         user.ID,
			ChannelID:        sql.NullInt64{Int64: channel.ID, Valid: true},
			SharedWithUserID: sql.NullInt64{Valid: false},
			Permission:       "view",
			ExpiresAt:        sql.NullTime{Valid: false},
		}

		_, err := testQueries.CreateFileShare(context.Background(), arg)
		require.NoError(t, err)
	}

	shares, err := testQueries.GetFileShares(context.Background(), file.ID)
	require.NoError(t, err)
	require.Len(t, shares, shareCount)

	for _, share := range shares {
		require.Equal(t, file.ID, share.FileID)
		require.Equal(t, user.ID, share.SharedBy)
		require.NotEmpty(t, share.SharedByFirstName)
		require.NotEmpty(t, share.SharedByLastName)
		require.NotEmpty(t, share.SharedByEmail)
	}
}

func TestGetDuplicateFiles(t *testing.T) {
	user := createRandomUser(t)
	workspace := createRandomWorkspaceForUser(t, user.ID)

	// Create files with same hash (duplicates)
	duplicateHash := util.RandomString(64)
	fileSize := int64(5000)

	for i := 0; i < 3; i++ {
		arg := CreateFileParams{
			WorkspaceID:      workspace.ID,
			UploaderID:       user.ID,
			OriginalFilename: util.RandomString(10) + ".jpg",
			StoredFilename:   util.RandomString(15) + ".jpg",
			FilePath:         "/uploads/" + util.RandomString(20) + ".jpg",
			FileSize:         fileSize,
			MimeType:         "image/jpeg",
			FileHash:         duplicateHash,
			IsPublic:         false,
			UploadCompleted:  true,
			ThumbnailPath:    sql.NullString{Valid: false},
		}

		_, err := testQueries.CreateFile(context.Background(), arg)
		require.NoError(t, err)
	}

	duplicates, err := testQueries.GetDuplicateFiles(context.Background(), workspace.ID)
	require.NoError(t, err)
	require.NotEmpty(t, duplicates)

	found := false
	for _, dup := range duplicates {
		if dup.FileHash == duplicateHash {
			require.Equal(t, int64(3), dup.Count)
			require.Equal(t, fileSize*3, dup.TotalSize)
			found = true
			break
		}
	}
	require.True(t, found, "Should find the duplicate files we created")
}

// Helper functions for creating test data

func createRandomWorkspaceForUser(t *testing.T, userID int64) Workspace {
	// Get the user to find their organization
	user, err := testQueries.GetUser(context.Background(), userID)
	require.NoError(t, err)

	arg := CreateWorkspaceParams{
		OrganizationID: user.OrganizationID, // Use same organization as user
		Name:           util.RandomString(10),
	}

	workspace, err := testQueries.CreateWorkspace(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, workspace)

	// Assign user to workspace
	updateArg := UpdateUserWorkspaceParams{
		ID:          userID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "admin",
	}

	_, err = testQueries.UpdateUserWorkspace(context.Background(), updateArg)
	require.NoError(t, err)

	return workspace
}

func createRandomChannelForWorkspace(t *testing.T, workspaceID, createdBy int64) Channel {
	arg := CreateChannelParams{
		WorkspaceID: workspaceID,
		Name:        util.RandomString(8),
		IsPrivate:   util.RandomBool(),
		CreatedBy:   createdBy,
	}

	channel, err := testQueries.CreateChannel(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, channel)

	return channel
}
