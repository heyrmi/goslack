package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func createRandomUser2FA(t *testing.T) User2fa {
	user := createRandomUser(t)
	secret := util.RandomString(32)
	backupCodes := []string{
		util.RandomString(8),
		util.RandomString(8),
		util.RandomString(8),
		util.RandomString(8),
		util.RandomString(8),
	}

	arg := CreateUser2FAParams{
		UserID:      user.ID,
		Secret:      secret,
		BackupCodes: backupCodes,
	}

	user2fa, err := testQueries.CreateUser2FA(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2fa)

	require.Equal(t, arg.UserID, user2fa.UserID)
	require.Equal(t, arg.Secret, user2fa.Secret)
	require.Equal(t, arg.BackupCodes, user2fa.BackupCodes)
	require.False(t, user2fa.Enabled)          // Should be disabled by default
	require.False(t, user2fa.VerifiedAt.Valid) // Should not be verified yet
	require.NotZero(t, user2fa.ID)
	require.NotZero(t, user2fa.CreatedAt)
	require.NotZero(t, user2fa.UpdatedAt)

	return user2fa
}

func TestCreateUser2FA(t *testing.T) {
	createRandomUser2FA(t)
}

func TestGetUser2FA(t *testing.T) {
	user2fa1 := createRandomUser2FA(t)

	user2fa2, err := testQueries.GetUser2FA(context.Background(), user2fa1.UserID)
	require.NoError(t, err)
	require.NotEmpty(t, user2fa2)

	require.Equal(t, user2fa1.ID, user2fa2.ID)
	require.Equal(t, user2fa1.UserID, user2fa2.UserID)
	require.Equal(t, user2fa1.Secret, user2fa2.Secret)
	require.Equal(t, user2fa1.BackupCodes, user2fa2.BackupCodes)
	require.Equal(t, user2fa1.Enabled, user2fa2.Enabled)
	require.Equal(t, user2fa1.VerifiedAt, user2fa2.VerifiedAt)
	require.WithinDuration(t, user2fa1.CreatedAt, user2fa2.CreatedAt, time.Second)
	require.WithinDuration(t, user2fa1.UpdatedAt, user2fa2.UpdatedAt, time.Second)
}

func TestGetUser2FANotFound(t *testing.T) {
	user := createRandomUser(t)

	_, err := testQueries.GetUser2FA(context.Background(), user.ID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestEnableUser2FA(t *testing.T) {
	user2fa := createRandomUser2FA(t)

	// Initially should be disabled
	require.False(t, user2fa.Enabled)
	require.False(t, user2fa.VerifiedAt.Valid)

	// Enable 2FA
	err := testQueries.EnableUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)

	// Verify it's enabled
	updatedUser2FA, err := testQueries.GetUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)
	require.True(t, updatedUser2FA.Enabled)
	require.True(t, updatedUser2FA.VerifiedAt.Valid)
	require.WithinDuration(t, time.Now(), updatedUser2FA.VerifiedAt.Time, time.Second*5)
	require.True(t, updatedUser2FA.UpdatedAt.After(user2fa.UpdatedAt))
}

func TestDisableUser2FA(t *testing.T) {
	user2fa := createRandomUser2FA(t)

	// First enable it
	err := testQueries.EnableUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)

	// Verify it's enabled
	enabledUser2FA, err := testQueries.GetUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)
	require.True(t, enabledUser2FA.Enabled)

	// Now disable it
	err = testQueries.DisableUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)

	// Verify it's disabled
	disabledUser2FA, err := testQueries.GetUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)
	require.False(t, disabledUser2FA.Enabled)
	require.True(t, disabledUser2FA.UpdatedAt.After(enabledUser2FA.UpdatedAt))
	// Note: VerifiedAt should remain as it was when enabled
	require.Equal(t, enabledUser2FA.VerifiedAt, disabledUser2FA.VerifiedAt)
}

func TestUpdateUser2FABackupCodes(t *testing.T) {
	user2fa := createRandomUser2FA(t)

	newBackupCodes := []string{
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
	}

	err := testQueries.UpdateUser2FABackupCodes(context.Background(), UpdateUser2FABackupCodesParams{
		UserID:      user2fa.UserID,
		BackupCodes: newBackupCodes,
	})
	require.NoError(t, err)

	// Verify backup codes are updated
	updatedUser2FA, err := testQueries.GetUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)
	require.Equal(t, newBackupCodes, updatedUser2FA.BackupCodes)
	require.True(t, updatedUser2FA.UpdatedAt.After(user2fa.UpdatedAt))

	// Verify other fields remain unchanged
	require.Equal(t, user2fa.UserID, updatedUser2FA.UserID)
	require.Equal(t, user2fa.Secret, updatedUser2FA.Secret)
	require.Equal(t, user2fa.Enabled, updatedUser2FA.Enabled)
	require.Equal(t, user2fa.VerifiedAt, updatedUser2FA.VerifiedAt)
}

func TestDeleteUser2FA(t *testing.T) {
	user2fa := createRandomUser2FA(t)

	// Delete 2FA
	err := testQueries.DeleteUser2FA(context.Background(), user2fa.UserID)
	require.NoError(t, err)

	// Verify it's deleted
	_, err = testQueries.GetUser2FA(context.Background(), user2fa.UserID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestUser2FALifecycle(t *testing.T) {
	user := createRandomUser(t)
	secret := util.RandomString(32)
	initialBackupCodes := []string{
		util.RandomString(8),
		util.RandomString(8),
		util.RandomString(8),
		util.RandomString(8),
		util.RandomString(8),
	}

	// Step 1: Create 2FA setup
	user2fa, err := testQueries.CreateUser2FA(context.Background(), CreateUser2FAParams{
		UserID:      user.ID,
		Secret:      secret,
		BackupCodes: initialBackupCodes,
	})
	require.NoError(t, err)
	require.False(t, user2fa.Enabled)
	require.False(t, user2fa.VerifiedAt.Valid)

	// Step 2: Enable 2FA (user has verified their setup)
	err = testQueries.EnableUser2FA(context.Background(), user.ID)
	require.NoError(t, err)

	enabledUser2FA, err := testQueries.GetUser2FA(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, enabledUser2FA.Enabled)
	require.True(t, enabledUser2FA.VerifiedAt.Valid)

	// Step 3: Update backup codes (user regenerated them)
	newBackupCodes := []string{
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
		util.RandomString(10),
	}

	err = testQueries.UpdateUser2FABackupCodes(context.Background(), UpdateUser2FABackupCodesParams{
		UserID:      user.ID,
		BackupCodes: newBackupCodes,
	})
	require.NoError(t, err)

	updatedUser2FA, err := testQueries.GetUser2FA(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, newBackupCodes, updatedUser2FA.BackupCodes)
	require.True(t, updatedUser2FA.Enabled) // Should still be enabled

	// Step 4: Temporarily disable 2FA
	err = testQueries.DisableUser2FA(context.Background(), user.ID)
	require.NoError(t, err)

	disabledUser2FA, err := testQueries.GetUser2FA(context.Background(), user.ID)
	require.NoError(t, err)
	require.False(t, disabledUser2FA.Enabled)
	require.Equal(t, newBackupCodes, disabledUser2FA.BackupCodes) // Backup codes should remain

	// Step 5: Re-enable 2FA
	err = testQueries.EnableUser2FA(context.Background(), user.ID)
	require.NoError(t, err)

	reEnabledUser2FA, err := testQueries.GetUser2FA(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, reEnabledUser2FA.Enabled)
	require.True(t, reEnabledUser2FA.VerifiedAt.Valid)

	// Step 6: Completely remove 2FA
	err = testQueries.DeleteUser2FA(context.Background(), user.ID)
	require.NoError(t, err)

	_, err = testQueries.GetUser2FA(context.Background(), user.ID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestUser2FABackupCodesVariations(t *testing.T) {
	testCases := []struct {
		name        string
		backupCodes []string
	}{
		{
			name:        "standard 5 codes",
			backupCodes: []string{"12345678", "87654321", "11111111", "22222222", "33333333"},
		},
		{
			name:        "10 codes",
			backupCodes: []string{"code1", "code2", "code3", "code4", "code5", "code6", "code7", "code8", "code9", "code10"},
		},
		{
			name:        "empty backup codes",
			backupCodes: []string{},
		},
		{
			name:        "single backup code",
			backupCodes: []string{"onlycode"},
		},
		{
			name:        "codes with special characters",
			backupCodes: []string{"ABC-123", "XYZ_789", "!@#$%^&*", "code.with.dots", "code-with-dashes"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := createRandomUser(t)
			secret := util.RandomString(32)

			user2fa, err := testQueries.CreateUser2FA(context.Background(), CreateUser2FAParams{
				UserID:      user.ID,
				Secret:      secret,
				BackupCodes: tc.backupCodes,
			})
			require.NoError(t, err)
			require.Equal(t, tc.backupCodes, user2fa.BackupCodes)

			// Verify retrieval
			retrievedUser2FA, err := testQueries.GetUser2FA(context.Background(), user.ID)
			require.NoError(t, err)
			require.Equal(t, tc.backupCodes, retrievedUser2FA.BackupCodes)
		})
	}
}

func TestUser2FAMultipleUsers(t *testing.T) {
	// Create multiple users with 2FA
	users := make([]User, 3)
	user2fas := make([]User2fa, 3)

	for i := 0; i < 3; i++ {
		users[i] = createRandomUser(t)

		user2fas[i], _ = testQueries.CreateUser2FA(context.Background(), CreateUser2FAParams{
			UserID:      users[i].ID,
			Secret:      util.RandomString(32),
			BackupCodes: []string{util.RandomString(8), util.RandomString(8)},
		})
	}

	// Enable 2FA for user 0 and 2
	err := testQueries.EnableUser2FA(context.Background(), users[0].ID)
	require.NoError(t, err)
	err = testQueries.EnableUser2FA(context.Background(), users[2].ID)
	require.NoError(t, err)

	// Verify each user's 2FA state
	user0_2fa, err := testQueries.GetUser2FA(context.Background(), users[0].ID)
	require.NoError(t, err)
	require.True(t, user0_2fa.Enabled)

	user1_2fa, err := testQueries.GetUser2FA(context.Background(), users[1].ID)
	require.NoError(t, err)
	require.False(t, user1_2fa.Enabled)

	user2_2fa, err := testQueries.GetUser2FA(context.Background(), users[2].ID)
	require.NoError(t, err)
	require.True(t, user2_2fa.Enabled)

	// Delete 2FA for user 1
	err = testQueries.DeleteUser2FA(context.Background(), users[1].ID)
	require.NoError(t, err)

	// Verify user 1's 2FA is deleted but others remain
	_, err = testQueries.GetUser2FA(context.Background(), users[1].ID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	_, err = testQueries.GetUser2FA(context.Background(), users[0].ID)
	require.NoError(t, err)

	_, err = testQueries.GetUser2FA(context.Background(), users[2].ID)
	require.NoError(t, err)
}

func TestUser2FAOperationsOnNonExistentUser(t *testing.T) {
	nonExistentUserID := int64(99999)

	// Try to get 2FA for non-existent user
	_, err := testQueries.GetUser2FA(context.Background(), nonExistentUserID)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	// Try to enable 2FA for non-existent user
	err = testQueries.EnableUser2FA(context.Background(), nonExistentUserID)
	require.NoError(t, err) // Should not error but should not affect anything

	// Try to disable 2FA for non-existent user
	err = testQueries.DisableUser2FA(context.Background(), nonExistentUserID)
	require.NoError(t, err) // Should not error but should not affect anything

	// Try to update backup codes for non-existent user
	err = testQueries.UpdateUser2FABackupCodes(context.Background(), UpdateUser2FABackupCodesParams{
		UserID:      nonExistentUserID,
		BackupCodes: []string{"code1", "code2"},
	})
	require.NoError(t, err) // Should not error but should not affect anything

	// Try to delete 2FA for non-existent user
	err = testQueries.DeleteUser2FA(context.Background(), nonExistentUserID)
	require.NoError(t, err) // Should not error but should not affect anything
}

func TestUser2FASecretHandling(t *testing.T) {
	user := createRandomUser(t)

	// Test with different secret formats
	testSecrets := []string{
		"JBSWY3DPEHPK3PXP", // Base32 encoded
		util.RandomString(16),
		util.RandomString(32),
		util.RandomString(64),
		"secret_with_underscores_123",
		"secret-with-dashes-456",
	}

	for i, secret := range testSecrets {
		t.Run("secret_"+string(rune(i)), func(t *testing.T) {
			// Delete any existing 2FA first
			_ = testQueries.DeleteUser2FA(context.Background(), user.ID)

			user2fa, err := testQueries.CreateUser2FA(context.Background(), CreateUser2FAParams{
				UserID:      user.ID,
				Secret:      secret,
				BackupCodes: []string{"backup1", "backup2"},
			})
			require.NoError(t, err)
			require.Equal(t, secret, user2fa.Secret)

			// Verify secret is stored and retrieved correctly
			retrievedUser2FA, err := testQueries.GetUser2FA(context.Background(), user.ID)
			require.NoError(t, err)
			require.Equal(t, secret, retrievedUser2FA.Secret)
		})
	}
}
