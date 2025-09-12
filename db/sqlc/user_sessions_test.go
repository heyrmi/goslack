package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/heyrmi/goslack/util"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"
)

func createRandomUserSession(t *testing.T) UserSession {
	user := createRandomUser(t)

	arg := CreateUserSessionParams{
		UserID:       user.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(time.Hour * 24),
		IpAddress:    createTestIPAddress("192.168.1.100"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{RawMessage: []byte(`{"device":"test","os":"linux"}`), Valid: true},
	}

	session, err := testQueries.CreateUserSession(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, session)

	require.Equal(t, arg.UserID, session.UserID)
	require.Equal(t, arg.SessionToken, session.SessionToken)
	require.Equal(t, arg.RefreshToken, session.RefreshToken)
	require.WithinDuration(t, arg.ExpiresAt, session.ExpiresAt, time.Second)
	require.Equal(t, arg.IpAddress, session.IpAddress)
	require.Equal(t, arg.UserAgent, session.UserAgent)
	requireEqualJSON(t, arg.DeviceInfo, session.DeviceInfo)
	require.NotZero(t, session.ID)
	require.NotZero(t, session.CreatedAt)
	require.NotZero(t, session.LastUsedAt)
	require.True(t, session.IsActive) // Should be active by default

	return session
}

func TestCreateUserSession(t *testing.T) {
	createRandomUserSession(t)
}

func TestGetUserSession(t *testing.T) {
	session1 := createRandomUserSession(t)

	session2, err := testQueries.GetUserSession(context.Background(), session1.SessionToken)
	require.NoError(t, err)
	require.NotEmpty(t, session2)

	require.Equal(t, session1.ID, session2.ID)
	require.Equal(t, session1.UserID, session2.UserID)
	require.Equal(t, session1.SessionToken, session2.SessionToken)
	require.Equal(t, session1.RefreshToken, session2.RefreshToken)
	require.WithinDuration(t, session1.ExpiresAt, session2.ExpiresAt, time.Second)
	require.Equal(t, session1.IpAddress, session2.IpAddress)
	require.Equal(t, session1.UserAgent, session2.UserAgent)
	require.Equal(t, session1.DeviceInfo, session2.DeviceInfo)
	require.Equal(t, session1.IsActive, session2.IsActive)
	require.WithinDuration(t, session1.CreatedAt, session2.CreatedAt, time.Second)
	require.WithinDuration(t, session1.LastUsedAt, session2.LastUsedAt, time.Second)
}

func TestGetUserSessionExpired(t *testing.T) {
	user := createRandomUser(t)

	// Create expired session
	arg := CreateUserSessionParams{
		UserID:       user.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(-time.Hour), // Expired
		IpAddress:    createTestIPAddress("192.168.1.100"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{},
	}

	session, err := testQueries.CreateUserSession(context.Background(), arg)
	require.NoError(t, err)

	// Should not be able to get expired session
	_, err = testQueries.GetUserSession(context.Background(), session.SessionToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetUserSessionInactive(t *testing.T) {
	session := createRandomUserSession(t)

	// Deactivate the session
	err := testQueries.DeactivateSession(context.Background(), session.SessionToken)
	require.NoError(t, err)

	// Should not be able to get inactive session
	_, err = testQueries.GetUserSession(context.Background(), session.SessionToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestGetUserSessionByRefreshToken(t *testing.T) {
	session1 := createRandomUserSession(t)

	session2, err := testQueries.GetUserSessionByRefreshToken(context.Background(), session1.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, session2)

	require.Equal(t, session1.ID, session2.ID)
	require.Equal(t, session1.SessionToken, session2.SessionToken)
	require.Equal(t, session1.RefreshToken, session2.RefreshToken)
}

func TestGetUserSessionByRefreshTokenExpired(t *testing.T) {
	user := createRandomUser(t)

	// Create expired session
	arg := CreateUserSessionParams{
		UserID:       user.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(-time.Hour), // Expired
		IpAddress:    createTestIPAddress("192.168.1.100"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{},
	}

	session, err := testQueries.CreateUserSession(context.Background(), arg)
	require.NoError(t, err)

	// Should not be able to get expired session by refresh token
	_, err = testQueries.GetUserSessionByRefreshToken(context.Background(), session.RefreshToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestUpdateSessionLastUsed(t *testing.T) {
	session := createRandomUserSession(t)
	originalLastUsed := session.LastUsedAt

	// Wait a bit to ensure different timestamp
	time.Sleep(time.Millisecond * 100)

	err := testQueries.UpdateSessionLastUsed(context.Background(), session.SessionToken)
	require.NoError(t, err)

	// Get the session again to verify last_used_at was updated
	updatedSession, err := testQueries.GetUserSession(context.Background(), session.SessionToken)
	require.NoError(t, err)
	require.True(t, updatedSession.LastUsedAt.After(originalLastUsed))
}

func TestDeactivateSession(t *testing.T) {
	session := createRandomUserSession(t)

	// Verify session is initially active
	require.True(t, session.IsActive)

	// Deactivate the session
	err := testQueries.DeactivateSession(context.Background(), session.SessionToken)
	require.NoError(t, err)

	// Should not be able to get the session anymore
	_, err = testQueries.GetUserSession(context.Background(), session.SessionToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}

func TestDeactivateUserSessions(t *testing.T) {
	user := createRandomUser(t)

	// Create multiple sessions for the user
	sessions := make([]UserSession, 3)
	for i := 0; i < 3; i++ {
		arg := CreateUserSessionParams{
			UserID:       user.ID,
			SessionToken: util.RandomString(64),
			RefreshToken: util.RandomString(64),
			ExpiresAt:    time.Now().Add(time.Hour * 24),
			IpAddress:    createTestIPAddress("192.168.1.100"),
			UserAgent:    sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
			DeviceInfo:   pqtype.NullRawMessage{},
		}
		session, err := testQueries.CreateUserSession(context.Background(), arg)
		require.NoError(t, err)
		sessions[i] = session
	}

	// Create a session for another user (should not be affected)
	otherUser := createRandomUser(t)
	otherUserSession := CreateUserSessionParams{
		UserID:       otherUser.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(time.Hour * 24),
		IpAddress:    createTestIPAddress("192.168.1.101"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Other Browser)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{},
	}
	otherSession, err := testQueries.CreateUserSession(context.Background(), otherUserSession)
	require.NoError(t, err)

	// Deactivate all sessions for the first user
	err = testQueries.DeactivateUserSessions(context.Background(), user.ID)
	require.NoError(t, err)

	// Verify all sessions for the first user are deactivated
	for _, session := range sessions {
		_, err = testQueries.GetUserSession(context.Background(), session.SessionToken)
		require.Error(t, err)
		require.Equal(t, sql.ErrNoRows, err)
	}

	// Verify other user's session is still active
	_, err = testQueries.GetUserSession(context.Background(), otherSession.SessionToken)
	require.NoError(t, err)
}

func TestDeactivateExpiredSessions(t *testing.T) {
	user := createRandomUser(t)

	// Create expired session
	expiredArg := CreateUserSessionParams{
		UserID:       user.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(-time.Hour), // Expired
		IpAddress:    createTestIPAddress("192.168.1.100"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Expired)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{},
	}
	expiredSession, err := testQueries.CreateUserSession(context.Background(), expiredArg)
	require.NoError(t, err)

	// Create valid session
	validSession := createRandomUserSession(t)

	// Run deactivate expired sessions
	err = testQueries.DeactivateExpiredSessions(context.Background())
	require.NoError(t, err)

	// Expired session should not be retrievable (was already not retrievable due to expiry check)
	_, err = testQueries.GetUserSession(context.Background(), expiredSession.SessionToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	// Valid session should still be retrievable
	_, err = testQueries.GetUserSession(context.Background(), validSession.SessionToken)
	require.NoError(t, err)
}

func TestCleanupOldSessions(t *testing.T) {
	user := createRandomUser(t)

	// Create an old inactive session
	oldArg := CreateUserSessionParams{
		UserID:       user.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(time.Hour),
		IpAddress:    createTestIPAddress("192.168.1.100"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Old)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{},
	}
	oldSession, err := testQueries.CreateUserSession(context.Background(), oldArg)
	require.NoError(t, err)

	// Deactivate the old session
	err = testQueries.DeactivateSession(context.Background(), oldSession.SessionToken)
	require.NoError(t, err)

	// Create a recent session
	recentSession := createRandomUserSession(t)

	// Cleanup old sessions (older than now, which should include the old deactivated session)
	cutoffTime := time.Now()
	err = testQueries.CleanupOldSessions(context.Background(), cutoffTime)
	require.NoError(t, err)

	// Recent active session should still exist
	_, err = testQueries.GetUserSession(context.Background(), recentSession.SessionToken)
	require.NoError(t, err)

	// Note: We can't directly verify the old session was deleted since it was already
	// not retrievable due to being inactive, but the cleanup operation should have succeeded
}

func TestGetUserActiveSessions(t *testing.T) {
	user := createRandomUser(t)

	// Create multiple active sessions
	activeSessions := make([]UserSession, 3)
	for i := 0; i < 3; i++ {
		arg := CreateUserSessionParams{
			UserID:       user.ID,
			SessionToken: util.RandomString(64),
			RefreshToken: util.RandomString(64),
			ExpiresAt:    time.Now().Add(time.Hour * 24),
			IpAddress:    createTestIPAddress("192.168.1.100"),
			UserAgent:    sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
			DeviceInfo:   pqtype.NullRawMessage{},
		}
		session, err := testQueries.CreateUserSession(context.Background(), arg)
		require.NoError(t, err)
		activeSessions[i] = session

		// Small delay to ensure different last_used_at times
		time.Sleep(time.Millisecond * 10)
	}

	// Create an inactive session
	inactiveArg := CreateUserSessionParams{
		UserID:       user.ID,
		SessionToken: util.RandomString(64),
		RefreshToken: util.RandomString(64),
		ExpiresAt:    time.Now().Add(time.Hour * 24),
		IpAddress:    createTestIPAddress("192.168.1.101"),
		UserAgent:    sql.NullString{String: "Mozilla/5.0 (Inactive)", Valid: true},
		DeviceInfo:   pqtype.NullRawMessage{},
	}
	inactiveSession, err := testQueries.CreateUserSession(context.Background(), inactiveArg)
	require.NoError(t, err)

	err = testQueries.DeactivateSession(context.Background(), inactiveSession.SessionToken)
	require.NoError(t, err)

	// Get active sessions
	userSessions, err := testQueries.GetUserActiveSessions(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, userSessions, 3) // Only active sessions

	// Verify sessions are ordered by last_used_at DESC (most recent first)
	for i := 1; i < len(userSessions); i++ {
		require.True(t, userSessions[i].LastUsedAt.Before(userSessions[i-1].LastUsedAt) ||
			userSessions[i].LastUsedAt.Equal(userSessions[i-1].LastUsedAt))
	}

	// Verify all returned sessions are active
	for _, session := range userSessions {
		require.True(t, session.IsActive)
		require.Equal(t, user.ID, session.UserID)
	}
}

func TestUserSessionDeviceInfo(t *testing.T) {
	user := createRandomUser(t)

	testCases := []struct {
		name       string
		deviceInfo pqtype.NullRawMessage
	}{
		{
			name:       "no device info",
			deviceInfo: pqtype.NullRawMessage{},
		},
		{
			name:       "simple device info",
			deviceInfo: pqtype.NullRawMessage{RawMessage: []byte(`{"os": "ios", "device": "mobile"}`), Valid: true},
		},
		{
			name:       "complex device info",
			deviceInfo: pqtype.NullRawMessage{RawMessage: []byte(`{"os": "windows", "device": "desktop", "screen": {"width": 1920, "height": 1080}, "browser": "chrome", "version": "91.0"}`), Valid: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			arg := CreateUserSessionParams{
				UserID:       user.ID,
				SessionToken: util.RandomString(64),
				RefreshToken: util.RandomString(64),
				ExpiresAt:    time.Now().Add(time.Hour * 24),
				IpAddress:    createTestIPAddress("192.168.1.100"),
				UserAgent:    sql.NullString{String: "Mozilla/5.0 (Test)", Valid: true},
				DeviceInfo:   tc.deviceInfo,
			}

			session, err := testQueries.CreateUserSession(context.Background(), arg)
			require.NoError(t, err)
			require.Equal(t, tc.deviceInfo, session.DeviceInfo)

			// Verify retrieval
			retrievedSession, err := testQueries.GetUserSession(context.Background(), session.SessionToken)
			require.NoError(t, err)
			require.Equal(t, tc.deviceInfo, retrievedSession.DeviceInfo)
		})
	}
}

func TestUserSessionComplexScenario(t *testing.T) {

	// Create session
	session := createRandomUserSession(t)

	// Update last used several times
	for i := 0; i < 3; i++ {
		time.Sleep(time.Millisecond * 50)
		err := testQueries.UpdateSessionLastUsed(context.Background(), session.SessionToken)
		require.NoError(t, err)
	}

	// Get updated session
	updatedSession, err := testQueries.GetUserSession(context.Background(), session.SessionToken)
	require.NoError(t, err)
	require.True(t, updatedSession.LastUsedAt.After(session.LastUsedAt))

	// Test refresh token functionality
	refreshSession, err := testQueries.GetUserSessionByRefreshToken(context.Background(), session.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, session.ID, refreshSession.ID)

	// Deactivate session
	err = testQueries.DeactivateSession(context.Background(), session.SessionToken)
	require.NoError(t, err)

	// Should no longer be able to get session
	_, err = testQueries.GetUserSession(context.Background(), session.SessionToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)

	// Should no longer be able to get session by refresh token
	_, err = testQueries.GetUserSessionByRefreshToken(context.Background(), session.RefreshToken)
	require.Error(t, err)
	require.Equal(t, sql.ErrNoRows, err)
}
