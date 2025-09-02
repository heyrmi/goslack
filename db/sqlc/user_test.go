package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/rahulmishra/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomOrganization(t *testing.T) Organization {
	name := util.RandomOrganizationName()

	organization, err := testQueries.CreateOrganization(context.Background(), name)
	require.NoError(t, err)
	require.NotEmpty(t, organization)

	require.Equal(t, name, organization.Name)
	require.NotZero(t, organization.ID)
	require.NotZero(t, organization.CreatedAt)

	return organization
}

func createRandomUser(t *testing.T) User {
	organization := createRandomOrganization(t)
	return createRandomUserForOrganization(t, organization.ID)
}

func createRandomUserForOrganization(t *testing.T, organizationID int64) User {
	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	arg := CreateUserParams{
		OrganizationID: organizationID,
		Email:          util.RandomEmail(),
		FirstName:      util.RandomString(6),
		LastName:       util.RandomString(6),
		HashedPassword: hashedPassword,
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.OrganizationID, user.OrganizationID)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.FirstName, user.FirstName)
	require.Equal(t, arg.LastName, user.LastName)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.PasswordChangedAt)
	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUser(context.Background(), user1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.OrganizationID, user2.OrganizationID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.FirstName, user2.FirstName)
	require.Equal(t, user1.LastName, user2.LastName)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func TestGetUserByEmail(t *testing.T) {
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUserByEmail(context.Background(), user1.Email)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.OrganizationID, user2.OrganizationID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.FirstName, user2.FirstName)
	require.Equal(t, user1.LastName, user2.LastName)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func TestUpdateUserProfile(t *testing.T) {
	user1 := createRandomUser(t)

	arg := UpdateUserProfileParams{
		ID:        user1.ID,
		FirstName: util.RandomString(6),
		LastName:  util.RandomString(6),
	}

	user2, err := testQueries.UpdateUserProfile(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.OrganizationID, user2.OrganizationID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, arg.FirstName, user2.FirstName)
	require.Equal(t, arg.LastName, user2.LastName)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.WithinDuration(t, user1.PasswordChangedAt, user2.PasswordChangedAt, time.Second)
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func TestUpdateUserPassword(t *testing.T) {
	user1 := createRandomUser(t)

	newPassword := util.RandomString(6)
	newHashedPassword, err := util.HashPassword(newPassword)
	require.NoError(t, err)

	arg := UpdateUserPasswordParams{
		ID:             user1.ID,
		HashedPassword: newHashedPassword,
	}

	user2, err := testQueries.UpdateUserPassword(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.OrganizationID, user2.OrganizationID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.FirstName, user2.FirstName)
	require.Equal(t, user1.LastName, user2.LastName)
	require.Equal(t, newHashedPassword, user2.HashedPassword)
	require.True(t, user2.PasswordChangedAt.After(user1.PasswordChangedAt))
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}

func TestListUsers(t *testing.T) {
	organization := createRandomOrganization(t)

	// Create multiple users for the same organization
	var users []User
	for range 10 {
		user := createRandomUserForOrganization(t, organization.ID)
		users = append(users, user)
	}

	arg := ListUsersParams{
		OrganizationID: organization.ID,
		Limit:          5,
		Offset:         5,
	}

	userList, err := testQueries.ListUsers(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, userList, 5)

	for _, user := range userList {
		require.NotEmpty(t, user)
		require.Equal(t, organization.ID, user.OrganizationID)
	}
}

func TestListUsersEmpty(t *testing.T) {
	organization := createRandomOrganization(t)

	arg := ListUsersParams{
		OrganizationID: organization.ID,
		Limit:          5,
		Offset:         0,
	}

	userList, err := testQueries.ListUsers(context.Background(), arg)
	require.NoError(t, err)
	require.Empty(t, userList)
}

func TestDeleteUser(t *testing.T) {
	user1 := createRandomUser(t)
	err := testQueries.DeleteUser(context.Background(), user1.ID)
	require.NoError(t, err)

	user2, err := testQueries.GetUser(context.Background(), user1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, user2)
}

func TestGetUserNotFound(t *testing.T) {
	user, err := testQueries.GetUser(context.Background(), 999999)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, user)
}

func TestGetUserByEmailNotFound(t *testing.T) {
	user, err := testQueries.GetUserByEmail(context.Background(), "nonexistent@example.com")
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, user)
}

func TestUpdateUserProfileNotFound(t *testing.T) {
	arg := UpdateUserProfileParams{
		ID:        999999,
		FirstName: util.RandomString(6),
		LastName:  util.RandomString(6),
	}

	user, err := testQueries.UpdateUserProfile(context.Background(), arg)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, user)
}

func TestUpdateUserPasswordNotFound(t *testing.T) {
	arg := UpdateUserPasswordParams{
		ID:             999999,
		HashedPassword: util.RandomString(60),
	}

	user, err := testQueries.UpdateUserPassword(context.Background(), arg)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, user)
}

func TestCreateUserWithInvalidOrganization(t *testing.T) {
	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	arg := CreateUserParams{
		OrganizationID: 999999, // Non-existent organization
		Email:          util.RandomEmail(),
		FirstName:      util.RandomString(6),
		LastName:       util.RandomString(6),
		HashedPassword: hashedPassword,
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, user)
}

func TestCreateUserWithDuplicateEmail(t *testing.T) {
	user1 := createRandomUser(t)

	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	arg := CreateUserParams{
		OrganizationID: user1.OrganizationID,
		Email:          user1.Email, // Duplicate email
		FirstName:      util.RandomString(6),
		LastName:       util.RandomString(6),
		HashedPassword: hashedPassword,
	}

	user2, err := testQueries.CreateUser(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, user2)
}
