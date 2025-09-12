package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomScheduledMessage(t *testing.T) ScheduledMessage {
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

	arg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      util.RandomString(100),
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour),
	}

	scheduledMsg, err := testQueries.CreateScheduledMessage(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, scheduledMsg)

	require.Equal(t, arg.UserID, scheduledMsg.UserID)
	require.Equal(t, arg.WorkspaceID, scheduledMsg.WorkspaceID)
	require.Equal(t, arg.ChannelID, scheduledMsg.ChannelID)
	require.Equal(t, arg.ReceiverID, scheduledMsg.ReceiverID)
	require.Equal(t, arg.ThreadID, scheduledMsg.ThreadID)
	require.Equal(t, arg.Content, scheduledMsg.Content)
	require.Equal(t, arg.ContentType, scheduledMsg.ContentType)
	require.WithinDuration(t, arg.ScheduledFor, scheduledMsg.ScheduledFor, time.Second)
	require.NotZero(t, scheduledMsg.ID)
	require.NotZero(t, scheduledMsg.CreatedAt)
	require.False(t, scheduledMsg.SentAt.Valid)
	require.False(t, scheduledMsg.CancelledAt.Valid)

	return scheduledMsg
}

func createRandomDirectScheduledMessage(t *testing.T) ScheduledMessage {
	user := createRandomUser(t)
	receiver := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)

	arg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{},
		ReceiverID:   sql.NullInt64{Int64: receiver.ID, Valid: true},
		ThreadID:     sql.NullInt64{},
		Content:      util.RandomString(100),
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour * 2),
	}

	scheduledMsg, err := testQueries.CreateScheduledMessage(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, scheduledMsg)

	return scheduledMsg
}

func TestCreateScheduledMessage(t *testing.T) {
	createRandomScheduledMessage(t)
}

func TestCreateDirectScheduledMessage(t *testing.T) {
	createRandomDirectScheduledMessage(t)
}

func TestGetScheduledMessage(t *testing.T) {
	scheduledMsg1 := createRandomScheduledMessage(t)

	scheduledMsg2, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     scheduledMsg1.ID,
		UserID: scheduledMsg1.UserID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, scheduledMsg2)

	require.Equal(t, scheduledMsg1.ID, scheduledMsg2.ID)
	require.Equal(t, scheduledMsg1.UserID, scheduledMsg2.UserID)
	require.Equal(t, scheduledMsg1.WorkspaceID, scheduledMsg2.WorkspaceID)
	require.Equal(t, scheduledMsg1.ChannelID, scheduledMsg2.ChannelID)
	require.Equal(t, scheduledMsg1.ReceiverID, scheduledMsg2.ReceiverID)
	require.Equal(t, scheduledMsg1.ThreadID, scheduledMsg2.ThreadID)
	require.Equal(t, scheduledMsg1.Content, scheduledMsg2.Content)
	require.Equal(t, scheduledMsg1.ContentType, scheduledMsg2.ContentType)
	require.WithinDuration(t, scheduledMsg1.ScheduledFor, scheduledMsg2.ScheduledFor, time.Second)
	require.WithinDuration(t, scheduledMsg1.CreatedAt, scheduledMsg2.CreatedAt, time.Second)
}

func TestGetScheduledMessageWrongUser(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)
	otherUser := createRandomUser(t)

	// Try to get scheduled message with wrong user ID
	_, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: otherUser.ID,
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetUserScheduledMessages(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	user := createRandomUserForWorkspace(t, workspace)

	// Create channel scheduled message
	channel := createRandomChannel(t, workspace, user)
	channelArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      util.RandomString(50),
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour),
	}
	channelMsg, err := testQueries.CreateScheduledMessage(context.Background(), channelArg)
	require.NoError(t, err)

	// Create direct message scheduled message
	receiver := createRandomUser(t)
	dmArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{},
		ReceiverID:   sql.NullInt64{Int64: receiver.ID, Valid: true},
		ThreadID:     sql.NullInt64{},
		Content:      util.RandomString(75),
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour * 2),
	}
	dmMsg, err := testQueries.CreateScheduledMessage(context.Background(), dmArg)
	require.NoError(t, err)

	// Create a sent message (should not be included)
	sentArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      util.RandomString(50),
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour * 3),
	}
	sentMsg, err := testQueries.CreateScheduledMessage(context.Background(), sentArg)
	require.NoError(t, err)

	// Mark it as sent
	err = testQueries.MarkScheduledMessageAsSent(context.Background(), sentMsg.ID)
	require.NoError(t, err)

	// Get user scheduled messages
	messages, err := testQueries.GetUserScheduledMessages(context.Background(), GetUserScheduledMessagesParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	})
	require.NoError(t, err)
	require.Len(t, messages, 2) // Only pending messages

	// Verify messages are ordered by scheduled_for ASC
	require.True(t, messages[0].ScheduledFor.Before(messages[1].ScheduledFor) ||
		messages[0].ScheduledFor.Equal(messages[1].ScheduledFor))

	// Check channel message details
	var foundChannelMsg, foundDmMsg bool
	for _, msg := range messages {
		if msg.ID == channelMsg.ID {
			foundChannelMsg = true
			require.Equal(t, channel.Name, msg.ChannelName.String)
			require.True(t, msg.ChannelName.Valid)
			require.False(t, msg.FirstName.Valid)
			require.False(t, msg.LastName.Valid)
		} else if msg.ID == dmMsg.ID {
			foundDmMsg = true
			require.False(t, msg.ChannelName.Valid)
			require.Equal(t, receiver.FirstName, msg.FirstName.String)
			require.Equal(t, receiver.LastName, msg.LastName.String)
			require.True(t, msg.FirstName.Valid)
			require.True(t, msg.LastName.Valid)
		}
	}
	require.True(t, foundChannelMsg)
	require.True(t, foundDmMsg)
}

func TestGetPendingScheduledMessages(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	user := createRandomUserForWorkspace(t, workspace)
	channel := createRandomChannel(t, workspace, user)

	// Create messages scheduled for different times
	pastTime := time.Now().Add(-time.Hour)
	nearPast := time.Now().Add(-time.Minute * 5) // 5 minutes ago (due)
	farFuture := time.Now().Add(time.Hour * 5)

	// Create past due message
	pastArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      "Past message",
		ContentType:  "text",
		ScheduledFor: pastTime,
	}
	pastMsg, err := testQueries.CreateScheduledMessage(context.Background(), pastArg)
	require.NoError(t, err)

	// Create near past message (also due)
	nearArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      "Near past message",
		ContentType:  "text",
		ScheduledFor: nearPast,
	}
	nearMsg, err := testQueries.CreateScheduledMessage(context.Background(), nearArg)
	require.NoError(t, err)

	// Create far future message
	farArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      "Far future message",
		ContentType:  "text",
		ScheduledFor: farFuture,
	}
	_, err = testQueries.CreateScheduledMessage(context.Background(), farArg)
	require.NoError(t, err)

	// First, let's verify our messages exist by checking them individually
	pastMsgCheck, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     pastMsg.ID,
		UserID: pastMsg.UserID,
	})
	require.NoError(t, err)
	require.Equal(t, "Past message", pastMsgCheck.Content)
	t.Logf("Past message scheduled for: %v, now: %v, is due: %v", pastMsgCheck.ScheduledFor, time.Now(), pastMsgCheck.ScheduledFor.Before(time.Now()) || pastMsgCheck.ScheduledFor.Equal(time.Now()))

	nearMsgCheck, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     nearMsg.ID,
		UserID: nearMsg.UserID,
	})
	require.NoError(t, err)
	require.Equal(t, "Near past message", nearMsgCheck.Content)
	t.Logf("Near message scheduled for: %v, now: %v, is due: %v", nearMsgCheck.ScheduledFor, time.Now(), nearMsgCheck.ScheduledFor.Before(time.Now()) || nearMsgCheck.ScheduledFor.Equal(time.Now()))

	// Get pending messages (only past and near past should be returned)
	pendingMessages, err := testQueries.GetPendingScheduledMessages(context.Background(), 10)
	require.NoError(t, err)

	// Debug: Let's also try to get all scheduled messages to see what's in the database
	allMessages, err := testQueries.GetUserScheduledMessages(context.Background(), GetUserScheduledMessagesParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
		Limit:       10,
	})
	require.NoError(t, err)
	t.Logf("User has %d scheduled messages total", len(allMessages))
	for i, msg := range allMessages {
		t.Logf("User message %d: ID=%d, Content=%s, ScheduledFor=%v, SentAt=%v, CancelledAt=%v",
			i, msg.ID, msg.Content, msg.ScheduledFor, msg.SentAt, msg.CancelledAt)
	}

	// The query should return at least our 2 due messages
	require.GreaterOrEqual(t, len(pendingMessages), 2)

	// Verify messages are ordered by scheduled_for ASC
	foundPast := false
	foundNear := false
	for _, msg := range pendingMessages {
		if msg.ID == pastMsg.ID {
			foundPast = true
			require.Equal(t, "Past message", msg.Content)
		} else if msg.ID == nearMsg.ID {
			foundNear = true
			require.Equal(t, "Near past message", msg.Content)
		}
		// Verify all returned messages are due
		require.True(t, msg.ScheduledFor.Before(time.Now()) || msg.ScheduledFor.Equal(time.Now()))
		require.False(t, msg.SentAt.Valid)
		require.False(t, msg.CancelledAt.Valid)
	}

	// If our messages are not found, let's check what the query actually returned
	if !foundPast || !foundNear {
		t.Logf("Query returned %d messages:", len(pendingMessages))
		for i, msg := range pendingMessages {
			t.Logf("  %d: ID=%d, Content=%s, ScheduledFor=%v", i, msg.ID, msg.Content, msg.ScheduledFor)
		}
	}

	require.True(t, foundPast, "Past message not found in results")
	require.True(t, foundNear, "Near past message not found in results")
}

func TestMarkScheduledMessageAsSent(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)

	// Mark as sent
	err := testQueries.MarkScheduledMessageAsSent(context.Background(), scheduledMsg.ID)
	require.NoError(t, err)

	// Verify it's marked as sent
	updatedMsg, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: scheduledMsg.UserID,
	})
	require.NoError(t, err)
	require.True(t, updatedMsg.SentAt.Valid)
	require.WithinDuration(t, time.Now(), updatedMsg.SentAt.Time, time.Second*5)
	require.False(t, updatedMsg.CancelledAt.Valid)
}

func TestCancelScheduledMessage(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)

	// Cancel the message
	err := testQueries.CancelScheduledMessage(context.Background(), CancelScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: scheduledMsg.UserID,
	})
	require.NoError(t, err)

	// Verify it's marked as cancelled
	updatedMsg, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: scheduledMsg.UserID,
	})
	require.NoError(t, err)
	require.False(t, updatedMsg.SentAt.Valid)
	require.True(t, updatedMsg.CancelledAt.Valid)
	require.WithinDuration(t, time.Now(), updatedMsg.CancelledAt.Time, time.Second*5)
}

func TestCancelScheduledMessageWrongUser(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)
	otherUser := createRandomUser(t)

	// Try to cancel with wrong user ID
	err := testQueries.CancelScheduledMessage(context.Background(), CancelScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: otherUser.ID,
	})
	require.NoError(t, err) // Should not error but should not cancel anything

	// Verify message is not cancelled
	updatedMsg, err := testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: scheduledMsg.UserID,
	})
	require.NoError(t, err)
	require.False(t, updatedMsg.CancelledAt.Valid)
}

func TestUpdateScheduledMessage(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)

	newContent := "Updated content"
	newContentType := "markdown"
	newScheduledFor := time.Now().Add(time.Hour * 3)

	updatedMsg, err := testQueries.UpdateScheduledMessage(context.Background(), UpdateScheduledMessageParams{
		ID:           scheduledMsg.ID,
		UserID:       scheduledMsg.UserID,
		Content:      newContent,
		ContentType:  newContentType,
		ScheduledFor: newScheduledFor,
	})
	require.NoError(t, err)
	require.NotEmpty(t, updatedMsg)

	require.Equal(t, scheduledMsg.ID, updatedMsg.ID)
	require.Equal(t, newContent, updatedMsg.Content)
	require.Equal(t, newContentType, updatedMsg.ContentType)
	require.WithinDuration(t, newScheduledFor, updatedMsg.ScheduledFor, time.Second)
}

func TestUpdateScheduledMessageAlreadySent(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)

	// Mark as sent first
	err := testQueries.MarkScheduledMessageAsSent(context.Background(), scheduledMsg.ID)
	require.NoError(t, err)

	// Try to update (should fail/not update since it's already sent)
	_, err = testQueries.UpdateScheduledMessage(context.Background(), UpdateScheduledMessageParams{
		ID:           scheduledMsg.ID,
		UserID:       scheduledMsg.UserID,
		Content:      "Should not update",
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour),
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeleteScheduledMessage(t *testing.T) {
	scheduledMsg := createRandomScheduledMessage(t)

	// Delete the message
	err := testQueries.DeleteScheduledMessage(context.Background(), DeleteScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: scheduledMsg.UserID,
	})
	require.NoError(t, err)

	// Verify it's deleted
	_, err = testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     scheduledMsg.ID,
		UserID: scheduledMsg.UserID,
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestCleanupOldScheduledMessages(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	user := createRandomUserForWorkspace(t, workspace)
	channel := createRandomChannel(t, workspace, user)

	// Create an old sent message
	oldSentArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      "Old sent message",
		ContentType:  "text",
		ScheduledFor: time.Now().Add(-time.Hour * 2),
	}
	oldSentMsg, err := testQueries.CreateScheduledMessage(context.Background(), oldSentArg)
	require.NoError(t, err)

	err = testQueries.MarkScheduledMessageAsSent(context.Background(), oldSentMsg.ID)
	require.NoError(t, err)

	// Create an old cancelled message
	oldCancelledArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      "Old cancelled message",
		ContentType:  "text",
		ScheduledFor: time.Now().Add(-time.Hour * 2),
	}
	oldCancelledMsg, err := testQueries.CreateScheduledMessage(context.Background(), oldCancelledArg)
	require.NoError(t, err)

	err = testQueries.CancelScheduledMessage(context.Background(), CancelScheduledMessageParams{
		ID:     oldCancelledMsg.ID,
		UserID: oldCancelledMsg.UserID,
	})
	require.NoError(t, err)

	// Create a pending message (should not be cleaned up)
	pendingMsg := createRandomScheduledMessage(t)

	// Cleanup old messages (older than 1 hour)
	// Clean up messages older than now (since we just created them)
	cutoffTime := time.Now()
	err = testQueries.CleanupOldScheduledMessages(context.Background(), cutoffTime)
	require.NoError(t, err)

	// Verify old messages are deleted
	_, err = testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     oldSentMsg.ID,
		UserID: oldSentMsg.UserID,
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	_, err = testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     oldCancelledMsg.ID,
		UserID: oldCancelledMsg.UserID,
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	// Verify pending message still exists
	_, err = testQueries.GetScheduledMessage(context.Background(), GetScheduledMessageParams{
		ID:     pendingMsg.ID,
		UserID: pendingMsg.UserID,
	})
	require.NoError(t, err)
}

func TestGetScheduledMessagesStats(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	user := createRandomUserForWorkspace(t, workspace)
	channel := createRandomChannel(t, workspace, user)

	// Create pending messages
	for i := 0; i < 3; i++ {
		arg := CreateScheduledMessageParams{
			UserID:       user.ID,
			WorkspaceID:  workspace.ID,
			ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
			ReceiverID:   sql.NullInt64{},
			ThreadID:     sql.NullInt64{},
			Content:      util.RandomString(50),
			ContentType:  "text",
			ScheduledFor: time.Now().Add(time.Hour * time.Duration(i+1)),
		}
		_, err := testQueries.CreateScheduledMessage(context.Background(), arg)
		require.NoError(t, err)
	}

	// Create sent messages
	for i := 0; i < 2; i++ {
		arg := CreateScheduledMessageParams{
			UserID:       user.ID,
			WorkspaceID:  workspace.ID,
			ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
			ReceiverID:   sql.NullInt64{},
			ThreadID:     sql.NullInt64{},
			Content:      util.RandomString(50),
			ContentType:  "text",
			ScheduledFor: time.Now().Add(-time.Hour * time.Duration(i+1)),
		}
		msg, err := testQueries.CreateScheduledMessage(context.Background(), arg)
		require.NoError(t, err)

		err = testQueries.MarkScheduledMessageAsSent(context.Background(), msg.ID)
		require.NoError(t, err)
	}

	// Create cancelled message
	cancelledArg := CreateScheduledMessageParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		ChannelID:    sql.NullInt64{Int64: channel.ID, Valid: true},
		ReceiverID:   sql.NullInt64{},
		ThreadID:     sql.NullInt64{},
		Content:      util.RandomString(50),
		ContentType:  "text",
		ScheduledFor: time.Now().Add(time.Hour * 10),
	}
	cancelledMsg, err := testQueries.CreateScheduledMessage(context.Background(), cancelledArg)
	require.NoError(t, err)

	err = testQueries.CancelScheduledMessage(context.Background(), CancelScheduledMessageParams{
		ID:     cancelledMsg.ID,
		UserID: cancelledMsg.UserID,
	})
	require.NoError(t, err)

	// Get stats
	stats, err := testQueries.GetScheduledMessagesStats(context.Background(), GetScheduledMessagesStatsParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
	})
	require.NoError(t, err)

	require.Equal(t, int64(3), stats.PendingCount)
	require.Equal(t, int64(2), stats.SentCount)
	require.Equal(t, int64(1), stats.CancelledCount)
}
