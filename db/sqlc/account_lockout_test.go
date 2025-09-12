package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func createRandomAccountLockout(t *testing.T) AccountLockout {
	user := createRandomUser(t)

	lockout, err := testQueries.CreateAccountLockout(context.Background(), user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, lockout)

	require.Equal(t, user.ID, lockout.UserID)
	require.Equal(t, int32(1), lockout.FailedAttempts)
	require.NotZero(t, lockout.ID)
	require.NotZero(t, lockout.CreatedAt)
	require.NotZero(t, lockout.UpdatedAt)
	require.NotZero(t, lockout.LastFailedAttempt)

	return lockout
}

func TestCreateAccountLockout(t *testing.T) {
	createRandomAccountLockout(t)
}

func TestGetAccountLockout(t *testing.T) {
	lockout1 := createRandomAccountLockout(t)
	lockout2, err := testQueries.GetAccountLockout(context.Background(), lockout1.UserID)
	require.NoError(t, err)
	require.NotEmpty(t, lockout2)

	require.Equal(t, lockout1.ID, lockout2.ID)
	require.Equal(t, lockout1.UserID, lockout2.UserID)
	require.Equal(t, lockout1.FailedAttempts, lockout2.FailedAttempts)
	require.WithinDuration(t, lockout1.CreatedAt, lockout2.CreatedAt, time.Second)
	require.WithinDuration(t, lockout1.UpdatedAt, lockout2.UpdatedAt, time.Second)
}

func TestIncrementFailedAttempts(t *testing.T) {
	lockout1 := createRandomAccountLockout(t)

	lockout2, err := testQueries.IncrementFailedAttempts(context.Background(), lockout1.UserID)
	require.NoError(t, err)
	require.NotEmpty(t, lockout2)

	require.Equal(t, lockout1.ID, lockout2.ID)
	require.Equal(t, lockout1.UserID, lockout2.UserID)
	require.Equal(t, lockout1.FailedAttempts+1, lockout2.FailedAttempts)
	require.True(t, lockout2.UpdatedAt.After(lockout1.UpdatedAt))
	require.True(t, lockout2.LastFailedAttempt.Valid)
}

func TestLockAccount(t *testing.T) {
	lockout := createRandomAccountLockout(t)
	lockUntil := time.Now().Add(time.Hour)

	err := testQueries.LockAccount(context.Background(), LockAccountParams{
		UserID:      lockout.UserID,
		LockedUntil: sql.NullTime{Time: lockUntil, Valid: true},
	})
	require.NoError(t, err)

	// Verify the account is locked
	lockedLockout, err := testQueries.GetAccountLockout(context.Background(), lockout.UserID)
	require.NoError(t, err)
	require.True(t, lockedLockout.LockedUntil.Valid)
	require.WithinDuration(t, lockUntil, lockedLockout.LockedUntil.Time, time.Second)
}

func TestUnlockAccount(t *testing.T) {
	lockout := createRandomAccountLockout(t)
	lockUntil := time.Now().Add(time.Hour)

	// First lock the account
	err := testQueries.LockAccount(context.Background(), LockAccountParams{
		UserID:      lockout.UserID,
		LockedUntil: sql.NullTime{Time: lockUntil, Valid: true},
	})
	require.NoError(t, err)

	// Then unlock it
	err = testQueries.UnlockAccount(context.Background(), lockout.UserID)
	require.NoError(t, err)

	// Verify the account is unlocked
	unlockedLockout, err := testQueries.GetAccountLockout(context.Background(), lockout.UserID)
	require.NoError(t, err)
	require.False(t, unlockedLockout.LockedUntil.Valid)
	require.Equal(t, int32(0), unlockedLockout.FailedAttempts)
}

func TestResetFailedAttempts(t *testing.T) {
	lockout := createRandomAccountLockout(t)

	// Increment failed attempts multiple times
	_, err := testQueries.IncrementFailedAttempts(context.Background(), lockout.UserID)
	require.NoError(t, err)
	_, err = testQueries.IncrementFailedAttempts(context.Background(), lockout.UserID)
	require.NoError(t, err)

	// Reset failed attempts
	err = testQueries.ResetFailedAttempts(context.Background(), lockout.UserID)
	require.NoError(t, err)

	// Verify failed attempts are reset
	resetLockout, err := testQueries.GetAccountLockout(context.Background(), lockout.UserID)
	require.NoError(t, err)
	require.Equal(t, int32(0), resetLockout.FailedAttempts)
	require.False(t, resetLockout.LastFailedAttempt.Valid)
}

func TestIsAccountLocked(t *testing.T) {
	lockout := createRandomAccountLockout(t)

	// Initially should not be locked
	result, err := testQueries.IsAccountLocked(context.Background(), lockout.UserID)
	require.NoError(t, err)
	require.False(t, result.Bool)

	// Lock the account
	lockUntil := time.Now().Add(time.Hour)
	err = testQueries.LockAccount(context.Background(), LockAccountParams{
		UserID:      lockout.UserID,
		LockedUntil: sql.NullTime{Time: lockUntil, Valid: true},
	})
	require.NoError(t, err)

	// Should now be locked
	result, err = testQueries.IsAccountLocked(context.Background(), lockout.UserID)
	require.NoError(t, err)
	require.True(t, result.Bool)

	// Lock with past time (expired lock)
	pastTime := time.Now().Add(-time.Hour)
	err = testQueries.LockAccount(context.Background(), LockAccountParams{
		UserID:      lockout.UserID,
		LockedUntil: sql.NullTime{Time: pastTime, Valid: true},
	})
	require.NoError(t, err)

	// Should not be locked anymore
	result, err = testQueries.IsAccountLocked(context.Background(), lockout.UserID)
	require.NoError(t, err)
	require.False(t, result.Bool)
}

func TestUnlockExpiredAccounts(t *testing.T) {
	// Create multiple lockouts
	lockout1 := createRandomAccountLockout(t)
	lockout2 := createRandomAccountLockout(t)
	lockout3 := createRandomAccountLockout(t)

	// Lock accounts with different expiry times
	pastTime := time.Now().Add(-time.Hour)
	futureTime := time.Now().Add(time.Hour)

	// Lock account 1 with expired time
	err := testQueries.LockAccount(context.Background(), LockAccountParams{
		UserID:      lockout1.UserID,
		LockedUntil: sql.NullTime{Time: pastTime, Valid: true},
	})
	require.NoError(t, err)

	// Lock account 2 with future time
	err = testQueries.LockAccount(context.Background(), LockAccountParams{
		UserID:      lockout2.UserID,
		LockedUntil: sql.NullTime{Time: futureTime, Valid: true},
	})
	require.NoError(t, err)

	// Leave account 3 unlocked

	// Run unlock expired accounts
	err = testQueries.UnlockExpiredAccounts(context.Background())
	require.NoError(t, err)

	// Check results
	result1, err := testQueries.GetAccountLockout(context.Background(), lockout1.UserID)
	require.NoError(t, err)
	require.False(t, result1.LockedUntil.Valid) // Should be unlocked
	require.Equal(t, int32(0), result1.FailedAttempts)

	result2, err := testQueries.GetAccountLockout(context.Background(), lockout2.UserID)
	require.NoError(t, err)
	require.True(t, result2.LockedUntil.Valid) // Should still be locked

	result3, err := testQueries.GetAccountLockout(context.Background(), lockout3.UserID)
	require.NoError(t, err)
	require.False(t, result3.LockedUntil.Valid) // Should remain unlocked
}
