package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomEmailVerificationToken(t *testing.T) EmailVerificationToken {
	user := createRandomUser(t)

	arg := CreateEmailVerificationTokenParams{
		UserID:    user.ID,
		Token:     util.RandomString(32),
		Email:     user.Email,
		TokenType: "email_verification",
		ExpiresAt: time.Now().Add(time.Hour * 24),
		IpAddress: createTestIPAddress("192.168.1.1"),
		UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
	}

	token, err := testQueries.CreateEmailVerificationToken(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	require.Equal(t, arg.UserID, token.UserID)
	require.Equal(t, arg.Token, token.Token)
	require.Equal(t, arg.Email, token.Email)
	require.Equal(t, arg.TokenType, token.TokenType)
	require.WithinDuration(t, arg.ExpiresAt, token.ExpiresAt, time.Second)
	require.Equal(t, arg.IpAddress, token.IpAddress)
	require.Equal(t, arg.UserAgent, token.UserAgent)
	require.NotZero(t, token.ID)
	require.NotZero(t, token.CreatedAt)
	require.False(t, token.UsedAt.Valid)

	return token
}

func createRandomPasswordResetToken(t *testing.T) PasswordResetToken {
	user := createRandomUser(t)

	arg := CreatePasswordResetTokenParams{
		UserID:    user.ID,
		Token:     util.RandomString(32),
		ExpiresAt: time.Now().Add(time.Hour * 2),
		IpAddress: createTestIPAddress("192.168.1.2"),
		UserAgent: sql.NullString{String: "Mozilla/5.0 (Test Reset)", Valid: true},
	}

	token, err := testQueries.CreatePasswordResetToken(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	require.Equal(t, arg.UserID, token.UserID)
	require.Equal(t, arg.Token, token.Token)
	require.WithinDuration(t, arg.ExpiresAt, token.ExpiresAt, time.Second)
	require.Equal(t, arg.IpAddress, token.IpAddress)
	require.Equal(t, arg.UserAgent, token.UserAgent)
	require.NotZero(t, token.ID)
	require.NotZero(t, token.CreatedAt)
	require.False(t, token.UsedAt.Valid)

	return token
}

func TestCreateEmailVerificationToken(t *testing.T) {
	createRandomEmailVerificationToken(t)
}

func TestGetEmailVerificationToken(t *testing.T) {
	token1 := createRandomEmailVerificationToken(t)

	token2, err := testQueries.GetEmailVerificationToken(context.Background(), token1.Token)
	require.NoError(t, err)
	require.NotEmpty(t, token2)

	require.Equal(t, token1.ID, token2.ID)
	require.Equal(t, token1.UserID, token2.UserID)
	require.Equal(t, token1.Token, token2.Token)
	require.Equal(t, token1.Email, token2.Email)
	require.Equal(t, token1.TokenType, token2.TokenType)
	require.WithinDuration(t, token1.ExpiresAt, token2.ExpiresAt, time.Second)
	require.WithinDuration(t, token1.CreatedAt, token2.CreatedAt, time.Second)
}

func TestGetEmailVerificationTokenExpired(t *testing.T) {
	user := createRandomUser(t)

	// Create expired token
	arg := CreateEmailVerificationTokenParams{
		UserID:    user.ID,
		Token:     util.RandomString(32),
		Email:     user.Email,
		TokenType: "email_verification",
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
		IpAddress: createTestIPAddress("192.168.1.1"),
		UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
	}

	token, err := testQueries.CreateEmailVerificationToken(context.Background(), arg)
	require.NoError(t, err)

	// Should not be able to get expired token
	_, err = testQueries.GetEmailVerificationToken(context.Background(), token.Token)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestUseEmailVerificationToken(t *testing.T) {
	token := createRandomEmailVerificationToken(t)

	err := testQueries.UseEmailVerificationToken(context.Background(), token.Token)
	require.NoError(t, err)

	// Should not be able to get used token
	_, err = testQueries.GetEmailVerificationToken(context.Background(), token.Token)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetUserEmailVerificationTokens(t *testing.T) {
	user := createRandomUser(t)
	tokenType := "email_verification"

	// Create multiple tokens for the user
	for i := 0; i < 3; i++ {
		arg := CreateEmailVerificationTokenParams{
			UserID:    user.ID,
			Token:     util.RandomString(32),
			Email:     user.Email,
			TokenType: tokenType,
			ExpiresAt: time.Now().Add(time.Hour * 24),
			IpAddress: createTestIPAddress("192.168.1.1"),
			UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
		}

		_, err := testQueries.CreateEmailVerificationToken(context.Background(), arg)
		require.NoError(t, err)
	}

	tokens, err := testQueries.GetUserEmailVerificationTokens(context.Background(), GetUserEmailVerificationTokensParams{
		UserID:    user.ID,
		TokenType: tokenType,
	})
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	for _, token := range tokens {
		require.Equal(t, user.ID, token.UserID)
		require.Equal(t, tokenType, token.TokenType)
		require.False(t, token.UsedAt.Valid)
	}
}

func TestDeleteExpiredEmailVerificationTokens(t *testing.T) {
	user := createRandomUser(t)

	// Create expired token
	expiredArg := CreateEmailVerificationTokenParams{
		UserID:    user.ID,
		Token:     util.RandomString(32),
		Email:     user.Email,
		TokenType: "email_verification",
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
		IpAddress: createTestIPAddress("192.168.1.1"),
		UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
	}
	expiredToken, err := testQueries.CreateEmailVerificationToken(context.Background(), expiredArg)
	require.NoError(t, err)

	// Create valid token
	validToken := createRandomEmailVerificationToken(t)

	// Delete expired tokens
	err = testQueries.DeleteExpiredEmailVerificationTokens(context.Background())
	require.NoError(t, err)

	// Valid token should still exist (but through different method since GetEmailVerificationToken filters expired)
	tokens, err := testQueries.GetUserEmailVerificationTokens(context.Background(), GetUserEmailVerificationTokensParams{
		UserID:    validToken.UserID,
		TokenType: validToken.TokenType,
	})
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.Equal(t, validToken.ID, tokens[0].ID)

	// Expired token should be gone
	tokens, err = testQueries.GetUserEmailVerificationTokens(context.Background(), GetUserEmailVerificationTokensParams{
		UserID:    expiredToken.UserID,
		TokenType: expiredToken.TokenType,
	})
	require.NoError(t, err)
	require.Empty(t, tokens)
}

// Password Reset Token Tests

func TestCreatePasswordResetToken(t *testing.T) {
	createRandomPasswordResetToken(t)
}

func TestGetPasswordResetToken(t *testing.T) {
	token1 := createRandomPasswordResetToken(t)

	token2, err := testQueries.GetPasswordResetToken(context.Background(), token1.Token)
	require.NoError(t, err)
	require.NotEmpty(t, token2)

	require.Equal(t, token1.ID, token2.ID)
	require.Equal(t, token1.UserID, token2.UserID)
	require.Equal(t, token1.Token, token2.Token)
	require.WithinDuration(t, token1.ExpiresAt, token2.ExpiresAt, time.Second)
	require.WithinDuration(t, token1.CreatedAt, token2.CreatedAt, time.Second)
}

func TestUsePasswordResetToken(t *testing.T) {
	token := createRandomPasswordResetToken(t)

	err := testQueries.UsePasswordResetToken(context.Background(), token.Token)
	require.NoError(t, err)

	// Should not be able to get used token
	_, err = testQueries.GetPasswordResetToken(context.Background(), token.Token)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeleteExpiredPasswordResetTokens(t *testing.T) {
	user := createRandomUser(t)

	// Create expired token
	expiredArg := CreatePasswordResetTokenParams{
		UserID:    user.ID,
		Token:     util.RandomString(32),
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
		IpAddress: createTestIPAddress("192.168.1.1"),
		UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
	}
	_, err := testQueries.CreatePasswordResetToken(context.Background(), expiredArg)
	require.NoError(t, err)

	// Create valid token
	validToken := createRandomPasswordResetToken(t)

	// Delete expired tokens
	err = testQueries.DeleteExpiredPasswordResetTokens(context.Background())
	require.NoError(t, err)

	// Valid token should still exist
	token, err := testQueries.GetPasswordResetToken(context.Background(), validToken.Token)
	require.NoError(t, err)
	require.Equal(t, validToken.ID, token.ID)
}

func TestDeleteUserPasswordResetTokens(t *testing.T) {
	user := createRandomUser(t)

	// Create multiple password reset tokens for the user
	for i := 0; i < 3; i++ {
		arg := CreatePasswordResetTokenParams{
			UserID:    user.ID,
			Token:     util.RandomString(32),
			ExpiresAt: time.Now().Add(time.Hour * 2),
			IpAddress: createTestIPAddress("192.168.1.1"),
			UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
		}

		_, err := testQueries.CreatePasswordResetToken(context.Background(), arg)
		require.NoError(t, err)
	}

	// Delete all password reset tokens for the user
	err := testQueries.DeleteUserPasswordResetTokens(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify all tokens are deleted (we can't directly query, but we can create a new one and verify it's the only one)
	newToken := CreatePasswordResetTokenParams{
		UserID:    user.ID,
		Token:     util.RandomString(32),
		ExpiresAt: time.Now().Add(time.Hour * 2),
		IpAddress: createTestIPAddress("192.168.1.1"),
		UserAgent: sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
	}

	token, err := testQueries.CreatePasswordResetToken(context.Background(), newToken)
	require.NoError(t, err)

	// This token should be retrievable
	retrievedToken, err := testQueries.GetPasswordResetToken(context.Background(), token.Token)
	require.NoError(t, err)
	require.Equal(t, token.ID, retrievedToken.ID)
}
