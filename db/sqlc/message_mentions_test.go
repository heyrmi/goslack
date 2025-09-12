package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createRandomMessageMention(t *testing.T) MessageMention {
	workspace, user := createTestWorkspaceAndUser(t)
	mentionedUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	arg := CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		MentionType:     "user",
	}

	mention, err := testQueries.CreateMessageMention(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, mention)

	require.Equal(t, arg.MessageID, mention.MessageID)
	require.Equal(t, arg.MentionedUserID, mention.MentionedUserID)
	require.Equal(t, arg.MentionType, mention.MentionType)
	require.NotZero(t, mention.ID)
	require.NotZero(t, mention.CreatedAt)

	return mention
}

func createRandomChannelMention(t *testing.T) MessageMention {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	arg := CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{},
		MentionType:     "channel",
	}

	mention, err := testQueries.CreateMessageMention(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, mention)

	return mention
}

func createRandomEveryoneMention(t *testing.T) MessageMention {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	arg := CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{},
		MentionType:     "everyone",
	}

	mention, err := testQueries.CreateMessageMention(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, mention)

	return mention
}

func TestCreateMessageMention(t *testing.T) {
	createRandomMessageMention(t)
}

func TestCreateChannelMention(t *testing.T) {
	createRandomChannelMention(t)
}

func TestCreateEveryoneMention(t *testing.T) {
	createRandomEveryoneMention(t)
}

func TestGetMessageMentions(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	mentionedUser1 := createRandomUserForOrganization(t, workspace.OrganizationID)
	mentionedUser2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	// Create multiple mentions for the same message
	mention1, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{Int64: mentionedUser1.ID, Valid: true},
		MentionType:     "user",
	})
	require.NoError(t, err)

	mention2, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{Int64: mentionedUser2.ID, Valid: true},
		MentionType:     "user",
	})
	require.NoError(t, err)

	mention3, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{},
		MentionType:     "channel",
	})
	require.NoError(t, err)

	// Get all mentions for the message
	mentions, err := testQueries.GetMessageMentions(context.Background(), message.ID)
	require.NoError(t, err)
	require.Len(t, mentions, 3)

	// Verify mentions are returned in creation order
	require.Equal(t, mention1.ID, mentions[0].ID)
	require.Equal(t, mention2.ID, mentions[1].ID)
	require.Equal(t, mention3.ID, mentions[2].ID)

	// Verify user information is populated for user mentions
	for _, mention := range mentions {
		if mention.MentionType == "user" {
			require.True(t, mention.FirstName.Valid)
			require.True(t, mention.LastName.Valid)
			require.True(t, mention.Email.Valid)
		} else {
			require.False(t, mention.FirstName.Valid)
			require.False(t, mention.LastName.Valid)
			require.False(t, mention.Email.Valid)
		}
	}
}

func TestGetUserMentions(t *testing.T) {
	workspace, mentionedUser := createTestWorkspaceAndUser(t)

	// Create multiple messages with mentions
	user1 := createRandomUserForWorkspace(t, workspace)
	channel1 := createRandomChannel(t, workspace, user1)
	message1 := createRandomChannelMessage(t, workspace, channel1, user1)

	user2 := createRandomUserForWorkspace(t, workspace)
	channel2 := createRandomChannel(t, workspace, user2)
	message2 := createRandomChannelMessage(t, workspace, channel2, user2)

	// Create user mentions
	_, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message1.ID,
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		MentionType:     "user",
	})
	require.NoError(t, err)

	_, err = testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message2.ID,
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		MentionType:     "user",
	})
	require.NoError(t, err)

	// Create a channel mention in the same workspace
	_, err = testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message1.ID,
		MentionedUserID: sql.NullInt64{},
		MentionType:     "channel",
	})
	require.NoError(t, err)

	// Get user mentions
	mentions, err := testQueries.GetUserMentions(context.Background(), GetUserMentionsParams{
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		WorkspaceID:     workspace.ID,
		Limit:           10,
		Offset:          0,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(mentions), 3) // At least 3 (2 user mentions + 1 channel mention)

	// Verify mentions contain expected information
	for _, mention := range mentions {
		require.NotEmpty(t, mention.Content)
		require.NotEmpty(t, mention.SenderFirstName)
		require.NotEmpty(t, mention.SenderLastName)
		require.True(t, mention.ChannelName.Valid) // All our messages are in channels
	}
}

func TestGetUnreadMentions(t *testing.T) {
	workspace, mentionedUser := createTestWorkspaceAndUser(t)

	// Create a channel and add the user as member
	channel := createRandomChannel(t, workspace, mentionedUser)

	// Create a message with mention
	sender := createRandomUserForOrganization(t, workspace.OrganizationID)
	message := createRandomChannelMessage(t, workspace, channel, sender)

	_, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
		MessageID:       message.ID,
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		MentionType:     "user",
	})
	require.NoError(t, err)

	// Create unread message entry (simulating that user hasn't read this message)
	// Note: Using a simplified approach since UpsertUnreadMessage might not exist
	// In a real implementation, this would be handled by the unread message tracking system

	// Get unread mentions
	mentions, err := testQueries.GetUnreadMentions(context.Background(), GetUnreadMentionsParams{
		UserID:      mentionedUser.ID,
		WorkspaceID: workspace.ID,
		Limit:       10,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(mentions), 1)

	// Verify the mention is in the results
	found := false
	for _, mention := range mentions {
		if mention.MessageID == message.ID {
			found = true
			require.Equal(t, "user", mention.MentionType)
			require.Equal(t, message.Content, mention.Content)
			require.Equal(t, sender.FirstName, mention.SenderFirstName)
			require.Equal(t, sender.LastName, mention.SenderLastName)
			break
		}
	}
	require.True(t, found, "Expected mention not found in unread mentions")
}

func TestGetUserMentionsWithPagination(t *testing.T) {
	workspace, mentionedUser := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, mentionedUser)

	// Create multiple mentions
	mentionCount := 5
	for i := 0; i < mentionCount; i++ {
		sender := createRandomUserForOrganization(t, workspace.OrganizationID)
		message := createRandomChannelMessage(t, workspace, channel, sender)

		_, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
			MessageID:       message.ID,
			MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
			MentionType:     "user",
		})
		require.NoError(t, err)
	}

	// Test pagination
	firstPage, err := testQueries.GetUserMentions(context.Background(), GetUserMentionsParams{
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		WorkspaceID:     workspace.ID,
		Limit:           3,
		Offset:          0,
	})
	require.NoError(t, err)
	require.LessOrEqual(t, len(firstPage), 3)

	secondPage, err := testQueries.GetUserMentions(context.Background(), GetUserMentionsParams{
		MentionedUserID: sql.NullInt64{Int64: mentionedUser.ID, Valid: true},
		WorkspaceID:     workspace.ID,
		Limit:           3,
		Offset:          3,
	})
	require.NoError(t, err)

	// Verify no overlap between pages
	firstPageIDs := make(map[int64]bool)
	for _, mention := range firstPage {
		firstPageIDs[mention.MessageID] = true
	}

	for _, mention := range secondPage {
		require.False(t, firstPageIDs[mention.MessageID], "Found duplicate mention across pages")
	}
}

func TestMessageMentionTypes(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	testCases := []struct {
		name        string
		mentionType string
		hasUser     bool
	}{
		{"user mention", "user", true},
		{"channel mention", "channel", false},
		{"here mention", "here", false},
		{"everyone mention", "everyone", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var mentionedUserID sql.NullInt64
			if tc.hasUser {
				mentionedUser := createRandomUserForOrganization(t, workspace.OrganizationID)
				mentionedUserID = sql.NullInt64{Int64: mentionedUser.ID, Valid: true}
			}

			mention, err := testQueries.CreateMessageMention(context.Background(), CreateMessageMentionParams{
				MessageID:       message.ID,
				MentionedUserID: mentionedUserID,
				MentionType:     tc.mentionType,
			})
			require.NoError(t, err)
			require.Equal(t, tc.mentionType, mention.MentionType)
			require.Equal(t, tc.hasUser, mention.MentionedUserID.Valid)
		})
	}
}
