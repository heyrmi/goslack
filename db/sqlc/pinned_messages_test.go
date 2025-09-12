package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createRandomPinnedMessage(t *testing.T) PinnedMessage {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channel
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	pinningUser := createRandomUser(t)

	arg := PinMessageParams{
		MessageID: message.ID,
		ChannelID: channel.ID,
		PinnedBy:  pinningUser.ID,
	}

	pinnedMessage, err := testQueries.PinMessage(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, pinnedMessage)

	require.Equal(t, arg.MessageID, pinnedMessage.MessageID)
	require.Equal(t, arg.ChannelID, pinnedMessage.ChannelID)
	require.Equal(t, arg.PinnedBy, pinnedMessage.PinnedBy)
	require.NotZero(t, pinnedMessage.ID)
	require.NotZero(t, pinnedMessage.PinnedAt)

	return pinnedMessage
}

func TestPinMessage(t *testing.T) {
	createRandomPinnedMessage(t)
}

func TestPinMessageDuplicate(t *testing.T) {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channel
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	pinningUser := createRandomUser(t)

	arg := PinMessageParams{
		MessageID: message.ID,
		ChannelID: channel.ID,
		PinnedBy:  pinningUser.ID,
	}

	// Pin the message first time
	pinnedMessage1, err := testQueries.PinMessage(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, pinnedMessage1)

	// Try to pin the same message again (should fail due to unique constraint)
	_, err = testQueries.PinMessage(context.Background(), arg)
	require.Error(t, err)
	// The error should be related to unique constraint violation
	require.Contains(t, err.Error(), "duplicate")
}

func TestUnpinMessage(t *testing.T) {
	pinnedMessage := createRandomPinnedMessage(t)

	// Unpin the message
	err := testQueries.UnpinMessage(context.Background(), pinnedMessage.MessageID)
	require.NoError(t, err)

	// Verify the message is no longer pinned
	isPinned, err := testQueries.IsMessagePinned(context.Background(), pinnedMessage.MessageID)
	require.NoError(t, err)
	require.False(t, isPinned)
}

func TestGetPinnedMessages(t *testing.T) {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channel
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel := createRandomChannel(t, workspace, user)

	// Create multiple messages and pin them
	var pinnedMessages []PinnedMessage
	for i := 0; i < 3; i++ {
		message := createRandomChannelMessage(t, workspace, channel, user)
		pinningUser := createRandomUser(t)

		pinnedMessage, err := testQueries.PinMessage(context.Background(), PinMessageParams{
			MessageID: message.ID,
			ChannelID: channel.ID,
			PinnedBy:  pinningUser.ID,
		})
		require.NoError(t, err)
		pinnedMessages = append(pinnedMessages, pinnedMessage)

		// Small delay to ensure different pinned_at times
		time.Sleep(time.Millisecond * 10)
	}

	// Get pinned messages for the channel
	result, err := testQueries.GetPinnedMessages(context.Background(), channel.ID)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Verify messages are ordered by pinned_at DESC (most recent first)
	for i := 1; i < len(result); i++ {
		require.True(t, result[i].PinnedAt.Before(result[i-1].PinnedAt) ||
			result[i].PinnedAt.Equal(result[i-1].PinnedAt))
	}

	// Verify all required fields are populated
	for _, pinned := range result {
		require.NotEmpty(t, pinned.Content)
		require.NotZero(t, pinned.MessageCreatedAt)
		require.NotEmpty(t, pinned.SenderFirstName)
		require.NotEmpty(t, pinned.SenderLastName)
		require.NotEmpty(t, pinned.PinnedByFirstName)
		require.NotEmpty(t, pinned.PinnedByLastName)
	}
}

func TestGetPinnedMessagesEmptyChannel(t *testing.T) {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channel
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel := createRandomChannel(t, workspace, user)

	// Get pinned messages for channel with no pinned messages
	result, err := testQueries.GetPinnedMessages(context.Background(), channel.ID)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestIsMessagePinned(t *testing.T) {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channel
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	// Initially should not be pinned
	isPinned, err := testQueries.IsMessagePinned(context.Background(), message.ID)
	require.NoError(t, err)
	require.False(t, isPinned)

	// Pin the message
	pinningUser := createRandomUser(t)
	_, err = testQueries.PinMessage(context.Background(), PinMessageParams{
		MessageID: message.ID,
		ChannelID: channel.ID,
		PinnedBy:  pinningUser.ID,
	})
	require.NoError(t, err)

	// Should now be pinned
	isPinned, err = testQueries.IsMessagePinned(context.Background(), message.ID)
	require.NoError(t, err)
	require.True(t, isPinned)

	// Unpin the message
	err = testQueries.UnpinMessage(context.Background(), message.ID)
	require.NoError(t, err)

	// Should no longer be pinned
	isPinned, err = testQueries.IsMessagePinned(context.Background(), message.ID)
	require.NoError(t, err)
	require.False(t, isPinned)
}

func TestPinnedMessagesWithDifferentUsers(t *testing.T) {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channel
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel := createRandomChannel(t, workspace, user)

	// Create messages from different users
	sender1 := createRandomUser(t)
	sender2 := createRandomUser(t)
	pinner1 := createRandomUser(t)
	pinner2 := createRandomUser(t)

	message1 := createRandomChannelMessage(t, workspace, channel, sender1)
	message2 := createRandomChannelMessage(t, workspace, channel, sender2)

	// Pin messages by different users
	_, err = testQueries.PinMessage(context.Background(), PinMessageParams{
		MessageID: message1.ID,
		ChannelID: channel.ID,
		PinnedBy:  pinner1.ID,
	})
	require.NoError(t, err)

	_, err = testQueries.PinMessage(context.Background(), PinMessageParams{
		MessageID: message2.ID,
		ChannelID: channel.ID,
		PinnedBy:  pinner2.ID,
	})
	require.NoError(t, err)

	// Get pinned messages
	pinnedMessages, err := testQueries.GetPinnedMessages(context.Background(), channel.ID)
	require.NoError(t, err)
	require.Len(t, pinnedMessages, 2)

	// Verify different senders and pinners are correctly populated
	senderNames := make(map[string]bool)
	pinnerNames := make(map[string]bool)

	for _, pinned := range pinnedMessages {
		senderName := pinned.SenderFirstName + " " + pinned.SenderLastName
		pinnerName := pinned.PinnedByFirstName + " " + pinned.PinnedByLastName
		senderNames[senderName] = true
		pinnerNames[pinnerName] = true
	}

	require.Len(t, senderNames, 2, "Should have messages from 2 different senders")
	require.Len(t, pinnerNames, 2, "Should have messages pinned by 2 different users")
}

func TestPinnedMessagesMultipleChannels(t *testing.T) {
	organization := createRandomOrganization(t)
	user := createRandomUserForOrganization(t, organization.ID)
	workspace := createRandomWorkspace(t, organization)

	// Assign user to workspace before creating channels
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channel1 := createRandomChannel(t, workspace, user)
	channel2 := createRandomChannel(t, workspace, user)

	message1 := createRandomChannelMessage(t, workspace, channel1, user)
	message2 := createRandomChannelMessage(t, workspace, channel2, user)

	pinningUser := createRandomUser(t)

	// Pin messages in different channels
	_, err = testQueries.PinMessage(context.Background(), PinMessageParams{
		MessageID: message1.ID,
		ChannelID: channel1.ID,
		PinnedBy:  pinningUser.ID,
	})
	require.NoError(t, err)

	_, err = testQueries.PinMessage(context.Background(), PinMessageParams{
		MessageID: message2.ID,
		ChannelID: channel2.ID,
		PinnedBy:  pinningUser.ID,
	})
	require.NoError(t, err)

	// Get pinned messages for channel 1
	pinnedChannel1, err := testQueries.GetPinnedMessages(context.Background(), channel1.ID)
	require.NoError(t, err)
	require.Len(t, pinnedChannel1, 1)
	require.Equal(t, message1.ID, pinnedChannel1[0].MessageID)

	// Get pinned messages for channel 2
	pinnedChannel2, err := testQueries.GetPinnedMessages(context.Background(), channel2.ID)
	require.NoError(t, err)
	require.Len(t, pinnedChannel2, 1)
	require.Equal(t, message2.ID, pinnedChannel2[0].MessageID)
}

func TestUnpinNonExistentMessage(t *testing.T) {
	// Try to unpin a message that doesn't exist or isn't pinned
	err := testQueries.UnpinMessage(context.Background(), 99999)
	require.NoError(t, err) // Should not error even if message doesn't exist
}

func TestPinMessageComplexScenario(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	user := createRandomUserForWorkspace(t, workspace)
	channel := createRandomChannel(t, workspace, user)

	// Create multiple messages
	messages := make([]Message, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createRandomChannelMessage(t, workspace, channel, user)
		time.Sleep(time.Millisecond * 10) // Ensure different creation times
	}

	pinningUser := createRandomUser(t)

	// Pin messages 1, 3, and 5 (odd indices)
	var pinnedMessageIDs []int64
	for i := 0; i < 5; i += 2 {
		_, err := testQueries.PinMessage(context.Background(), PinMessageParams{
			MessageID: messages[i].ID,
			ChannelID: channel.ID,
			PinnedBy:  pinningUser.ID,
		})
		require.NoError(t, err)
		pinnedMessageIDs = append(pinnedMessageIDs, messages[i].ID)
		time.Sleep(time.Millisecond * 10) // Ensure different pinned_at times
	}

	// Get all pinned messages
	pinnedMessages, err := testQueries.GetPinnedMessages(context.Background(), channel.ID)
	require.NoError(t, err)
	require.Len(t, pinnedMessages, 3)

	// Verify only the correct messages are pinned
	pinnedIDs := make(map[int64]bool)
	for _, pinned := range pinnedMessages {
		pinnedIDs[pinned.MessageID] = true
	}

	for _, expectedID := range pinnedMessageIDs {
		require.True(t, pinnedIDs[expectedID], "Expected message %d to be pinned", expectedID)
	}

	// Verify non-pinned messages are not in the results
	for i := 1; i < 5; i += 2 {
		require.False(t, pinnedIDs[messages[i].ID], "Message %d should not be pinned", messages[i].ID)
	}

	// Unpin the middle message
	middleMessageID := pinnedMessageIDs[1]
	err = testQueries.UnpinMessage(context.Background(), middleMessageID)
	require.NoError(t, err)

	// Verify only 2 messages remain pinned
	remainingPinned, err := testQueries.GetPinnedMessages(context.Background(), channel.ID)
	require.NoError(t, err)
	require.Len(t, remainingPinned, 2)

	// Verify the unpinned message is not in the results
	for _, pinned := range remainingPinned {
		require.NotEqual(t, middleMessageID, pinned.MessageID, "Unpinned message should not appear in results")
	}
}
