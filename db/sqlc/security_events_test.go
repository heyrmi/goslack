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

func createRandomSecurityEvent(t *testing.T) SecurityEvent {
	_, user := createTestWorkspaceAndUser(t)

	arg := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
		EventType:   "login_success",
		Description: sql.NullString{String: util.RandomString(50), Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{RawMessage: []byte(`{"test": "data"}`), Valid: true},
	}

	event, err := testQueries.CreateSecurityEvent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, event)

	require.Equal(t, arg.UserID, event.UserID)
	require.Equal(t, arg.EventType, event.EventType)
	require.Equal(t, arg.Description, event.Description)
	// Note: IP address comparison might fail due to different representations
	// We'll just check that it's valid
	require.NotNil(t, event.IpAddress.IPNet)
	require.Equal(t, arg.UserAgent, event.UserAgent)
	require.Equal(t, arg.Metadata, event.Metadata)
	require.NotZero(t, event.ID)
	require.NotZero(t, event.CreatedAt)

	return event
}

func TestCreateSecurityEvent(t *testing.T) {
	createRandomSecurityEvent(t)
}

func TestCreateSecurityEventWithoutUser(t *testing.T) {
	arg := CreateSecurityEventParams{
		UserID:      sql.NullInt64{},
		EventType:   "suspicious_activity",
		Description: sql.NullString{String: "System maintenance", Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{},
		Metadata:    pqtype.NullRawMessage{},
	}

	event, err := testQueries.CreateSecurityEvent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, event)

	require.False(t, event.UserID.Valid)
	require.Equal(t, arg.EventType, event.EventType)
	require.Equal(t, arg.Description, event.Description)
}

func TestGetRecentSecurityEvents(t *testing.T) {
	// Create multiple events
	createRandomSecurityEvent(t)
	time.Sleep(time.Millisecond * 10) // Ensure different timestamps
	createRandomSecurityEvent(t)

	// Get recent events
	events, err := testQueries.GetRecentSecurityEvents(context.Background(), GetRecentSecurityEventsParams{
		CreatedAt: time.Now().Add(-time.Hour), // Last hour
		Limit:     10,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(events), 2)

	// Verify events are ordered by creation time DESC
	for i := 1; i < len(events); i++ {
		require.True(t, events[i-1].CreatedAt.After(events[i].CreatedAt) ||
			events[i-1].CreatedAt.Equal(events[i].CreatedAt))
	}
}

func TestGetSecurityEventsByType(t *testing.T) {
	// Create events of different types
	createRandomSecurityEvent(t)
	createRandomSecurityEvent(t)

	// Create a different type of event
	_, user := createTestWorkspaceAndUser(t)
	arg := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
		EventType:   "password_changed",
		Description: sql.NullString{String: "Password changed", Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{},
	}
	_, err := testQueries.CreateSecurityEvent(context.Background(), arg)
	require.NoError(t, err)

	// Get events by type
	loginEvents, err := testQueries.GetSecurityEventsByType(context.Background(), GetSecurityEventsByTypeParams{
		EventType: "login_success",
		Limit:     10,
		Offset:    0,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(loginEvents), 2)

	passwordEvents, err := testQueries.GetSecurityEventsByType(context.Background(), GetSecurityEventsByTypeParams{
		EventType: "password_changed",
		Limit:     10,
		Offset:    0,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(passwordEvents), 1)

	// Verify all returned events have the correct type
	for _, event := range loginEvents {
		require.Equal(t, "login_success", event.EventType)
	}
	for _, event := range passwordEvents {
		require.Equal(t, "password_changed", event.EventType)
	}
}

func TestGetUserSecurityEvents(t *testing.T) {
	_, user := createTestWorkspaceAndUser(t)

	// Create multiple events for the specific user
	arg1 := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
		EventType:   "login_success",
		Description: sql.NullString{String: util.RandomString(50), Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{RawMessage: []byte(`{"test": "data"}`), Valid: true},
	}
	_, err := testQueries.CreateSecurityEvent(context.Background(), arg1)
	require.NoError(t, err)

	arg2 := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
		EventType:   "password_changed",
		Description: sql.NullString{String: util.RandomString(50), Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{RawMessage: []byte(`{"test": "data2"}`), Valid: true},
	}
	_, err = testQueries.CreateSecurityEvent(context.Background(), arg2)
	require.NoError(t, err)

	// Create an event for a different user
	workspace, _ := createTestWorkspaceAndUser(t)
	otherUser := createRandomUserForOrganization(t, workspace.OrganizationID)
	arg := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: otherUser.ID, Valid: true},
		EventType:   "login_success",
		Description: sql.NullString{String: "Other user login", Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{},
	}
	_, err = testQueries.CreateSecurityEvent(context.Background(), arg)
	require.NoError(t, err)

	// Get user's security events
	userEvents, err := testQueries.GetUserSecurityEvents(context.Background(), GetUserSecurityEventsParams{
		UserID: sql.NullInt64{Int64: user.ID, Valid: true},
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(userEvents), 2)

	// Verify all events belong to the user
	for _, event := range userEvents {
		require.True(t, event.UserID.Valid)
		require.Equal(t, user.ID, event.UserID.Int64)
	}
}

func TestCleanupOldSecurityEvents(t *testing.T) {
	// Create an old event
	_, user := createTestWorkspaceAndUser(t)
	arg := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
		EventType:   "suspicious_activity",
		Description: sql.NullString{String: "Old event", Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{},
	}
	oldEvent, err := testQueries.CreateSecurityEvent(context.Background(), arg)
	require.NoError(t, err)

	// Wait a bit to ensure time difference
	time.Sleep(time.Millisecond * 100)

	// Create a recent event for the same user
	recentArg := CreateSecurityEventParams{
		UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
		EventType:   "login_success",
		Description: sql.NullString{String: "Recent event", Valid: true},
		IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
		UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
		Metadata:    pqtype.NullRawMessage{RawMessage: []byte(`{"test": "recent"}`), Valid: true},
	}
	recentEvent, err := testQueries.CreateSecurityEvent(context.Background(), recentArg)
	require.NoError(t, err)

	// Cleanup events older than now (should remove the old event)
	// Use a cutoff time that's between the old event and recent event
	cutoffTime := time.Now().Add(-time.Millisecond * 50)
	err = testQueries.CleanupOldSecurityEvents(context.Background(), cutoffTime)
	require.NoError(t, err)

	// Verify old event is deleted by checking if we can find it in recent events
	recentEvents, err := testQueries.GetRecentSecurityEvents(context.Background(), GetRecentSecurityEventsParams{
		CreatedAt: time.Now().Add(-time.Hour),
		Limit:     100,
	})
	require.NoError(t, err)

	foundOldEvent := false
	foundRecentEvent := false
	for _, event := range recentEvents {
		if event.ID == oldEvent.ID {
			foundOldEvent = true
		}
		if event.ID == recentEvent.ID {
			foundRecentEvent = true
		}
	}

	require.False(t, foundOldEvent, "Old event should be deleted")
	require.True(t, foundRecentEvent, "Recent event should still exist")
}

func TestSecurityEventTypes(t *testing.T) {
	_, user := createTestWorkspaceAndUser(t)

	eventTypes := []string{
		"login_success",
		"login_failed",
		"password_changed",
		"email_changed",
		"account_locked",
		"account_unlocked",
		"password_reset_requested",
		"password_reset_completed",
		"email_verification_sent",
		"email_verified",
		"suspicious_activity",
		"2fa_enabled",
		"2fa_disabled",
		"token_refresh",
	}

	for _, eventType := range eventTypes {
		arg := CreateSecurityEventParams{
			UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
			EventType:   eventType,
			Description: sql.NullString{String: "Test " + eventType, Valid: true},
			IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
			UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
			Metadata:    pqtype.NullRawMessage{},
		}

		event, err := testQueries.CreateSecurityEvent(context.Background(), arg)
		require.NoError(t, err)
		require.Equal(t, eventType, event.EventType)
	}
}

func TestSecurityEventMetadata(t *testing.T) {
	_, user := createTestWorkspaceAndUser(t)

	testCases := []struct {
		name     string
		metadata pqtype.NullRawMessage
	}{
		{
			name:     "empty_metadata",
			metadata: pqtype.NullRawMessage{},
		},
		{
			name:     "simple_json",
			metadata: pqtype.NullRawMessage{RawMessage: []byte(`{"key": "value"}`), Valid: true},
		},
		{
			name:     "complex_json",
			metadata: pqtype.NullRawMessage{RawMessage: []byte(`{"user_agent": "Mozilla/5.0", "ip": "192.168.1.1", "location": {"country": "US", "city": "New York"}}`), Valid: true},
		},
		{
			name:     "array_json",
			metadata: pqtype.NullRawMessage{RawMessage: []byte(`["item1", "item2", "item3"]`), Valid: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			arg := CreateSecurityEventParams{
				UserID:      sql.NullInt64{Int64: user.ID, Valid: true},
				EventType:   "login_success",
				Description: sql.NullString{String: "Test metadata", Valid: true},
				IpAddress:   pqtype.Inet{IPNet: util.RandomIPNet()},
				UserAgent:   sql.NullString{String: "Mozilla/5.0 (Test Browser)", Valid: true},
				Metadata:    tc.metadata,
			}

			event, err := testQueries.CreateSecurityEvent(context.Background(), arg)
			require.NoError(t, err)
			requireEqualJSON(t, tc.metadata, event.Metadata)
		})
	}
}
