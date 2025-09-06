package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomChannelMessage(t *testing.T, workspace Workspace, channel Channel, sender User) Message {
	arg := CreateChannelMessageParams{
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    sender.ID,
		Content:     util.RandomString(50),
	}

	message, err := testQueries.CreateChannelMessage(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, message)

	require.Equal(t, arg.WorkspaceID, message.WorkspaceID)
	require.Equal(t, arg.ChannelID, message.ChannelID)
	require.Equal(t, arg.SenderID, message.SenderID)
	require.Equal(t, arg.Content, message.Content)
	require.Equal(t, "channel", message.MessageType)

	require.NotZero(t, message.ID)
	require.NotZero(t, message.CreatedAt)
	require.False(t, message.EditedAt.Valid)
	require.False(t, message.DeletedAt.Valid)

	return message
}

func createRandomDirectMessage(t *testing.T, workspace Workspace, sender User, receiver User) Message {
	arg := CreateDirectMessageParams{
		WorkspaceID: workspace.ID,
		SenderID:    sender.ID,
		ReceiverID:  sql.NullInt64{Int64: receiver.ID, Valid: true},
		Content:     util.RandomString(50),
	}

	message, err := testQueries.CreateDirectMessage(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, message)

	require.Equal(t, arg.WorkspaceID, message.WorkspaceID)
	require.Equal(t, arg.SenderID, message.SenderID)
	require.Equal(t, arg.ReceiverID, message.ReceiverID)
	require.Equal(t, arg.Content, message.Content)
	require.Equal(t, "direct", message.MessageType)

	require.NotZero(t, message.ID)
	require.NotZero(t, message.CreatedAt)
	require.False(t, message.EditedAt.Valid)
	require.False(t, message.DeletedAt.Valid)
	require.False(t, message.ChannelID.Valid)

	return message
}

func TestCreateChannelMessage(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	createRandomChannelMessage(t, workspace, channel, user)
}

func TestCreateDirectMessage(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)

	createRandomDirectMessage(t, workspace, user1, user2)
}

func TestGetChannelMessages(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	// Create multiple messages
	var messages []Message
	for i := 0; i < 5; i++ {
		message := createRandomChannelMessage(t, workspace, channel, user)
		messages = append(messages, message)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	arg := GetChannelMessagesParams{
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	}

	result, err := testQueries.GetChannelMessages(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, result, 5)

	// Messages should be ordered by created_at DESC
	for i := 1; i < len(result); i++ {
		require.True(t, result[i-1].CreatedAt.After(result[i].CreatedAt) || result[i-1].CreatedAt.Equal(result[i].CreatedAt))
	}

	// Check that sender information is included
	for _, msg := range result {
		require.Equal(t, user.FirstName, msg.SenderFirstName)
		require.Equal(t, user.LastName, msg.SenderLastName)
		require.Equal(t, user.Email, msg.SenderEmail)
	}
}

func TestGetChannelMessagesPagination(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	// Create 5 messages
	for i := 0; i < 5; i++ {
		createRandomChannelMessage(t, workspace, channel, user)
		time.Sleep(time.Millisecond)
	}

	// Test first page
	arg := GetChannelMessagesParams{
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		WorkspaceID: workspace.ID,
		Limit:       2,
		Offset:      0,
	}

	page1, err := testQueries.GetChannelMessages(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	// Test second page
	arg.Offset = 2
	page2, err := testQueries.GetChannelMessages(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	// Test third page
	arg.Offset = 4
	page3, err := testQueries.GetChannelMessages(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page3, 1)

	// Verify no duplicates
	allMessageIDs := make(map[int64]bool)
	for _, message := range append(append(page1, page2...), page3...) {
		require.False(t, allMessageIDs[message.ID], "Duplicate message found in pagination")
		allMessageIDs[message.ID] = true
	}
}

func TestGetDirectMessagesBetweenUsers(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	user3 := createRandomUserForOrganization(t, workspace.OrganizationID)

	// Create messages between user1 and user2
	message1 := createRandomDirectMessage(t, workspace, user1, user2)
	message2 := createRandomDirectMessage(t, workspace, user2, user1)

	// Create a message between user1 and user3 (should not be included)
	createRandomDirectMessage(t, workspace, user1, user3)

	arg := GetDirectMessagesBetweenUsersParams{
		WorkspaceID: workspace.ID,
		SenderID:    user1.ID,
		ReceiverID:  sql.NullInt64{Int64: user2.ID, Valid: true},
		Limit:       10,
		Offset:      0,
	}

	result, err := testQueries.GetDirectMessagesBetweenUsers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Check that both messages are included
	messageIDs := make(map[int64]bool)
	for _, msg := range result {
		messageIDs[msg.ID] = true
		require.Equal(t, workspace.ID, msg.WorkspaceID)
		require.Equal(t, "direct", msg.MessageType)

		// Should be messages between user1 and user2
		require.True(t,
			(msg.SenderID == user1.ID && msg.ReceiverID.Int64 == user2.ID) ||
				(msg.SenderID == user2.ID && msg.ReceiverID.Int64 == user1.ID))
	}

	require.True(t, messageIDs[message1.ID])
	require.True(t, messageIDs[message2.ID])
}

func TestGetMessageByID(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message1 := createRandomChannelMessage(t, workspace, channel, user)

	result, err := testQueries.GetMessageByID(context.Background(), message1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, message1.ID, result.ID)
	require.Equal(t, message1.WorkspaceID, result.WorkspaceID)
	require.Equal(t, message1.ChannelID, result.ChannelID)
	require.Equal(t, message1.SenderID, result.SenderID)
	require.Equal(t, message1.Content, result.Content)
	require.Equal(t, user.FirstName, result.SenderFirstName)
	require.Equal(t, user.LastName, result.SenderLastName)
	require.Equal(t, user.Email, result.SenderEmail)
}

func TestGetMessageByIDNotFound(t *testing.T) {
	_, err := testQueries.GetMessageByID(context.Background(), 999999)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestUpdateMessageContent(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	newContent := "Updated content: " + util.RandomString(20)
	arg := UpdateMessageContentParams{
		ID:      message.ID,
		Content: newContent,
	}

	updatedMessage, err := testQueries.UpdateMessageContent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedMessage)

	require.Equal(t, message.ID, updatedMessage.ID)
	require.Equal(t, newContent, updatedMessage.Content)
	require.True(t, updatedMessage.EditedAt.Valid)
	require.True(t, updatedMessage.EditedAt.Time.After(message.CreatedAt))

	// Other fields should remain the same
	require.Equal(t, message.WorkspaceID, updatedMessage.WorkspaceID)
	require.Equal(t, message.SenderID, updatedMessage.SenderID)
	require.Equal(t, message.MessageType, updatedMessage.MessageType)
}

func TestSoftDeleteMessage(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	// Soft delete the message
	err := testQueries.SoftDeleteMessage(context.Background(), message.ID)
	require.NoError(t, err)

	// Message should not appear in channel messages (due to WHERE deleted_at IS NULL)
	arg := GetChannelMessagesParams{
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	}

	result, err := testQueries.GetChannelMessages(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, result, 0) // Should be empty because message is soft deleted

	// But GetMessageByID should still return error (due to WHERE deleted_at IS NULL)
	_, err = testQueries.GetMessageByID(context.Background(), message.ID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetRecentWorkspaceMessages(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel1 := createRandomChannel(t, workspace, user1)
	channel2 := createRandomChannel(t, workspace, user1)

	// Create messages in different channels and direct messages
	channelMessage1 := createRandomChannelMessage(t, workspace, channel1, user1)
	time.Sleep(time.Millisecond)
	channelMessage2 := createRandomChannelMessage(t, workspace, channel2, user2)
	time.Sleep(time.Millisecond)
	directMessage := createRandomDirectMessage(t, workspace, user1, user2)

	arg := GetRecentWorkspaceMessagesParams{
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	}

	result, err := testQueries.GetRecentWorkspaceMessages(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Messages should be ordered by created_at DESC
	for i := 1; i < len(result); i++ {
		require.True(t, result[i-1].CreatedAt.After(result[i].CreatedAt) || result[i-1].CreatedAt.Equal(result[i].CreatedAt))
	}

	// Check that all messages are included
	messageIDs := make(map[int64]bool)
	for _, msg := range result {
		messageIDs[msg.ID] = true
		require.Equal(t, workspace.ID, msg.WorkspaceID)
	}

	require.True(t, messageIDs[channelMessage1.ID])
	require.True(t, messageIDs[channelMessage2.ID])
	require.True(t, messageIDs[directMessage.ID])
}

func TestCheckMessageAuthor(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	authorID, err := testQueries.CheckMessageAuthor(context.Background(), message.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, authorID)
}

func TestCheckMessageAuthorNotFound(t *testing.T) {
	_, err := testQueries.CheckMessageAuthor(context.Background(), 999999)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestMessageWithThreading(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	// Create parent message
	parentMessage := createRandomChannelMessage(t, workspace, channel, user)

	// Create reply message with thread_id
	arg := CreateChannelMessageParams{
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    user.ID,
		Content:     "Reply to parent message",
	}

	replyMessage, err := testQueries.CreateChannelMessage(context.Background(), arg)
	require.NoError(t, err)

	// Manually update with thread_id (since SQLC doesn't support optional fields in INSERT)
	// In a real application, you might have a separate query for threaded messages
	require.Equal(t, "channel", replyMessage.MessageType)
	require.False(t, replyMessage.ThreadID.Valid) // Should be NULL initially

	// Verify parent message exists
	require.NotZero(t, parentMessage.ID)

	// Note: To properly test threading, you would need additional queries
	// that support setting thread_id during creation
}

func TestMessageContentValidation(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	// Test with maximum allowed content length (4000 characters)
	longContent := util.RandomString(4000)
	arg := CreateChannelMessageParams{
		WorkspaceID: workspace.ID,
		ChannelID:   sql.NullInt64{Int64: channel.ID, Valid: true},
		SenderID:    user.ID,
		Content:     longContent,
	}

	message, err := testQueries.CreateChannelMessage(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, longContent, message.Content)

	// Test with content exceeding limit (should fail)
	tooLongContent := util.RandomString(4001)
	arg.Content = tooLongContent

	_, err = testQueries.CreateChannelMessage(context.Background(), arg)
	require.Error(t, err) // Should fail due to CHECK constraint
}

func TestDirectMessageBetweenSameUser(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	// Try to create a direct message where sender and receiver are the same
	arg := CreateDirectMessageParams{
		WorkspaceID: workspace.ID,
		SenderID:    user.ID,
		ReceiverID:  sql.NullInt64{Int64: user.ID, Valid: true}, // Same as sender
		Content:     "Message to myself",
	}

	_, err := testQueries.CreateDirectMessage(context.Background(), arg)
	require.Error(t, err) // Should fail due to trigger constraint
}
