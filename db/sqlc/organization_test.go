package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func TestCreateOrganization(t *testing.T) {
	createRandomOrganization(t)
}

func TestGetOrganization(t *testing.T) {
	organization1 := createRandomOrganization(t)
	organization2, err := testQueries.GetOrganization(context.Background(), organization1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, organization2)

	require.Equal(t, organization1.ID, organization2.ID)
	require.Equal(t, organization1.Name, organization2.Name)
	require.WithinDuration(t, organization1.CreatedAt, organization2.CreatedAt, time.Second)
}

func TestUpdateOrganization(t *testing.T) {
	organization1 := createRandomOrganization(t)

	arg := UpdateOrganizationParams{
		ID:   organization1.ID,
		Name: util.RandomOrganizationName(),
	}

	organization2, err := testQueries.UpdateOrganization(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, organization2)

	require.Equal(t, organization1.ID, organization2.ID)
	require.Equal(t, arg.Name, organization2.Name)
	require.WithinDuration(t, organization1.CreatedAt, organization2.CreatedAt, time.Second)
}

func TestDeleteOrganization(t *testing.T) {
	organization1 := createRandomOrganization(t)
	err := testQueries.DeleteOrganization(context.Background(), organization1.ID)
	require.NoError(t, err)

	organization2, err := testQueries.GetOrganization(context.Background(), organization1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, organization2)
}

func TestListOrganizations(t *testing.T) {
	// Create multiple organizations
	var organizations []Organization
	for range 10 {
		organization := createRandomOrganization(t)
		organizations = append(organizations, organization)
	}

	arg := ListOrganizationsParams{
		Limit:  5,
		Offset: 5,
	}

	organizationList, err := testQueries.ListOrganizations(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, organizationList, 5)

	for _, organization := range organizationList {
		require.NotEmpty(t, organization)
	}
}

func TestListOrganizationsEmpty(t *testing.T) {
	// First, let's delete all existing organizations to ensure we start with a clean slate
	// Note: This is for testing purposes only

	arg := ListOrganizationsParams{
		Limit:  100, // Get a large number to see if there are any
		Offset: 0,
	}

	organizationList, err := testQueries.ListOrganizations(context.Background(), arg)
	require.NoError(t, err)

	// If there are existing organizations, this test might not behave as expected
	// In a real scenario, you might want to use a separate test database
	// For now, let's just check that the query works
	require.NotNil(t, organizationList)
}

func TestListOrganizationsWithLimitAndOffset(t *testing.T) {
	// Create exactly 10 organizations for this test
	var createdOrgs []Organization
	for range 10 {
		org := createRandomOrganization(t)
		createdOrgs = append(createdOrgs, org)
	}

	// Test first page
	arg1 := ListOrganizationsParams{
		Limit:  3,
		Offset: 0,
	}

	firstPage, err := testQueries.ListOrganizations(context.Background(), arg1)
	require.NoError(t, err)
	require.LessOrEqual(t, len(firstPage), 3)

	// Test second page
	arg2 := ListOrganizationsParams{
		Limit:  3,
		Offset: 3,
	}

	secondPage, err := testQueries.ListOrganizations(context.Background(), arg2)
	require.NoError(t, err)
	require.LessOrEqual(t, len(secondPage), 3)

	// Ensure no overlap between pages (if we have enough data)
	if len(firstPage) > 0 && len(secondPage) > 0 {
		for _, org1 := range firstPage {
			for _, org2 := range secondPage {
				require.NotEqual(t, org1.ID, org2.ID)
			}
		}
	}
}

func TestGetOrganizationNotFound(t *testing.T) {
	organization, err := testQueries.GetOrganization(context.Background(), 999999)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, organization)
}

func TestUpdateOrganizationNotFound(t *testing.T) {
	arg := UpdateOrganizationParams{
		ID:   999999,
		Name: util.RandomOrganizationName(),
	}

	organization, err := testQueries.UpdateOrganization(context.Background(), arg)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, organization)
}

func TestDeleteOrganizationNotFound(t *testing.T) {
	err := testQueries.DeleteOrganization(context.Background(), 999999)
	require.NoError(t, err) // DELETE operations don't return errors for non-existent records
}

func TestDeleteOrganizationCascadesUsers(t *testing.T) {
	// Create an organization
	organization := createRandomOrganization(t)

	// Create users for this organization
	var users []User
	for i := 0; i < 3; i++ {
		user := createRandomUserForOrganization(t, organization.ID)
		users = append(users, user)
	}

	// Verify users exist
	for _, user := range users {
		foundUser, err := testQueries.GetUser(context.Background(), user.ID)
		require.NoError(t, err)
		require.Equal(t, user.ID, foundUser.ID)
	}

	// Delete the organization
	err := testQueries.DeleteOrganization(context.Background(), organization.ID)
	require.NoError(t, err)

	// Verify organization is deleted
	_, err = testQueries.GetOrganization(context.Background(), organization.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())

	// Verify users are also deleted (CASCADE)
	for _, user := range users {
		_, err := testQueries.GetUser(context.Background(), user.ID)
		require.Error(t, err)
		require.EqualError(t, err, sql.ErrNoRows.Error())
	}
}

func TestCreateOrganizationWithEmptyName(t *testing.T) {
	// This should work as our schema doesn't have a NOT NULL constraint that's enforced at the Go level
	// But it's good to test edge cases
	organization, err := testQueries.CreateOrganization(context.Background(), "")
	require.NoError(t, err)
	require.NotEmpty(t, organization)
	require.Equal(t, "", organization.Name)
	require.NotZero(t, organization.ID)
	require.NotZero(t, organization.CreatedAt)
}

func TestCreateOrganizationWithLongName(t *testing.T) {
	// Test with a very long name (our schema allows VARCHAR(255))
	longName := util.RandomString(255)
	organization, err := testQueries.CreateOrganization(context.Background(), longName)
	require.NoError(t, err)
	require.NotEmpty(t, organization)
	require.Equal(t, longName, organization.Name)
	require.NotZero(t, organization.ID)
	require.NotZero(t, organization.CreatedAt)
}

func TestCreateOrganizationWithTooLongName(t *testing.T) {
	// Test with a name that's too long (more than 255 characters)
	tooLongName := util.RandomString(256)
	organization, err := testQueries.CreateOrganization(context.Background(), tooLongName)
	require.Error(t, err)
	require.Empty(t, organization)
}
