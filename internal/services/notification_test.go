package service_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repoMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	emailMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendgrid/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewNotificationService(t *testing.T) {
	mockRepo := repoMocks.NewMockNotificationRepository(t)
	mockUserRepo := repoMocks.NewMockUserRepository(t)
	mockEmailService := emailMocks.NewMockEmailService(t)

	svc := service.NewNotificationService(mockRepo, mockUserRepo, mockEmailService)
	assert.NotNil(t, svc)
}

func TestSendEmail(t *testing.T) {
	ctx := t.Context()
	mockRepo := repoMocks.NewMockNotificationRepository(t)
	mockUserRepo := repoMocks.NewMockUserRepository(t)
	mockEmailService := emailMocks.NewMockEmailService(t)
	svc := service.NewNotificationService(mockRepo, mockUserRepo, mockEmailService)

	testEmail := "test@example.com"
	testSubject := "Test Subject"
	testContent := "Test Content"
	testMetadata := map[string]string{"key": "value"}
	metadataBytes, err := json.Marshal(testMetadata)
	if err != nil {
		t.Fatalf("failed to marshal testMetadata: %v", err)
	}

	req := &models.EmailNotificationRequest{
		To:       testEmail,
		Subject:  testSubject,
		Content:  testContent,
		Metadata: testMetadata,
	}

	user := &models.User{ID: uuid.New(), Email: testEmail}
	dbErr := errors.New("database error")
	sendErr := errors.New("sendgrid error")
	notFoundErr := errors.New("not found")

	t.Run("Success - Send Email", func(t *testing.T) {
		mockUserRepo.EXPECT().GetUserByEmail(ctx, testEmail).Return(user, nil).Once()
		mockRepo.EXPECT().CreateNotification(ctx, mock.MatchedBy(func(n *models.Notification) bool {
			return n.Recipient == testEmail && n.Subject == testSubject && n.Status == models.StatusPending && string(n.Metadata) == string(metadataBytes)
		})).Return(nil).Once()
		mockEmailService.EXPECT().Send(ctx, req).Return(nil).Once()
		mockRepo.EXPECT().UpdateNotificationStatus(ctx, mock.AnythingOfType("uuid.UUID"), models.StatusSent, "").Return(nil).Once()

		resp, err := svc.SendEmail(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, testEmail, resp.Recipient)
		assert.Equal(t, models.NotificationTypeEmail, resp.Type)
		assert.Equal(t, models.StatusSent, resp.Status)
		assert.NotEqual(t, uuid.Nil, resp.ID)

		mockRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("Success without metadata", func(t *testing.T) {
		reqNoMeta := &models.EmailNotificationRequest{
			To:      testEmail,
			Subject: testSubject,
			Content: testContent,
		}

		mockUserRepo.EXPECT().GetUserByEmail(ctx, testEmail).Return(user, nil).Once()
		mockRepo.EXPECT().CreateNotification(ctx, mock.MatchedBy(func(n *models.Notification) bool {
			return n.Recipient == testEmail && n.Subject == testSubject && n.Status == models.StatusPending && n.Metadata == nil
		})).Return(nil).Once()
		mockEmailService.EXPECT().Send(ctx, reqNoMeta).Return(nil).Once()
		mockRepo.EXPECT().UpdateNotificationStatus(ctx, mock.AnythingOfType("uuid.UUID"), models.StatusSent, "").Return(nil).Once()

		resp, err := svc.SendEmail(ctx, reqNoMeta)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, testEmail, resp.Recipient)
		mockRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("Failure - User Not Found", func(t *testing.T) {
		mockUserRepo.EXPECT().GetUserByEmail(ctx, testEmail).Return(nil, notFoundErr).Once()

		resp, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		assert.ErrorIs(t, err, notFoundErr)
		mockRepo.AssertNotCalled(t, "CreateNotification")
		mockEmailService.AssertNotCalled(t, "Send")
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Create Notification Fails", func(t *testing.T) {
		mockUserRepo.EXPECT().GetUserByEmail(ctx, testEmail).Return(user, nil).Once()
		mockRepo.EXPECT().CreateNotification(ctx, mock.AnythingOfType("*models.Notification")).Return(dbErr).Once()

		resp, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)
		mockEmailService.AssertNotCalled(t, "Send")
		mockRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Email Send Fails", func(t *testing.T) {
		mockUserRepo.EXPECT().GetUserByEmail(ctx, testEmail).Return(user, nil).Once()
		mockRepo.EXPECT().CreateNotification(ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Once()
		mockEmailService.EXPECT().Send(ctx, req).Return(sendErr).Once()
		mockRepo.EXPECT().UpdateNotificationStatus(ctx, mock.AnythingOfType("uuid.UUID"), models.StatusFailed, sendErr.Error()).Return(nil).Once()

		resp, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeThirdPartyError, appErr.Code)
		assert.ErrorIs(t, err, sendErr)
		mockRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("Failure - Update Status Fails After Send Success", func(t *testing.T) {
		mockUserRepo.EXPECT().GetUserByEmail(ctx, testEmail).Return(user, nil).Once()
		mockRepo.EXPECT().CreateNotification(ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Once()
		mockEmailService.EXPECT().Send(ctx, req).Return(nil).Once()
		mockRepo.EXPECT().UpdateNotificationStatus(ctx, mock.AnythingOfType("uuid.UUID"), models.StatusSent, "").Return(dbErr).Once()

		resp, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)
		mockRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})
}

func TestGetNotification(t *testing.T) {
	ctx := t.Context()
	mockRepo := repoMocks.NewMockNotificationRepository(t)
	mockUserRepo := repoMocks.NewMockUserRepository(t)
	mockEmailService := emailMocks.NewMockEmailService(t)
	svc := service.NewNotificationService(mockRepo, mockUserRepo, mockEmailService)

	testID := uuid.New()
	expectedNotification := &models.Notification{
		ID:        testID,
		Type:      models.NotificationTypeEmail,
		Recipient: "found@example.com",
		Status:    models.StatusSent,
		CreatedAt: time.Now(),
	}
	dbErr := errors.New("database error")
	notFoundErr := errors.New("not found")

	t.Run("Success", func(t *testing.T) {
		mockRepo.EXPECT().GetNotificationByID(ctx, testID).Return(expectedNotification, nil).Once()
		notification, err := svc.GetNotification(ctx, testID)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotification, notification)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		mockRepo.EXPECT().GetNotificationByID(ctx, testID).Return(nil, notFoundErr).Once()
		notification, err := svc.GetNotification(ctx, testID)

		assert.Error(t, err)
		assert.Nil(t, notification)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		assert.ErrorIs(t, err, notFoundErr)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Other DB Error", func(t *testing.T) {
		mockRepo.EXPECT().GetNotificationByID(ctx, testID).Return(nil, dbErr).Once()
		notification, err := svc.GetNotification(ctx, testID)

		assert.Error(t, err)
		assert.Nil(t, notification)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		assert.ErrorIs(t, err, dbErr)
		mockRepo.AssertExpectations(t)
	})
}

func TestListNotifications(t *testing.T) {
	ctx := t.Context()
	mockRepo := repoMocks.NewMockNotificationRepository(t)
	mockUserRepo := repoMocks.NewMockUserRepository(t)
	mockEmailService := emailMocks.NewMockEmailService(t)
	svc := service.NewNotificationService(mockRepo, mockUserRepo, mockEmailService)

	expectedNotifications := []*models.Notification{
		{ID: uuid.New(), Recipient: "user1@example.com"},
		{ID: uuid.New(), Recipient: "user2@example.com"},
	}
	expectedTotal := 15
	dbErr := errors.New("database error")

	t.Run("Success - Specific Page and Size", func(t *testing.T) {
		page, size := 2, 5
		mockRepo.EXPECT().ListNotifications(ctx, page, size).Return(expectedNotifications, expectedTotal, nil).Once()

		notifications, total, err := svc.ListNotifications(ctx, page, size)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotifications, notifications)
		assert.Equal(t, expectedTotal, total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Default Page and Size (Page < 1)", func(t *testing.T) {
		page, size := 0, 5
		expectedPage := 1
		mockRepo.EXPECT().ListNotifications(ctx, expectedPage, size).Return(expectedNotifications, expectedTotal, nil).Once()

		notifications, total, err := svc.ListNotifications(ctx, page, size)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotifications, notifications)
		assert.Equal(t, expectedTotal, total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Default Page and Size (Size < 1)", func(t *testing.T) {
		page, size := 1, 0
		expectedSize := 10
		mockRepo.EXPECT().ListNotifications(ctx, page, expectedSize).Return(expectedNotifications, expectedTotal, nil).Once()

		notifications, total, err := svc.ListNotifications(ctx, page, size)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotifications, notifications)
		assert.Equal(t, expectedTotal, total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Default Page and Size (Size > 10)", func(t *testing.T) {
		page, size := 1, 20
		expectedSize := 10
		mockRepo.EXPECT().ListNotifications(ctx, page, expectedSize).Return(expectedNotifications, expectedTotal, nil).Once()

		notifications, total, err := svc.ListNotifications(ctx, page, size)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotifications, notifications)
		assert.Equal(t, expectedTotal, total)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Repository Error", func(t *testing.T) {
		page, size := 1, 10
		mockRepo.EXPECT().ListNotifications(ctx, page, size).Return(nil, 0, dbErr).Once()

		notifications, total, err := svc.ListNotifications(ctx, page, size)

		assert.Error(t, err)
		assert.Nil(t, notifications)
		assert.Equal(t, 0, total)

		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)
		mockRepo.AssertExpectations(t)
	})
}
