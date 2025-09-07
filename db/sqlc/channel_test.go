package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomChannel(t *testing.T, workspace Workspace, user User) Channel {
	arg := CreateChannelParams{
		WorkspaceID: workspace.ID,
		Name:        util.RandomString(10),
		IsPrivate:   util.RandomBool(),
		CreatedBy:   user.ID,
	}

	channel, err := testQueries.CreateChannel(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, channel)

	require.Equal(t, arg.WorkspaceID, channel.WorkspaceID)
	require.Equal(t, arg.Name, channel.Name)
	require.Equal(t, arg.IsPrivate, channel.IsPrivate)
	require.Equal(t, arg.CreatedBy, channel.CreatedBy)

	require.NotZero(t, channel.ID)
	require.NotZero(t, channel.CreatedAt)

	return channel
}

func createTestWorkspaceAndUser(t *testing.T) (Workspace, User) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	user := createRandomUserForOrganization(t, organization.ID)

	// Assign user to workspace
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	return workspace, user
}

func TestCreateChannel(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	createRandomChannel(t, workspace, user)
}

func TestGetChannel(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel1 := createRandomChannel(t, workspace, user)

	channel2, err := testQueries.GetChannel(context.Background(), channel1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, channel2)

	require.Equal(t, channel1.ID, channel2.ID)
	require.Equal(t, channel1.WorkspaceID, channel2.WorkspaceID)
	require.Equal(t, channel1.Name, channel2.Name)
	require.Equal(t, channel1.IsPrivate, channel2.IsPrivate)
	require.Equal(t, channel1.CreatedBy, channel2.CreatedBy)
	require.WithinDuration(t, channel1.CreatedAt, channel2.CreatedAt, time.Second)
}

func TestGetChannelByID(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel1 := createRandomChannel(t, workspace, user)

	channel2, err := testQueries.GetChannelByID(context.Background(), channel1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, channel2)

	require.Equal(t, channel1.ID, channel2.ID)
	require.Equal(t, channel1.WorkspaceID, channel2.WorkspaceID)
	require.Equal(t, channel1.Name, channel2.Name)
	require.Equal(t, channel1.IsPrivate, channel2.IsPrivate)
	require.Equal(t, channel1.CreatedBy, channel2.CreatedBy)
	require.WithinDuration(t, channel1.CreatedAt, channel2.CreatedAt, time.Second)
}

func TestUpdateChannel(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel1 := createRandomChannel(t, workspace, user)

	arg := UpdateChannelParams{
		ID:        channel1.ID,
		Name:      util.RandomString(10),
		IsPrivate: !channel1.IsPrivate, // Toggle privacy
	}

	channel2, err := testQueries.UpdateChannel(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, channel2)

	require.Equal(t, channel1.ID, channel2.ID)
	require.Equal(t, channel1.WorkspaceID, channel2.WorkspaceID)
	require.Equal(t, arg.Name, channel2.Name)
	require.Equal(t, arg.IsPrivate, channel2.IsPrivate)
	require.Equal(t, channel1.CreatedBy, channel2.CreatedBy)
	require.WithinDuration(t, channel1.CreatedAt, channel2.CreatedAt, time.Second)
}

func TestDeleteChannel(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel1 := createRandomChannel(t, workspace, user)

	err := testQueries.DeleteChannel(context.Background(), channel1.ID)
	require.NoError(t, err)

	channel2, err := testQueries.GetChannel(context.Background(), channel1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, channel2)
}

func TestListChannelsByWorkspace(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	// Create multiple channels
	channels := make([]Channel, 0)
	for i := 0; i < 10; i++ {
		channel := createRandomChannel(t, workspace, user)
		channels = append(channels, channel)
	}

	arg := ListChannelsByWorkspaceParams{
		WorkspaceID: workspace.ID,
		Limit:       5,
		Offset:      0,
	}

	retrievedChannels, err := testQueries.ListChannelsByWorkspace(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, retrievedChannels, 5)

	for _, channel := range retrievedChannels {
		require.NotEmpty(t, channel)
		require.Equal(t, workspace.ID, channel.WorkspaceID)
	}
}

func TestListPublicChannelsByWorkspace(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	// Create mix of public and private channels
	publicCount := 0
	privateCount := 0
	for i := 0; i < 10; i++ {
		arg := CreateChannelParams{
			WorkspaceID: workspace.ID,
			Name:        util.RandomString(10),
			IsPrivate:   i%2 == 0, // Alternating public/private
			CreatedBy:   user.ID,
		}

		_, err := testQueries.CreateChannel(context.Background(), arg)
		require.NoError(t, err)

		if arg.IsPrivate {
			privateCount++
		} else {
			publicCount++
		}
	}

	arg := ListPublicChannelsByWorkspaceParams{
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	}

	publicChannels, err := testQueries.ListPublicChannelsByWorkspace(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, publicChannels, publicCount)

	for _, channel := range publicChannels {
		require.NotEmpty(t, channel)
		require.Equal(t, workspace.ID, channel.WorkspaceID)
		require.False(t, channel.IsPrivate) // All should be public
	}
}

func TestListChannelsByWorkspaceEmpty(t *testing.T) {
	workspace, _ := createTestWorkspaceAndUser(t)

	arg := ListChannelsByWorkspaceParams{
		WorkspaceID: workspace.ID,
		Limit:       5,
		Offset:      0,
	}

	channels, err := testQueries.ListChannelsByWorkspace(context.Background(), arg)
	require.NoError(t, err)
	require.Empty(t, channels)
}

func TestGetChannelWithCreator(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	result, err := testQueries.GetChannelWithCreator(context.Background(), channel.ID)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, channel.ID, result.ID)
	require.Equal(t, channel.WorkspaceID, result.WorkspaceID)
	require.Equal(t, channel.Name, result.Name)
	require.Equal(t, channel.IsPrivate, result.IsPrivate)
	require.Equal(t, channel.CreatedBy, result.CreatedBy)
	require.WithinDuration(t, channel.CreatedAt, result.CreatedAt, time.Second)

	// Check creator information
	require.Equal(t, user.FirstName, result.CreatorFirstName)
	require.Equal(t, user.LastName, result.CreatorLastName)
	require.Equal(t, user.Email, result.CreatorEmail)
}

func TestGetChannelNotFound(t *testing.T) {
	channel, err := testQueries.GetChannel(context.Background(), 999999)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, channel)
}

func TestUpdateChannelNotFound(t *testing.T) {
	arg := UpdateChannelParams{
		ID:        999999,
		Name:      util.RandomString(10),
		IsPrivate: false,
	}

	channel, err := testQueries.UpdateChannel(context.Background(), arg)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, channel)
}

func TestCreateChannelWithInvalidWorkspace(t *testing.T) {
	_, user := createTestWorkspaceAndUser(t)

	arg := CreateChannelParams{
		WorkspaceID: 999999, // Non-existent workspace
		Name:        util.RandomString(10),
		IsPrivate:   false,
		CreatedBy:   user.ID,
	}

	channel, err := testQueries.CreateChannel(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, channel)
}

func TestCreateChannelWithInvalidUser(t *testing.T) {
	workspace, _ := createTestWorkspaceAndUser(t)

	arg := CreateChannelParams{
		WorkspaceID: workspace.ID,
		Name:        util.RandomString(10),
		IsPrivate:   false,
		CreatedBy:   999999, // Non-existent user
	}

	channel, err := testQueries.CreateChannel(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, channel)
}

func TestCreateChannelWithDuplicateName(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel1 := createRandomChannel(t, workspace, user)

	// Try to create another channel with the same name in the same workspace
	arg := CreateChannelParams{
		WorkspaceID: workspace.ID,
		Name:        channel1.Name,
		IsPrivate:   false,
		CreatedBy:   user.ID,
	}

	channel2, err := testQueries.CreateChannel(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, channel2)
}

func TestCreateChannelWithSameNameDifferentWorkspace(t *testing.T) {
	// This should be allowed - same channel name in different workspaces
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)
	workspace2 := createRandomWorkspace(t, organization)
	user := createRandomUserForOrganization(t, organization.ID)

	// Assign user to both workspaces
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace1.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	channelName := util.RandomString(10)

	arg1 := CreateChannelParams{
		WorkspaceID: workspace1.ID,
		Name:        channelName,
		IsPrivate:   false,
		CreatedBy:   user.ID,
	}

	channel1, err := testQueries.CreateChannel(context.Background(), arg1)
	require.NoError(t, err)
	require.NotEmpty(t, channel1)

	// Update user to workspace2 to create channel there
	_, err = testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace2.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	arg2 := CreateChannelParams{
		WorkspaceID: workspace2.ID,
		Name:        channelName,
		IsPrivate:   false,
		CreatedBy:   user.ID,
	}

	channel2, err := testQueries.CreateChannel(context.Background(), arg2)
	require.NoError(t, err)
	require.NotEmpty(t, channel2)

	require.Equal(t, channelName, channel1.Name)
	require.Equal(t, channelName, channel2.Name)
	require.NotEqual(t, channel1.WorkspaceID, channel2.WorkspaceID)
}

func TestChannelConstraints(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)

	// Create a user NOT in the workspace
	user := createRandomUserForOrganization(t, organization.ID)
	// Don't assign user to workspace

	// This should fail due to the trigger constraint
	arg := CreateChannelParams{
		WorkspaceID: workspace.ID,
		Name:        util.RandomString(10),
		IsPrivate:   false,
		CreatedBy:   user.ID,
	}

	channel, err := testQueries.CreateChannel(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, channel)
	require.Contains(t, err.Error(), "Channel creator must be a member of the workspace")
}

func TestChannelCascadeDelete(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user)

	// Delete the workspace
	err := testQueries.DeleteWorkspace(context.Background(), workspace.ID)
	require.NoError(t, err)

	// Channel should be deleted due to cascade
	deletedChannel, err := testQueries.GetChannel(context.Background(), channel.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, deletedChannel)
}
