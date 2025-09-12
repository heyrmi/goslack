package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/heyrmi/goslack/db/mock"
	db "github.com/heyrmi/goslack/db/sqlc"
	"github.com/heyrmi/goslack/token"
	"github.com/heyrmi/goslack/util"
	"github.com/stretchr/testify/require"
)

func TestAuthService_SendEmailVerification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	config := util.Config{}
	emailService := NewMockEmailService()
	authService := NewAuthService(store, tokenMaker, emailService, config)

	testCases := []struct {
		name        string
		request     EmailVerificationRequest
		buildStubs  func(store *mockdb.MockStore)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "OK",
			request: EmailVerificationRequest{
				Email:     "user@example.com",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), "user@example.com").
					Times(1).
					Return(db.User{
						ID:            1,
						Email:         "user@example.com",
						FirstName:     "John",
						LastName:      "Doe",
						EmailVerified: false,
					}, nil)

				store.EXPECT().
					CreateEmailVerificationToken(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.EmailVerificationToken{
						ID:        1,
						UserID:    1,
						Token:     "test_token",
						ExpiresAt: time.Now().Add(24 * time.Hour),
						CreatedAt: time.Now(),
					}, nil)

				store.EXPECT().
					CreateSecurityEvent(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.SecurityEvent{}, nil)
			},
			checkResult: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "UserNotFound",
			request: EmailVerificationRequest{
				Email:     "nonexistent@example.com",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), "nonexistent@example.com").
					Times(1).
					Return(db.User{}, sql.ErrNoRows)
			},
			checkResult: func(t *testing.T, err error) {
				// Should not return error for security reasons
				require.NoError(t, err)
			},
		},
		{
			name: "EmailAlreadyVerified",
			request: EmailVerificationRequest{
				Email:     "verified@example.com",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), "verified@example.com").
					Times(1).
					Return(db.User{
						ID:            1,
						Email:         "verified@example.com",
						FirstName:     "John",
						LastName:      "Doe",
						EmailVerified: true,
					}, nil)
			},
			checkResult: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "already verified")
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(store)

			err := authService.SendEmailVerification(context.Background(), tc.request)
			tc.checkResult(t, err)
		})
	}
}

func TestAuthService_VerifyEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	config := util.Config{}
	emailService := NewMockEmailService()
	authService := NewAuthService(store, tokenMaker, emailService, config)

	testCases := []struct {
		name        string
		request     VerifyEmailRequest
		buildStubs  func(store *mockdb.MockStore)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "OK",
			request: VerifyEmailRequest{
				Token:     "valid_token",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetEmailVerificationToken(gomock.Any(), "valid_token").
					Times(1).
					Return(db.EmailVerificationToken{
						ID:        1,
						UserID:    1,
						Token:     "valid_token",
						ExpiresAt: time.Now().Add(time.Hour),
						CreatedAt: time.Now(),
						UsedAt:    sql.NullTime{Valid: false},
					}, nil)

				store.EXPECT().
					GetUser(gomock.Any(), int64(1)).
					Times(1).
					Return(db.User{
						ID:            1,
						Email:         "user@example.com",
						FirstName:     "John",
						LastName:      "Doe",
						EmailVerified: false,
					}, nil)

				store.EXPECT().
					UseEmailVerificationToken(gomock.Any(), "valid_token").
					Times(1).
					Return(nil)

				store.EXPECT().
					VerifyUserEmail(gomock.Any(), int64(1)).
					Times(1).
					Return(nil)

				store.EXPECT().
					CreateSecurityEvent(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.SecurityEvent{}, nil)
			},
			checkResult: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "TokenNotFound",
			request: VerifyEmailRequest{
				Token:     "invalid_token",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetEmailVerificationToken(gomock.Any(), "invalid_token").
					Times(1).
					Return(db.EmailVerificationToken{}, sql.ErrNoRows)
			},
			checkResult: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid or expired")
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(store)

			err := authService.VerifyEmail(context.Background(), tc.request)
			tc.checkResult(t, err)
		})
	}
}

func TestAuthService_SendPasswordReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	config := util.Config{}
	emailService := NewMockEmailService()
	authService := NewAuthService(store, tokenMaker, emailService, config)

	testCases := []struct {
		name        string
		request     PasswordResetRequest
		buildStubs  func(store *mockdb.MockStore)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "OK",
			request: PasswordResetRequest{
				Email:     "user@example.com",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					IsAccountLocked(gomock.Any(), int64(1)).
					Times(1).
					Return(sql.NullBool{Bool: false, Valid: true}, nil)

				store.EXPECT().
					GetUserByEmail(gomock.Any(), "user@example.com").
					Times(1).
					Return(db.User{
						ID:        1,
						Email:     "user@example.com",
						FirstName: "John",
						LastName:  "Doe",
					}, nil)

				store.EXPECT().
					DeleteUserPasswordResetTokens(gomock.Any(), int64(1)).
					Times(1).
					Return(nil)

				store.EXPECT().
					CreatePasswordResetToken(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.PasswordResetToken{
						ID:        1,
						UserID:    1,
						Token:     "reset_token",
						ExpiresAt: time.Now().Add(time.Hour),
						CreatedAt: time.Now(),
					}, nil)

				store.EXPECT().
					CreateSecurityEvent(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.SecurityEvent{}, nil)
			},
			checkResult: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "UserNotFound",
			request: PasswordResetRequest{
				Email:     "nonexistent@example.com",
				IPAddress: "127.0.0.1",
				UserAgent: "test-agent",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByEmail(gomock.Any(), "nonexistent@example.com").
					Times(1).
					Return(db.User{}, sql.ErrNoRows)
			},
			checkResult: func(t *testing.T, err error) {
				require.NoError(t, err) // Should not return error for security reasons
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(store)

			err := authService.SendPasswordReset(context.Background(), tc.request)
			tc.checkResult(t, err)
		})
	}
}

func TestAuthService_IsAccountLocked(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mockdb.NewMockStore(ctrl)
	tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	config := util.Config{}
	emailService := NewMockEmailService()
	authService := NewAuthService(store, tokenMaker, emailService, config)

	testCases := []struct {
		name        string
		userID      int64
		buildStubs  func(store *mockdb.MockStore)
		checkResult func(t *testing.T, locked bool, err error)
	}{
		{
			name:   "NotLocked",
			userID: 1,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					IsAccountLocked(gomock.Any(), int64(1)).
					Times(1).
					Return(sql.NullBool{Bool: false, Valid: true}, nil)
			},
			checkResult: func(t *testing.T, locked bool, err error) {
				require.NoError(t, err)
				require.False(t, locked)
			},
		},
		{
			name:   "Locked",
			userID: 1,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					IsAccountLocked(gomock.Any(), int64(1)).
					Times(1).
					Return(sql.NullBool{Bool: true, Valid: true}, nil)
			},
			checkResult: func(t *testing.T, locked bool, err error) {
				require.NoError(t, err)
				require.True(t, locked)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			tc.buildStubs(store)

			locked, err := authService.IsAccountLocked(context.Background(), tc.userID)
			tc.checkResult(t, locked, err)
		})
	}
}
