package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/notifications/email"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/repositories"
)

type NotificationService struct {
	profileRepo repositories.ProfileRepository
	emailClient *email.Client
	logger      logging.Logger
}

func NewNotificationService(
	profileRepo repositories.ProfileRepository,
	emailClient *email.Client,
	logger logging.Logger,
) *NotificationService {
	return &NotificationService{
		profileRepo: profileRepo,
		emailClient: emailClient,
		logger:      logger,
	}
}

// SendLikeNotificationEmail sends email notification when someone likes a profile
func (s *NotificationService) SendLikeNotificationEmail(ctx context.Context, recipientUserID, senderProfileID, senderName string) error {
	// Parse recipient UUID
	recipientUUID, err := uuid.Parse(recipientUserID)
	if err != nil {
		return fmt.Errorf("invalid recipient user ID: %w", err)
	}

	// Get recipient profile to get their email
	recipientProfile, err := s.profileRepo.GetProfileByUserID(ctx, recipientUUID)
	if err != nil {
		s.logger.Error("Failed to get recipient profile", "recipientUserID", recipientUserID, "error", err)
		return fmt.Errorf("failed to get recipient profile: %w", err)
	}

	if recipientProfile == nil {
		s.logger.Warn("Recipient profile not found", "recipientUserID", recipientUserID)
		return fmt.Errorf("recipient profile not found")
	}

	if recipientProfile.Email == "" {
		s.logger.Warn("Recipient has no email address", "recipientUserID", recipientUserID)
		return fmt.Errorf("recipient has no email address")
	}

	// Send email notification
	if err := s.sendLikeEmail(recipientProfile.Email, senderProfileID, senderName); err != nil {
		s.logger.Error("Failed to send like email", "email", recipientProfile.Email, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Like notification email sent successfully",
		"recipientUserID", recipientUserID,
		"senderProfileID", senderProfileID)
	return nil
}

// sendLikeEmail sends the actual email
func (s *NotificationService) sendLikeEmail(recipientEmail, senderProfileID, senderName string) error {
	subject := "You have received an interest!"

	// Create HTML email body (UUID not exposed, only profile ID and name)
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
				<h2 style="color: #2c5aa0;">🎉 You have received an interest!</h2>
				
				<p>Hello,</p>
				
				<p>Great news! Someone is interested in your profile on Qubool Kallyanam.</p>
				
				<div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
					<p style="margin: 0;"><strong>Profile ID:</strong> %s</p>
					<p style="margin: 10px 0 0 0;"><strong>Name:</strong> %s</p>
				</div>
				
				<p>Log in to your account to view their complete profile and connect with them!</p>
				
				<div style="text-align: center; margin: 30px 0;">
					<a href="#" style="background-color: #2c5aa0; color: white; padding: 12px 25px; text-decoration: none; border-radius: 5px; display: inline-block;">View Profile</a>
				</div>
				
				<hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
				
				<p style="font-size: 12px; color: #666;">
					This email was sent to you because someone expressed interest in your profile on Qubool Kallyanam.
					<br>If you don't want to receive these notifications, you can update your preferences in your account settings.
				</p>
			</div>
		</body>
		</html>
	`, senderProfileID, senderName)

	emailData := email.EmailData{
		To:      recipientEmail,
		Subject: subject,
		Body:    body,
		IsHTML:  true,
	}

	return s.emailClient.SendEmail(emailData)
}
