package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomWorkspaceInvitation(t *testing.T) WorkspaceInvitation {
	inviter := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	inviteeEmail := util.RandomEmail()

	arg := CreateWorkspaceInvitationParams{
		WorkspaceID:    workspace.ID,
		InviterID:      inviter.ID,
		InviteeEmail:   inviteeEmail,
		InviteeID:      sql.NullInt64{},
		InvitationCode: util.RandomString(32),
		Role:           "member",
		ExpiresAt:      time.Now().Add(time.Hour * 48),
	}

	invitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, invitation)

	require.Equal(t, arg.WorkspaceID, invitation.WorkspaceID)
	require.Equal(t, arg.InviterID, invitation.InviterID)
	require.Equal(t, arg.InviteeEmail, invitation.InviteeEmail)
	require.Equal(t, arg.InviteeID, invitation.InviteeID)
	require.Equal(t, arg.InvitationCode, invitation.InvitationCode)
	require.Equal(t, arg.Role, invitation.Role)
	require.WithinDuration(t, arg.ExpiresAt, invitation.ExpiresAt, time.Second)
	require.NotZero(t, invitation.ID)
	require.NotZero(t, invitation.CreatedAt)
	require.Equal(t, "pending", invitation.Status) // Should default to pending
	require.False(t, invitation.AcceptedAt.Valid)

	return invitation
}

func createRandomWorkspaceInvitationWithInvitee(t *testing.T) WorkspaceInvitation {
	inviter := createRandomUser(t)
	invitee := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)

	arg := CreateWorkspaceInvitationParams{
		WorkspaceID:    workspace.ID,
		InviterID:      inviter.ID,
		InviteeEmail:   invitee.Email,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
		InvitationCode: util.RandomString(32),
		Role:           "admin",
		ExpiresAt:      time.Now().Add(time.Hour * 72),
	}

	invitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, invitation)

	return invitation
}

func TestCreateWorkspaceInvitation(t *testing.T) {
	createRandomWorkspaceInvitation(t)
}

func TestCreateWorkspaceInvitationWithInvitee(t *testing.T) {
	createRandomWorkspaceInvitationWithInvitee(t)
}

func TestGetWorkspaceInvitation(t *testing.T) {
	invitation1 := createRandomWorkspaceInvitation(t)

	invitation2, err := testQueries.GetWorkspaceInvitation(context.Background(), invitation1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, invitation2)

	require.Equal(t, invitation1.ID, invitation2.ID)
	require.Equal(t, invitation1.WorkspaceID, invitation2.WorkspaceID)
	require.Equal(t, invitation1.InviterID, invitation2.InviterID)
	require.Equal(t, invitation1.InviteeEmail, invitation2.InviteeEmail)
	require.Equal(t, invitation1.InviteeID, invitation2.InviteeID)
	require.Equal(t, invitation1.InvitationCode, invitation2.InvitationCode)
	require.Equal(t, invitation1.Role, invitation2.Role)
	require.Equal(t, invitation1.Status, invitation2.Status)
	require.WithinDuration(t, invitation1.ExpiresAt, invitation2.ExpiresAt, time.Second)
	require.WithinDuration(t, invitation1.CreatedAt, invitation2.CreatedAt, time.Second)
}

func TestGetWorkspaceInvitationByCode(t *testing.T) {
	invitation1 := createRandomWorkspaceInvitation(t)

	invitation2, err := testQueries.GetWorkspaceInvitationByCode(context.Background(), invitation1.InvitationCode)
	require.NoError(t, err)
	require.NotEmpty(t, invitation2)

	require.Equal(t, invitation1.ID, invitation2.ID)
	require.Equal(t, invitation1.InvitationCode, invitation2.InvitationCode)
	require.Equal(t, "pending", invitation2.Status)
}

func TestGetWorkspaceInvitationByCodeExpired(t *testing.T) {
	inviter := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)

	// Create expired invitation
	arg := CreateWorkspaceInvitationParams{
		WorkspaceID:    workspace.ID,
		InviterID:      inviter.ID,
		InviteeEmail:   util.RandomEmail(),
		InviteeID:      sql.NullInt64{},
		InvitationCode: util.RandomString(32),
		Role:           "member",
		ExpiresAt:      time.Now().Add(-time.Hour), // Expired
	}

	invitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
	require.NoError(t, err)

	// Should not be able to get expired invitation
	_, err = testQueries.GetWorkspaceInvitationByCode(context.Background(), invitation.InvitationCode)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetWorkspaceInvitationByCodeAccepted(t *testing.T) {
	invitation := createRandomWorkspaceInvitation(t)
	invitee := createRandomUser(t)

	// Accept the invitation first
	_, err := testQueries.AcceptWorkspaceInvitation(context.Background(), AcceptWorkspaceInvitationParams{
		InvitationCode: invitation.InvitationCode,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
	})
	require.NoError(t, err)

	// Should not be able to get accepted invitation by code (only pending ones)
	_, err = testQueries.GetWorkspaceInvitationByCode(context.Background(), invitation.InvitationCode)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestListWorkspaceInvitations(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	inviter := createRandomUser(t)

	// Create multiple invitations for the workspace
	invitationCount := 5
	for i := 0; i < invitationCount; i++ {
		arg := CreateWorkspaceInvitationParams{
			WorkspaceID:    workspace.ID,
			InviterID:      inviter.ID,
			InviteeEmail:   util.RandomEmail(),
			InviteeID:      sql.NullInt64{},
			InvitationCode: util.RandomString(32),
			Role:           "member",
			ExpiresAt:      time.Now().Add(time.Hour * 48),
		}
		_, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
		require.NoError(t, err)

		// Small delay to ensure different creation times
		time.Sleep(time.Millisecond * 10)
	}

	// Create invitation for another workspace (should not be included)
	otherWorkspace := createRandomWorkspace(t, organization)
	otherArg := CreateWorkspaceInvitationParams{
		WorkspaceID:    otherWorkspace.ID,
		InviterID:      inviter.ID,
		InviteeEmail:   util.RandomEmail(),
		InviteeID:      sql.NullInt64{},
		InvitationCode: util.RandomString(32),
		Role:           "member",
		ExpiresAt:      time.Now().Add(time.Hour * 48),
	}
	_, err := testQueries.CreateWorkspaceInvitation(context.Background(), otherArg)
	require.NoError(t, err)

	// List invitations for the first workspace
	invitations, err := testQueries.ListWorkspaceInvitations(context.Background(), ListWorkspaceInvitationsParams{
		WorkspaceID: workspace.ID,
		Limit:       10,
		Offset:      0,
	})
	require.NoError(t, err)
	require.Len(t, invitations, invitationCount)

	// Verify all invitations belong to the correct workspace
	for _, invitation := range invitations {
		require.Equal(t, workspace.ID, invitation.WorkspaceID)
	}

	// Verify invitations are ordered by created_at DESC (most recent first)
	for i := 1; i < len(invitations); i++ {
		require.True(t, invitations[i].CreatedAt.Before(invitations[i-1].CreatedAt) ||
			invitations[i].CreatedAt.Equal(invitations[i-1].CreatedAt))
	}
}

func TestListWorkspaceInvitationsPagination(t *testing.T) {
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	inviter := createRandomUser(t)

	// Create multiple invitations
	invitationCount := 7
	for i := 0; i < invitationCount; i++ {
		arg := CreateWorkspaceInvitationParams{
			WorkspaceID:    workspace.ID,
			InviterID:      inviter.ID,
			InviteeEmail:   util.RandomEmail(),
			InviteeID:      sql.NullInt64{},
			InvitationCode: util.RandomString(32),
			Role:           "member",
			ExpiresAt:      time.Now().Add(time.Hour * 48),
		}
		_, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 10)
	}

	// Get first page
	firstPage, err := testQueries.ListWorkspaceInvitations(context.Background(), ListWorkspaceInvitationsParams{
		WorkspaceID: workspace.ID,
		Limit:       3,
		Offset:      0,
	})
	require.NoError(t, err)
	require.Len(t, firstPage, 3)

	// Get second page
	secondPage, err := testQueries.ListWorkspaceInvitations(context.Background(), ListWorkspaceInvitationsParams{
		WorkspaceID: workspace.ID,
		Limit:       3,
		Offset:      3,
	})
	require.NoError(t, err)
	require.Len(t, secondPage, 3)

	// Get third page
	thirdPage, err := testQueries.ListWorkspaceInvitations(context.Background(), ListWorkspaceInvitationsParams{
		WorkspaceID: workspace.ID,
		Limit:       3,
		Offset:      6,
	})
	require.NoError(t, err)
	require.Len(t, thirdPage, 1) // Only 1 remaining

	// Verify no overlap between pages
	allIDs := make(map[int64]bool)
	for _, invitation := range firstPage {
		allIDs[invitation.ID] = true
	}
	for _, invitation := range secondPage {
		require.False(t, allIDs[invitation.ID], "Found duplicate invitation across pages")
		allIDs[invitation.ID] = true
	}
	for _, invitation := range thirdPage {
		require.False(t, allIDs[invitation.ID], "Found duplicate invitation across pages")
	}
}

func TestAcceptWorkspaceInvitation(t *testing.T) {
	invitation := createRandomWorkspaceInvitation(t)
	invitee := createRandomUser(t)

	acceptedInvitation, err := testQueries.AcceptWorkspaceInvitation(context.Background(), AcceptWorkspaceInvitationParams{
		InvitationCode: invitation.InvitationCode,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
	})
	require.NoError(t, err)
	require.NotEmpty(t, acceptedInvitation)

	require.Equal(t, invitation.ID, acceptedInvitation.ID)
	require.Equal(t, "accepted", acceptedInvitation.Status)
	require.Equal(t, invitee.ID, acceptedInvitation.InviteeID.Int64)
	require.True(t, acceptedInvitation.InviteeID.Valid)
	require.True(t, acceptedInvitation.AcceptedAt.Valid)
	require.WithinDuration(t, time.Now(), acceptedInvitation.AcceptedAt.Time, time.Second*5)
}

func TestAcceptWorkspaceInvitationExpired(t *testing.T) {
	inviter := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	invitee := createRandomUser(t)

	// Create expired invitation
	arg := CreateWorkspaceInvitationParams{
		WorkspaceID:    workspace.ID,
		InviterID:      inviter.ID,
		InviteeEmail:   util.RandomEmail(),
		InviteeID:      sql.NullInt64{},
		InvitationCode: util.RandomString(32),
		Role:           "member",
		ExpiresAt:      time.Now().Add(-time.Hour), // Expired
	}

	invitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
	require.NoError(t, err)

	// Try to accept expired invitation
	_, err = testQueries.AcceptWorkspaceInvitation(context.Background(), AcceptWorkspaceInvitationParams{
		InvitationCode: invitation.InvitationCode,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
	})
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeclineWorkspaceInvitation(t *testing.T) {
	invitation := createRandomWorkspaceInvitation(t)

	declinedInvitation, err := testQueries.DeclineWorkspaceInvitation(context.Background(), invitation.InvitationCode)
	require.NoError(t, err)
	require.NotEmpty(t, declinedInvitation)

	require.Equal(t, invitation.ID, declinedInvitation.ID)
	require.Equal(t, "declined", declinedInvitation.Status)
	require.False(t, declinedInvitation.AcceptedAt.Valid)
}

func TestDeclineWorkspaceInvitationAlreadyAccepted(t *testing.T) {
	invitation := createRandomWorkspaceInvitation(t)
	invitee := createRandomUser(t)

	// Accept the invitation first
	_, err := testQueries.AcceptWorkspaceInvitation(context.Background(), AcceptWorkspaceInvitationParams{
		InvitationCode: invitation.InvitationCode,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
	})
	require.NoError(t, err)

	// Try to decline already accepted invitation
	_, err = testQueries.DeclineWorkspaceInvitation(context.Background(), invitation.InvitationCode)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestExpireWorkspaceInvitation(t *testing.T) {
	invitation := createRandomWorkspaceInvitation(t)

	err := testQueries.ExpireWorkspaceInvitation(context.Background(), invitation.ID)
	require.NoError(t, err)

	// Verify the invitation is expired
	updatedInvitation, err := testQueries.GetWorkspaceInvitation(context.Background(), invitation.ID)
	require.NoError(t, err)
	require.Equal(t, "expired", updatedInvitation.Status)
}

func TestDeleteWorkspaceInvitation(t *testing.T) {
	invitation := createRandomWorkspaceInvitation(t)

	err := testQueries.DeleteWorkspaceInvitation(context.Background(), invitation.ID)
	require.NoError(t, err)

	// Verify the invitation is deleted
	_, err = testQueries.GetWorkspaceInvitation(context.Background(), invitation.ID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetPendingInvitationsForUser(t *testing.T) {
	invitee := createRandomUser(t)

	// Create multiple pending invitations for the user
	workspaces := make([]Workspace, 3)
	inviters := make([]User, 3)
	organizations := make([]Organization, 3)

	for i := 0; i < 3; i++ {
		organizations[i] = createRandomOrganization(t)
		workspaces[i] = createRandomWorkspace(t, organizations[i])
		inviters[i] = createRandomUser(t)

		arg := CreateWorkspaceInvitationParams{
			WorkspaceID:    workspaces[i].ID,
			InviterID:      inviters[i].ID,
			InviteeEmail:   invitee.Email,
			InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
			InvitationCode: util.RandomString(32),
			Role:           "member",
			ExpiresAt:      time.Now().Add(time.Hour * 48),
		}
		_, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
		require.NoError(t, err)

		time.Sleep(time.Millisecond * 10)
	}

	// Create an expired invitation (should not be included)
	expiredOrg := createRandomOrganization(t)
	expiredWorkspace := createRandomWorkspace(t, expiredOrg)
	expiredInviter := createRandomUser(t)
	expiredArg := CreateWorkspaceInvitationParams{
		WorkspaceID:    expiredWorkspace.ID,
		InviterID:      expiredInviter.ID,
		InviteeEmail:   invitee.Email,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
		InvitationCode: util.RandomString(32),
		Role:           "member",
		ExpiresAt:      time.Now().Add(-time.Hour), // Expired
	}
	_, err := testQueries.CreateWorkspaceInvitation(context.Background(), expiredArg)
	require.NoError(t, err)

	// Create an accepted invitation (should not be included)
	acceptedOrg := createRandomOrganization(t)
	acceptedWorkspace := createRandomWorkspace(t, acceptedOrg)
	acceptedInviter := createRandomUser(t)
	acceptedArg := CreateWorkspaceInvitationParams{
		WorkspaceID:    acceptedWorkspace.ID,
		InviterID:      acceptedInviter.ID,
		InviteeEmail:   invitee.Email,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
		InvitationCode: util.RandomString(32),
		Role:           "admin",
		ExpiresAt:      time.Now().Add(time.Hour * 48),
	}
	acceptedInvitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), acceptedArg)
	require.NoError(t, err)

	_, err = testQueries.AcceptWorkspaceInvitation(context.Background(), AcceptWorkspaceInvitationParams{
		InvitationCode: acceptedInvitation.InvitationCode,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
	})
	require.NoError(t, err)

	// Get pending invitations for the user
	pendingInvitations, err := testQueries.GetPendingInvitationsForUser(context.Background(), invitee.Email)
	require.NoError(t, err)
	require.Len(t, pendingInvitations, 3) // Only pending, non-expired invitations

	// Verify all invitations are pending and contain workspace/inviter info
	for _, invitation := range pendingInvitations {
		require.Equal(t, "pending", invitation.Status)
		require.Equal(t, invitee.Email, invitation.InviteeEmail)
		require.NotEmpty(t, invitation.WorkspaceName)
		require.NotEmpty(t, invitation.InviterFirstName)
		require.NotEmpty(t, invitation.InviterLastName)
		require.True(t, invitation.ExpiresAt.After(time.Now()))
	}

	// Verify invitations are ordered by created_at DESC (most recent first)
	for i := 1; i < len(pendingInvitations); i++ {
		require.True(t, pendingInvitations[i].CreatedAt.Before(pendingInvitations[i-1].CreatedAt) ||
			pendingInvitations[i].CreatedAt.Equal(pendingInvitations[i-1].CreatedAt))
	}
}

func TestWorkspaceInvitationRoles(t *testing.T) {
	inviter := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)

	testRoles := []string{"admin", "member"}

	for _, role := range testRoles {
		t.Run("role_"+role, func(t *testing.T) {
			arg := CreateWorkspaceInvitationParams{
				WorkspaceID:    workspace.ID,
				InviterID:      inviter.ID,
				InviteeEmail:   util.RandomEmail(),
				InviteeID:      sql.NullInt64{},
				InvitationCode: util.RandomString(32),
				Role:           role,
				ExpiresAt:      time.Now().Add(time.Hour * 48),
			}

			invitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
			require.NoError(t, err)
			require.Equal(t, role, invitation.Role)

			// Verify retrieval
			retrievedInvitation, err := testQueries.GetWorkspaceInvitation(context.Background(), invitation.ID)
			require.NoError(t, err)
			require.Equal(t, role, retrievedInvitation.Role)
		})
	}
}

func TestWorkspaceInvitationLifecycle(t *testing.T) {
	inviter := createRandomUser(t)
	organization := createRandomOrganization(t)
	workspace := createRandomWorkspace(t, organization)
	inviteeEmail := util.RandomEmail()
	invitationCode := util.RandomString(32)

	// Step 1: Create invitation
	arg := CreateWorkspaceInvitationParams{
		WorkspaceID:    workspace.ID,
		InviterID:      inviter.ID,
		InviteeEmail:   inviteeEmail,
		InviteeID:      sql.NullInt64{},
		InvitationCode: invitationCode,
		Role:           "member",
		ExpiresAt:      time.Now().Add(time.Hour * 48),
	}

	invitation, err := testQueries.CreateWorkspaceInvitation(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, "pending", invitation.Status)

	// Step 2: Verify invitation can be retrieved by code
	codeInvitation, err := testQueries.GetWorkspaceInvitationByCode(context.Background(), invitationCode)
	require.NoError(t, err)
	require.Equal(t, invitation.ID, codeInvitation.ID)

	// Step 3: Check pending invitations for user
	pendingInvitations, err := testQueries.GetPendingInvitationsForUser(context.Background(), inviteeEmail)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(pendingInvitations), 1)

	// Step 4: Accept invitation
	invitee := createRandomUser(t)
	acceptedInvitation, err := testQueries.AcceptWorkspaceInvitation(context.Background(), AcceptWorkspaceInvitationParams{
		InvitationCode: invitationCode,
		InviteeID:      sql.NullInt64{Int64: invitee.ID, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, "accepted", acceptedInvitation.Status)
	require.True(t, acceptedInvitation.AcceptedAt.Valid)

	// Step 5: Verify invitation can no longer be retrieved by code (only pending ones)
	_, err = testQueries.GetWorkspaceInvitationByCode(context.Background(), invitationCode)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	// Step 6: Verify invitation can still be retrieved by ID
	finalInvitation, err := testQueries.GetWorkspaceInvitation(context.Background(), invitation.ID)
	require.NoError(t, err)
	require.Equal(t, "accepted", finalInvitation.Status)
}
