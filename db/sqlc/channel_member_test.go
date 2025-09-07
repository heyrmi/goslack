package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func createRandomChannelMember(t *testing.T, channel Channel, user User, addedBy User) ChannelMember {
	arg := AddChannelMemberParams{
		ChannelID: channel.ID,
		UserID:    user.ID,
		AddedBy:   addedBy.ID,
		Role:      "member",
	}

	member, err := testQueries.AddChannelMember(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, member)

	require.Equal(t, arg.ChannelID, member.ChannelID)
	require.Equal(t, arg.UserID, member.UserID)
	require.Equal(t, arg.AddedBy, member.AddedBy)
	require.Equal(t, arg.Role, member.Role)

	require.NotZero(t, member.ID)
	require.NotZero(t, member.JoinedAt)

	return member
}

func TestAddChannelMember(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)

	createRandomChannelMember(t, channel, user2, user1)
}

func TestAddChannelMemberAdmin(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)

	arg := AddChannelMemberParams{
		ChannelID: channel.ID,
		UserID:    user2.ID,
		AddedBy:   user1.ID,
		Role:      "admin",
	}

	member, err := testQueries.AddChannelMember(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, member)

	require.Equal(t, arg.ChannelID, member.ChannelID)
	require.Equal(t, arg.UserID, member.UserID)
	require.Equal(t, arg.AddedBy, member.AddedBy)
	require.Equal(t, "admin", member.Role)
}

func TestRemoveChannelMember(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)
	member := createRandomChannelMember(t, channel, user2, user1)

	// Remove the member
	arg := RemoveChannelMemberParams{
		ChannelID: member.ChannelID,
		UserID:    member.UserID,
	}

	err := testQueries.RemoveChannelMember(context.Background(), arg)
	require.NoError(t, err)

	// Verify member is removed
	role, err := testQueries.CheckChannelMembership(context.Background(), CheckChannelMembershipParams{
		ChannelID: member.ChannelID,
		UserID:    member.UserID,
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
	require.Empty(t, role)
}

func TestGetChannelMembers(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	user3 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)

	// Add multiple members
	member1 := createRandomChannelMember(t, channel, user2, user1)
	member2 := createRandomChannelMember(t, channel, user3, user1)

	arg := GetChannelMembersParams{
		ChannelID: channel.ID,
		Limit:     10,
		Offset:    0,
	}

	members, err := testQueries.GetChannelMembers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, members, 2)

	// Check that both members are returned
	memberIDs := make(map[int64]bool)
	for _, member := range members {
		memberIDs[member.UserID] = true
		require.Equal(t, channel.ID, member.ChannelID)
		require.NotEmpty(t, member.FirstName)
		require.NotEmpty(t, member.LastName)
		require.NotEmpty(t, member.Email)
	}

	require.True(t, memberIDs[member1.UserID])
	require.True(t, memberIDs[member2.UserID])
}

func TestGetChannelMembersPagination(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	channel := createRandomChannel(t, workspace, user1)

	// Add 5 members
	var members []ChannelMember
	for i := 0; i < 5; i++ {
		user := createRandomUserForOrganization(t, workspace.OrganizationID)
		member := createRandomChannelMember(t, channel, user, user1)
		members = append(members, member)
	}

	// Test pagination - first page
	arg := GetChannelMembersParams{
		ChannelID: channel.ID,
		Limit:     2,
		Offset:    0,
	}

	page1, err := testQueries.GetChannelMembers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	// Test pagination - second page
	arg.Offset = 2
	page2, err := testQueries.GetChannelMembers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	// Test pagination - third page
	arg.Offset = 4
	page3, err := testQueries.GetChannelMembers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page3, 1)

	// Verify no overlap
	allMemberIDs := make(map[int64]bool)
	for _, member := range append(append(page1, page2...), page3...) {
		require.False(t, allMemberIDs[member.UserID], "Duplicate member found in pagination")
		allMemberIDs[member.UserID] = true
	}
}

func TestCheckChannelMembership(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)
	member := createRandomChannelMember(t, channel, user2, user1)

	arg := CheckChannelMembershipParams{
		ChannelID: member.ChannelID,
		UserID:    member.UserID,
	}

	role, err := testQueries.CheckChannelMembership(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, member.Role, role)
}

func TestCheckChannelMembershipNotFound(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)

	arg := CheckChannelMembershipParams{
		ChannelID: channel.ID,
		UserID:    user2.ID, // User is not a member
	}

	role, err := testQueries.CheckChannelMembership(context.Background(), arg)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
	require.Empty(t, role)
}

func TestGetUserChannels(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)

	// Create multiple channels and add user2 to some of them
	channel1 := createRandomChannel(t, workspace, user1)
	channel2 := createRandomChannel(t, workspace, user1)
	channel3 := createRandomChannel(t, workspace, user1)

	// Add user2 to channel1 and channel3
	createRandomChannelMember(t, channel1, user2, user1)
	createRandomChannelMember(t, channel3, user2, user1)

	arg := GetUserChannelsParams{
		UserID:      user2.ID,
		WorkspaceID: workspace.ID,
	}

	channels, err := testQueries.GetUserChannels(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, channels, 2)

	// Check that correct channels are returned
	channelIDs := make(map[int64]bool)
	for _, channel := range channels {
		channelIDs[channel.ID] = true
		require.Equal(t, workspace.ID, channel.WorkspaceID)
	}

	require.True(t, channelIDs[channel1.ID])
	require.True(t, channelIDs[channel3.ID])
	require.False(t, channelIDs[channel2.ID]) // User2 is not a member of channel2
}

func TestIsChannelMember(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)
	createRandomChannelMember(t, channel, user2, user1)

	arg := IsChannelMemberParams{
		ChannelID: channel.ID,
		UserID:    user2.ID,
	}

	isMember, err := testQueries.IsChannelMember(context.Background(), arg)
	require.NoError(t, err)
	require.True(t, isMember)
}

func TestIsChannelMemberFalse(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)

	arg := IsChannelMemberParams{
		ChannelID: channel.ID,
		UserID:    user2.ID, // User is not a member
	}

	isMember, err := testQueries.IsChannelMember(context.Background(), arg)
	require.NoError(t, err)
	require.False(t, isMember)
}

func TestAddChannelMemberDuplicate(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	channel := createRandomChannel(t, workspace, user1)

	// Add member first time
	createRandomChannelMember(t, channel, user2, user1)

	// Try to add the same member again
	arg := AddChannelMemberParams{
		ChannelID: channel.ID,
		UserID:    user2.ID,
		AddedBy:   user1.ID,
		Role:      "member",
	}

	_, err := testQueries.AddChannelMember(context.Background(), arg)
	require.Error(t, err) // Should fail due to unique constraint
}
