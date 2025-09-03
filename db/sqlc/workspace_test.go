package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomWorkspace(t *testing.T, organization Organization) Workspace {
	arg := CreateWorkspaceParams{
		OrganizationID: organization.ID,
		Name:           util.RandomString(10),
	}

	workspace, err := testQueries.CreateWorkspace(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, workspace)

	require.Equal(t, arg.OrganizationID, workspace.OrganizationID)
	require.Equal(t, arg.Name, workspace.Name)

	require.NotZero(t, workspace.ID)
	require.NotZero(t, workspace.CreatedAt)

	return workspace
}

func TestCreateWorkspace(t *testing.T) {
	organization := createRandomOrganization(t)
	createRandomWorkspace(t, organization)
}

func TestGetWorkspace(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)

	workspace2, err := testQueries.GetWorkspace(context.Background(), workspace1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, workspace2)

	require.Equal(t, workspace1.ID, workspace2.ID)
	require.Equal(t, workspace1.OrganizationID, workspace2.OrganizationID)
	require.Equal(t, workspace1.Name, workspace2.Name)
	require.WithinDuration(t, workspace1.CreatedAt, workspace2.CreatedAt, time.Second)
}

func TestGetWorkspaceByID(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)

	workspace2, err := testQueries.GetWorkspaceByID(context.Background(), workspace1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, workspace2)

	require.Equal(t, workspace1.ID, workspace2.ID)
	require.Equal(t, workspace1.OrganizationID, workspace2.OrganizationID)
	require.Equal(t, workspace1.Name, workspace2.Name)
	require.WithinDuration(t, workspace1.CreatedAt, workspace2.CreatedAt, time.Second)
}

func TestUpdateWorkspace(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)

	arg := UpdateWorkspaceParams{
		ID:   workspace1.ID,
		Name: util.RandomString(10),
	}

	workspace2, err := testQueries.UpdateWorkspace(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, workspace2)

	require.Equal(t, workspace1.ID, workspace2.ID)
	require.Equal(t, workspace1.OrganizationID, workspace2.OrganizationID)
	require.Equal(t, arg.Name, workspace2.Name)
	require.WithinDuration(t, workspace1.CreatedAt, workspace2.CreatedAt, time.Second)
}

func TestDeleteWorkspace(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)

	err := testQueries.DeleteWorkspace(context.Background(), workspace1.ID)
	require.NoError(t, err)

	workspace2, err := testQueries.GetWorkspace(context.Background(), workspace1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, workspace2)
}

func TestListWorkspacesByOrganization(t *testing.T) {
	organization := createRandomOrganization(t)

	// Create multiple workspaces
	workspaces := make([]Workspace, 0)
	for i := 0; i < 10; i++ {
		workspace := createRandomWorkspace(t, organization)
		workspaces = append(workspaces, workspace)
	}

	arg := ListWorkspacesByOrganizationParams{
		OrganizationID: organization.ID,
		Limit:          5,
		Offset:         5,
	}

	retrievedWorkspaces, err := testQueries.ListWorkspacesByOrganization(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, retrievedWorkspaces, 5)

	for _, workspace := range retrievedWorkspaces {
		require.NotEmpty(t, workspace)
		require.Equal(t, organization.ID, workspace.OrganizationID)
	}
}

func TestListWorkspacesByOrganizationEmpty(t *testing.T) {
	organization := createRandomOrganization(t)

	arg := ListWorkspacesByOrganizationParams{
		OrganizationID: organization.ID,
		Limit:          5,
		Offset:         0,
	}

	workspaces, err := testQueries.ListWorkspacesByOrganization(context.Background(), arg)
	require.NoError(t, err)
	require.Empty(t, workspaces)
}

func TestGetWorkspaceWithUserCount(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)

	// Create users and assign them to the workspace
	user1 := createRandomUserForOrganization(t, organization.ID)
	user2 := createRandomUserForOrganization(t, organization.ID)

	// Assign users to workspace
	_, err := testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user1.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "admin",
	})
	require.NoError(t, err)

	_, err = testQueries.UpdateUserWorkspace(context.Background(), UpdateUserWorkspaceParams{
		ID:          user2.ID,
		WorkspaceID: sql.NullInt64{Int64: workspace.ID, Valid: true},
		Role:        "member",
	})
	require.NoError(t, err)

	result, err := testQueries.GetWorkspaceWithUserCount(context.Background(), workspace.ID)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, workspace.ID, result.ID)
	require.Equal(t, workspace.OrganizationID, result.OrganizationID)
	require.Equal(t, workspace.Name, result.Name)
	require.Equal(t, int64(2), result.UserCount)
	require.WithinDuration(t, workspace.CreatedAt, result.CreatedAt, time.Second)
}

func TestGetWorkspaceNotFound(t *testing.T) {
	workspace, err := testQueries.GetWorkspace(context.Background(), 999999)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, workspace)
}

func TestUpdateWorkspaceNotFound(t *testing.T) {
	arg := UpdateWorkspaceParams{
		ID:   999999,
		Name: util.RandomString(10),
	}

	workspace, err := testQueries.UpdateWorkspace(context.Background(), arg)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, workspace)
}

func TestCreateWorkspaceWithInvalidOrganization(t *testing.T) {
	arg := CreateWorkspaceParams{
		OrganizationID: 999999, // Non-existent organization
		Name:           util.RandomString(10),
	}

	workspace, err := testQueries.CreateWorkspace(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, workspace)
}

func TestCreateWorkspaceWithDuplicateName(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace1 := createRandomWorkspace(t, organization)

	// Try to create another workspace with the same name in the same organization
	arg := CreateWorkspaceParams{
		OrganizationID: organization.ID,
		Name:           workspace1.Name,
	}

	workspace2, err := testQueries.CreateWorkspace(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, workspace2)
}

func TestCreateWorkspaceWithSameNameDifferentOrganization(t *testing.T) {
	// This should be allowed - same workspace name in different organizations
	organization1 := createRandomOrganization(t)
	organization2 := createRandomOrganization(t)

	workspaceName := util.RandomString(10)

	arg1 := CreateWorkspaceParams{
		OrganizationID: organization1.ID,
		Name:           workspaceName,
	}

	workspace1, err := testQueries.CreateWorkspace(context.Background(), arg1)
	require.NoError(t, err)
	require.NotEmpty(t, workspace1)

	arg2 := CreateWorkspaceParams{
		OrganizationID: organization2.ID,
		Name:           workspaceName,
	}

	workspace2, err := testQueries.CreateWorkspace(context.Background(), arg2)
	require.NoError(t, err)
	require.NotEmpty(t, workspace2)

	require.Equal(t, workspaceName, workspace1.Name)
	require.Equal(t, workspaceName, workspace2.Name)
	require.NotEqual(t, workspace1.OrganizationID, workspace2.OrganizationID)
}
