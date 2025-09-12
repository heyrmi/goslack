package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createRandomMessageReaction(t *testing.T) MessageReaction {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	reactingUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	emoji := "👍"

	arg := AddMessageReactionParams{
		MessageID: message.ID,
		UserID:    reactingUser.ID,
		Emoji:     emoji,
	}

	reaction, err := testQueries.AddMessageReaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, reaction)

	require.Equal(t, arg.MessageID, reaction.MessageID)
	require.Equal(t, arg.UserID, reaction.UserID)
	require.Equal(t, arg.Emoji, reaction.Emoji)
	require.NotZero(t, reaction.ID)
	require.NotZero(t, reaction.CreatedAt)

	return reaction
}

func TestAddMessageReaction(t *testing.T) {
	createRandomMessageReaction(t)
}

func TestAddMessageReactionDuplicate(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	reactingUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	emoji := "❤️"

	arg := AddMessageReactionParams{
		MessageID: message.ID,
		UserID:    reactingUser.ID,
		Emoji:     emoji,
	}

	// Add first reaction
	reaction1, err := testQueries.AddMessageReaction(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, reaction1)

	// Try to add the same reaction again (should be ignored due to ON CONFLICT DO NOTHING)
	_, err = testQueries.AddMessageReaction(context.Background(), arg)
	require.Error(t, err) // Should return sql.ErrNoRows due to ON CONFLICT DO NOTHING
	require.Equal(t, sql.ErrNoRows, err)

	// Verify only one reaction exists
	reactions, err := testQueries.GetMessageReactions(context.Background(), message.ID)
	require.NoError(t, err)

	emojiReactions := 0
	for _, r := range reactions {
		if r.Emoji == emoji && r.UserID == reactingUser.ID {
			emojiReactions++
		}
	}
	require.Equal(t, 1, emojiReactions, "Should have exactly one reaction of this type")
}

func TestRemoveMessageReaction(t *testing.T) {
	reaction := createRandomMessageReaction(t)

	// Remove the reaction
	err := testQueries.RemoveMessageReaction(context.Background(), RemoveMessageReactionParams{
		MessageID: reaction.MessageID,
		UserID:    reaction.UserID,
		Emoji:     reaction.Emoji,
	})
	require.NoError(t, err)

	// Verify reaction is removed
	reactions, err := testQueries.GetMessageReactions(context.Background(), reaction.MessageID)
	require.NoError(t, err)

	for _, r := range reactions {
		require.False(t, r.ID == reaction.ID, "Reaction should be removed")
	}
}

func TestGetMessageReactions(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	// Create multiple reactions from different users
	user1 := createRandomUserForOrganization(t, workspace.OrganizationID)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	user3 := createRandomUserForOrganization(t, workspace.OrganizationID)

	reactions := []AddMessageReactionParams{
		{MessageID: message.ID, UserID: user1.ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: user2.ID, Emoji: "❤️"},
		{MessageID: message.ID, UserID: user3.ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: user1.ID, Emoji: "😂"},
	}

	for _, reactionArg := range reactions {
		_, err := testQueries.AddMessageReaction(context.Background(), reactionArg)
		require.NoError(t, err)
	}

	// Get all reactions
	messageReactions, err := testQueries.GetMessageReactions(context.Background(), message.ID)
	require.NoError(t, err)
	require.Len(t, messageReactions, 4)

	// Verify user information is populated
	for _, reaction := range messageReactions {
		require.NotEmpty(t, reaction.FirstName)
		require.NotEmpty(t, reaction.LastName)
		require.Contains(t, []string{"👍", "❤️", "😂"}, reaction.Emoji)
	}

	// Verify reactions are ordered by creation time
	for i := 1; i < len(messageReactions); i++ {
		require.True(t, messageReactions[i].CreatedAt.After(messageReactions[i-1].CreatedAt) ||
			messageReactions[i].CreatedAt.Equal(messageReactions[i-1].CreatedAt))
	}
}

func TestGetMessageReactionCounts(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	// Create multiple reactions
	users := make([]User, 5)
	for i := 0; i < 5; i++ {
		users[i] = createRandomUserForOrganization(t, workspace.OrganizationID)
	}

	// Add reactions: 3x 👍, 2x ❤️, 1x 😂
	reactions := []AddMessageReactionParams{
		{MessageID: message.ID, UserID: users[0].ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: users[1].ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: users[2].ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: users[3].ID, Emoji: "❤️"},
		{MessageID: message.ID, UserID: users[4].ID, Emoji: "❤️"},
		{MessageID: message.ID, UserID: users[0].ID, Emoji: "😂"},
	}

	for _, reactionArg := range reactions {
		_, err := testQueries.AddMessageReaction(context.Background(), reactionArg)
		require.NoError(t, err)
	}

	// Get reaction counts
	counts, err := testQueries.GetMessageReactionCounts(context.Background(), message.ID)
	require.NoError(t, err)
	require.Len(t, counts, 3)

	// Verify counts are ordered by count DESC, emoji ASC
	expectedOrder := []struct {
		emoji string
		count int64
	}{
		{"👍", 3},
		{"❤️", 2},
		{"😂", 1},
	}

	for i, expected := range expectedOrder {
		require.Equal(t, expected.emoji, counts[i].Emoji)
		require.Equal(t, expected.count, counts[i].Count)
	}
}

func TestHasUserReacted(t *testing.T) {
	reaction := createRandomMessageReaction(t)

	// Check if user has reacted
	result, err := testQueries.HasUserReacted(context.Background(), HasUserReactedParams{
		MessageID: reaction.MessageID,
		UserID:    reaction.UserID,
		Emoji:     reaction.Emoji,
	})
	require.NoError(t, err)
	require.True(t, result)

	// Check with different emoji
	result, err = testQueries.HasUserReacted(context.Background(), HasUserReactedParams{
		MessageID: reaction.MessageID,
		UserID:    reaction.UserID,
		Emoji:     "❤️",
	})
	require.NoError(t, err)
	require.False(t, result)

	// Check with different user - we need to get the workspace from the reaction's context
	// For this test, we'll create a new user in the same organization
	workspace, _ := createTestWorkspaceAndUser(t)
	otherUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	result, err = testQueries.HasUserReacted(context.Background(), HasUserReactedParams{
		MessageID: reaction.MessageID,
		UserID:    otherUser.ID,
		Emoji:     reaction.Emoji,
	})
	require.NoError(t, err)
	require.False(t, result)
}

func TestGetUserReactionsForMessage(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	reactingUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	emojis := []string{"👍", "❤️", "😂"}

	// Add multiple reactions from the same user
	for _, emoji := range emojis {
		_, err := testQueries.AddMessageReaction(context.Background(), AddMessageReactionParams{
			MessageID: message.ID,
			UserID:    reactingUser.ID,
			Emoji:     emoji,
		})
		require.NoError(t, err)
	}

	// Add a reaction from another user (should not be included)
	otherUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	_, err := testQueries.AddMessageReaction(context.Background(), AddMessageReactionParams{
		MessageID: message.ID,
		UserID:    otherUser.ID,
		Emoji:     "🎉",
	})
	require.NoError(t, err)

	// Get user reactions for message
	userReactions, err := testQueries.GetUserReactionsForMessage(context.Background(), GetUserReactionsForMessageParams{
		MessageID: message.ID,
		UserID:    reactingUser.ID,
	})
	require.NoError(t, err)
	require.Len(t, userReactions, 3)

	// Verify all reactions are from the correct user and contain expected emojis
	foundEmojis := make(map[string]bool)
	for _, emoji := range userReactions {
		foundEmojis[emoji] = true
	}

	for _, emoji := range emojis {
		require.True(t, foundEmojis[emoji], "Expected emoji %s not found", emoji)
	}
	require.False(t, foundEmojis["🎉"], "Should not include reactions from other users")
}

func TestMessageReactionComplexScenario(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)
	message := createRandomChannelMessage(t, workspace, channel, user)

	// Create multiple users
	users := make([]User, 4)
	for i := 0; i < 4; i++ {
		users[i] = createRandomUserForOrganization(t, workspace.OrganizationID)
	}

	// Complex reaction scenario:
	// User 0: 👍, ❤️
	// User 1: 👍, 😂
	// User 2: ❤️
	// User 3: 👍, ❤️, 😂

	reactions := []AddMessageReactionParams{
		{MessageID: message.ID, UserID: users[0].ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: users[0].ID, Emoji: "❤️"},
		{MessageID: message.ID, UserID: users[1].ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: users[1].ID, Emoji: "😂"},
		{MessageID: message.ID, UserID: users[2].ID, Emoji: "❤️"},
		{MessageID: message.ID, UserID: users[3].ID, Emoji: "👍"},
		{MessageID: message.ID, UserID: users[3].ID, Emoji: "❤️"},
		{MessageID: message.ID, UserID: users[3].ID, Emoji: "😂"},
	}

	for _, reactionArg := range reactions {
		_, err := testQueries.AddMessageReaction(context.Background(), reactionArg)
		require.NoError(t, err)
	}

	// Test reaction counts
	counts, err := testQueries.GetMessageReactionCounts(context.Background(), message.ID)
	require.NoError(t, err)

	expectedCounts := map[string]int64{
		"👍":  3, // Users 0, 1, 3
		"❤️": 3, // Users 0, 2, 3
		"😂":  2, // Users 1, 3
	}

	for _, count := range counts {
		expected, exists := expectedCounts[count.Emoji]
		require.True(t, exists, "Unexpected emoji in counts: %s", count.Emoji)
		require.Equal(t, expected, count.Count, "Wrong count for emoji %s", count.Emoji)
	}

	// Test user-specific reactions
	user3Reactions, err := testQueries.GetUserReactionsForMessage(context.Background(), GetUserReactionsForMessageParams{
		MessageID: message.ID,
		UserID:    users[3].ID,
	})
	require.NoError(t, err)
	require.Len(t, user3Reactions, 3) // User 3 has all three emojis

	// Test has reacted
	hasReacted, err := testQueries.HasUserReacted(context.Background(), HasUserReactedParams{
		MessageID: message.ID,
		UserID:    users[2].ID,
		Emoji:     "❤️",
	})
	require.NoError(t, err)
	require.True(t, hasReacted)

	hasReacted, err = testQueries.HasUserReacted(context.Background(), HasUserReactedParams{
		MessageID: message.ID,
		UserID:    users[2].ID,
		Emoji:     "👍",
	})
	require.NoError(t, err)
	require.False(t, hasReacted)

	// Remove a reaction and verify
	err = testQueries.RemoveMessageReaction(context.Background(), RemoveMessageReactionParams{
		MessageID: message.ID,
		UserID:    users[3].ID,
		Emoji:     "👍",
	})
	require.NoError(t, err)

	// Verify count decreased
	counts, err = testQueries.GetMessageReactionCounts(context.Background(), message.ID)
	require.NoError(t, err)

	for _, count := range counts {
		if count.Emoji == "👍" {
			require.Equal(t, int64(2), count.Count) // Should be 2 now (users 0, 1)
		}
	}
}
