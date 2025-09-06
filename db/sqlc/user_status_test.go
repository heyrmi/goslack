package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomUserStatus(t *testing.T, user User, workspace Workspace) UserStatus {
	statuses := []string{"online", "away", "busy", "offline"}
	status := statuses[util.RandomInt(0, int64(len(statuses)-1))]
	customStatus := util.RandomString(20)

	arg := UpsertUserStatusParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		Status:       status,
		CustomStatus: sql.NullString{String: customStatus, Valid: true},
	}

	userStatus, err := testQueries.UpsertUserStatus(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, userStatus)

	require.Equal(t, arg.UserID, userStatus.UserID)
	require.Equal(t, arg.WorkspaceID, userStatus.WorkspaceID)
	require.Equal(t, arg.Status, userStatus.Status)
	require.Equal(t, arg.CustomStatus, userStatus.CustomStatus)

	require.NotZero(t, userStatus.LastActivityAt)
	require.NotZero(t, userStatus.LastSeenAt)
	require.NotZero(t, userStatus.UpdatedAt)

	return userStatus
}

func TestUpsertUserStatus(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	createRandomUserStatus(t, user, workspace)
}

func TestUpsertUserStatusUpdate(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	// Create initial status
	initialStatus := createRandomUserStatus(t, user, workspace)

	// Update the status
	newStatus := "busy"
	newCustomStatus := "In a meeting"
	arg := UpsertUserStatusParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		Status:       newStatus,
		CustomStatus: sql.NullString{String: newCustomStatus, Valid: true},
	}

	updatedStatus, err := testQueries.UpsertUserStatus(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, updatedStatus)

	require.Equal(t, user.ID, updatedStatus.UserID)
	require.Equal(t, workspace.ID, updatedStatus.WorkspaceID)
	require.Equal(t, newStatus, updatedStatus.Status)
	require.Equal(t, newCustomStatus, updatedStatus.CustomStatus.String)

	// Updated timestamp should be after initial
	require.True(t, updatedStatus.UpdatedAt.After(initialStatus.UpdatedAt))

	// Should still be the same record (upsert, not insert)
	require.Equal(t, initialStatus.UserID, updatedStatus.UserID)
}

func TestUpsertUserStatusWithoutCustomStatus(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	arg := UpsertUserStatusParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		Status:       "online",
		CustomStatus: sql.NullString{Valid: false}, // No custom status
	}

	userStatus, err := testQueries.UpsertUserStatus(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, userStatus)

	require.Equal(t, "online", userStatus.Status)
	require.False(t, userStatus.CustomStatus.Valid)
}

func TestGetUserStatus(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	status1 := createRandomUserStatus(t, user, workspace)

	arg := GetUserStatusParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
	}

	status2, err := testQueries.GetUserStatus(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, status2)

	require.Equal(t, status1.UserID, status2.UserID)
	require.Equal(t, status1.WorkspaceID, status2.WorkspaceID)
	require.Equal(t, status1.Status, status2.Status)
	require.Equal(t, status1.CustomStatus, status2.CustomStatus)
	require.WithinDuration(t, status1.UpdatedAt, status2.UpdatedAt, time.Second)
}

func TestGetUserStatusNotFound(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	arg := GetUserStatusParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
	}

	_, err := testQueries.GetUserStatus(context.Background(), arg)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetWorkspaceUserStatuses(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	user3 := createRandomUserForOrganization(t, workspace.OrganizationID)

	// Assign users to workspace
	testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user2.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user3.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})

	// Create statuses for multiple users
	status1 := createRandomUserStatus(t, user1, workspace)
	createRandomUserStatus(t, user2, workspace)
	createRandomUserStatus(t, user3, workspace)

	arg := GetWorkspaceUserStatusesParams{
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	}

	statuses, err := testQueries.GetWorkspaceUserStatuses(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, statuses, 3)

	// Check that all statuses are included with user information
	statusMap := make(map[int64]GetWorkspaceUserStatusesRow)
	for _, status := range statuses {
		statusMap[status.UserID] = status
		require.Equal(t, workspace.ID, status.WorkspaceID)
		require.NotEmpty(t, status.FirstName)
		require.NotEmpty(t, status.LastName)
		require.NotEmpty(t, status.Email)
	}

	// Verify all users are included
	require.Contains(t, statusMap, status1.UserID)
	require.Contains(t, statusMap, user2.ID)
	require.Contains(t, statusMap, user3.ID)

	// Check user information matches
	require.Equal(t, user1.FirstName, statusMap[user1.ID].FirstName)
	require.Equal(t, user1.LastName, statusMap[user1.ID].LastName)
	require.Equal(t, user1.Email, statusMap[user1.ID].Email)
}

func TestGetWorkspaceUserStatusesPagination(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)

	// Create statuses for 5 users
	var users []User
	users = append(users, user1)
	for i := 0; i < 4; i++ {
		user := createRandomUserForOrganization(t, workspace.OrganizationID)

		// Assign user to workspace
		testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
			ID:          user.ID,
			WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
			Role:        "member",
		})

		users = append(users, user)
		createRandomUserStatus(t, user, workspace)
	}
	createRandomUserStatus(t, user1, workspace)

	// Test first page
	arg := GetWorkspaceUserStatusesParams{
		WorkspaceID: workspace.ID,
		Limit:       2,
		Offset:      0,
	}

	page1, err := testQueries.GetWorkspaceUserStatuses(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	// Test second page
	arg.Offset = 2
	page2, err := testQueries.GetWorkspaceUserStatuses(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	// Test third page
	arg.Offset = 4
	page3, err := testQueries.GetWorkspaceUserStatuses(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, page3, 1)

	// Verify no duplicates
	allUserIDs := make(map[int64]bool)
	for _, status := range append(append(page1, page2...), page3...) {
		require.False(t, allUserIDs[status.UserID], "Duplicate user found in pagination")
		allUserIDs[status.UserID] = true
	}
}

func TestUpdateLastActivity(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)
	initialStatus := createRandomUserStatus(t, user, workspace)

	// Wait a bit to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	arg := UpdateLastActivityParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
	}

	err := testQueries.UpdateLastActivity(context.Background(), arg)
	require.NoError(t, err)

	// Check that activity was updated
	getArg := GetUserStatusParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
	}

	updatedStatus, err := testQueries.GetUserStatus(context.Background(), getArg)
	require.NoError(t, err)

	// Activity timestamps should be updated
	require.True(t, updatedStatus.LastActivityAt.After(initialStatus.LastActivityAt))
	require.True(t, updatedStatus.LastSeenAt.After(initialStatus.LastSeenAt))
	require.True(t, updatedStatus.UpdatedAt.After(initialStatus.UpdatedAt))

	// Other fields should remain the same
	require.Equal(t, initialStatus.Status, updatedStatus.Status)
	require.Equal(t, initialStatus.CustomStatus, updatedStatus.CustomStatus)
}

func TestUpdateLastActivityNonExistentUser(t *testing.T) {
	workspace, _ := createTestWorkspaceAndUser(t)

	arg := UpdateLastActivityParams{
		UserID:      999999, // Non-existent user
		WorkspaceID: workspace.ID,
	}

	err := testQueries.UpdateLastActivity(context.Background(), arg)
	require.NoError(t, err) // UPDATE with no matching rows should not error
}

func TestSetUsersOfflineAfterInactivity(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)

	// Assign user2 to workspace
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user2.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	// Set both to online initially
	_, err = testQueries.UpsertUserStatus(context.Background(), UpsertUserStatusParams{
		UserID:      user1.ID,
		WorkspaceID: workspace.ID,
		Status:      "online",
	})
	require.NoError(t, err)

	_, err = testQueries.UpsertUserStatus(context.Background(), UpsertUserStatusParams{
		UserID:      user2.ID,
		WorkspaceID: workspace.ID,
		Status:      "online",
	})
	require.NoError(t, err)

	// Set inactivity threshold to past time (no users should be affected because their activity is recent)
	pastTime := time.Now().UTC().Add(-1 * time.Hour) // 1 hour ago
	err = testQueries.SetUsersOfflineAfterInactivity(context.Background(), pastTime)
	require.NoError(t, err)

	// Both users should still be online (their activity is more recent than 1 hour ago)
	status1After, err := testQueries.GetUserStatus(context.Background(), GetUserStatusParams{
		UserID: user1.ID, WorkspaceID: workspace.ID,
	})
	require.NoError(t, err)
	status2After, err := testQueries.GetUserStatus(context.Background(), GetUserStatusParams{
		UserID: user2.ID, WorkspaceID: workspace.ID,
	})
	require.NoError(t, err)

	require.Equal(t, "online", status1After.Status)
	require.Equal(t, "online", status2After.Status)

	// Set inactivity threshold to future time (all users should be affected because their activity is before this)
	futureTime := time.Now().UTC().Add(1 * time.Hour) // 1 hour from now
	err = testQueries.SetUsersOfflineAfterInactivity(context.Background(), futureTime)
	require.NoError(t, err)

	// Both users should now be offline
	status1Final, _ := testQueries.GetUserStatus(context.Background(), GetUserStatusParams{
		UserID: user1.ID, WorkspaceID: workspace.ID,
	})
	status2Final, _ := testQueries.GetUserStatus(context.Background(), GetUserStatusParams{
		UserID: user2.ID, WorkspaceID: workspace.ID,
	})

	require.Equal(t, "offline", status1Final.Status)
	require.Equal(t, "offline", status2Final.Status)
}

func TestGetOnlineUsersInWorkspace(t *testing.T) {
	workspace, user1 := createTestWorkspaceAndUser(t)
	user2 := createRandomUserForOrganization(t, workspace.OrganizationID)
	user3 := createRandomUserForOrganization(t, workspace.OrganizationID)

	// Assign users to workspace
	testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user2.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user3.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})

	// Set different statuses
	testQueries.UpsertUserStatus(context.Background(), UpsertUserStatusParams{
		UserID:      user1.ID,
		WorkspaceID: workspace.ID,
		Status:      "online",
	})
	testQueries.UpsertUserStatus(context.Background(), UpsertUserStatusParams{
		UserID:      user2.ID,
		WorkspaceID: workspace.ID,
		Status:      "away",
	})
	testQueries.UpsertUserStatus(context.Background(), UpsertUserStatusParams{
		UserID:      user3.ID,
		WorkspaceID: workspace.ID,
		Status:      "offline",
	})

	onlineUsers, err := testQueries.GetOnlineUsersInWorkspace(context.Background(), workspace.ID)
	require.NoError(t, err)
	require.Len(t, onlineUsers, 2) // online and away users

	// Check that only online/away/busy users are returned (not offline)
	userIDs := make(map[int64]bool)
	for _, user := range onlineUsers {
		userIDs[user.UserID] = true
		require.Contains(t, []string{"online", "away", "busy"}, user.Status)
		require.NotEmpty(t, user.FirstName)
		require.NotEmpty(t, user.LastName)
		require.NotEmpty(t, user.Email)
	}

	require.True(t, userIDs[user1.ID])
	require.True(t, userIDs[user2.ID])
	require.False(t, userIDs[user3.ID]) // offline user should not be included
}

func TestUserStatusValidStatuses(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	validStatuses := []string{"online", "away", "busy", "offline"}

	for _, status := range validStatuses {
		arg := UpsertUserStatusParams{
			UserID:      user.ID,
			WorkspaceID: workspace.ID,
			Status:      status,
		}

		userStatus, err := testQueries.UpsertUserStatus(context.Background(), arg)
		require.NoError(t, err)
		require.Equal(t, status, userStatus.Status)
	}
}

func TestUserStatusInvalidStatus(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	arg := UpsertUserStatusParams{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
		Status:      "invalid_status", // Not in CHECK constraint
	}

	_, err := testQueries.UpsertUserStatus(context.Background(), arg)
	require.Error(t, err) // Should fail due to CHECK constraint
}

func TestUserStatusCustomStatusLength(t *testing.T) {
	workspace, user := createTestWorkspaceAndUser(t)

	// Test with maximum allowed custom status length (100 characters)
	longCustomStatus := util.RandomString(100)
	arg := UpsertUserStatusParams{
		UserID:       user.ID,
		WorkspaceID:  workspace.ID,
		Status:       "online",
		CustomStatus: sql.NullString{String: longCustomStatus, Valid: true},
	}

	userStatus, err := testQueries.UpsertUserStatus(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, longCustomStatus, userStatus.CustomStatus.String)

	// Test with custom status exceeding limit (should fail)
	tooLongCustomStatus := util.RandomString(101)
	arg.CustomStatus = sql.NullString{String: tooLongCustomStatus, Valid: true}

	_, err = testQueries.UpsertUserStatus(context.Background(), arg)
	require.Error(t, err) // Should fail due to CHECK constraint
}

func TestUserStatusWorkspaceConstraint(t *testing.T) {
	// This test verifies that the trigger prevents creating status for wrong workspace
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)
	workspace2 := createRandomWorkspace(t, organization)
	user := createRandomUserForOrganization(t, organization.ID)

	// Assign user to workspace1
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace1.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	// Try to create status for workspace2 (should fail due to trigger)
	arg := UpsertUserStatusParams{
		UserID:      user.ID,
		WorkspaceID: workspace2.ID, // Wrong workspace
		Status:      "online",
	}

	_, err = testQueries.UpsertUserStatus(context.Background(), arg)
	require.Error(t, err) // Should fail due to trigger constraint

	// But creating status for correct workspace should work
	arg.WorkspaceID = workspace1.ID
	_, err = testQueries.UpsertUserStatus(context.Background(), arg)
	require.NoError(t, err)
}
