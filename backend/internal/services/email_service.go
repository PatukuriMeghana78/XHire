package services

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"gorm.io/gorm"
)

type EmailService struct {
	db             *gorm.DB
	appRepo        *repositories.ApplicationRepository
	emailRepo      *repositories.EmailEventRepository
	userRepo       *repositories.UserRepository
	notificationRepo *repositories.NotificationRepository
	nlpURL         string
	oauthConfig    *oauth2.Config
}

func NewEmailService(
	db *gorm.DB,
	appRepo *repositories.ApplicationRepository,
	emailRepo *repositories.EmailEventRepository,
	userRepo *repositories.UserRepository,
	notificationRepo *repositories.NotificationRepository,
	nlpURL string,
) *EmailService {

	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.modify",
		},
		Endpoint: google.Endpoint,
	}

	log.Println("📧 Gmail OAuth Config Loaded:")
	log.Println("   Client ID:", oauthConfig.ClientID)
	log.Println("   Redirect URL:", oauthConfig.RedirectURL)

	return &EmailService{
		db:               db,
		appRepo:          appRepo,
		emailRepo:        emailRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		nlpURL:           nlpURL,
		oauthConfig:      oauthConfig,
	}
}

// DB getter
func (s *EmailService) DB() *gorm.DB {
	return s.db
}

// OAuth
func (s *EmailService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (s *EmailService) ExchangeCode(code string) (*oauth2.Token, error) {
	ctx := context.Background()
	return s.oauthConfig.Exchange(ctx, code)
}

// Gmail Polling
func (s *EmailService) PollUnreadEmails() {

	var tokens []models.GmailToken
	s.db.Find(&tokens)

	for _, token := range tokens {

		ctx := context.Background()

		client := s.oauthConfig.Client(ctx, &oauth2.Token{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		})

		gmailService, err := gmail.New(client)
		if err != nil {
			log.Println("❌ Gmail client error:", err)
			continue
		}

		resp, err := gmailService.Users.Messages.List("me").
			Q("is:unread newer_than:2d").
			MaxResults(5).
			Do()

		if err != nil {
			log.Println("❌ Gmail fetch error:", err)
			continue
		}

		log.Println("📩 Unread messages found:", len(resp.Messages))

		for _, msg := range resp.Messages {

			fullMsg, err := gmailService.Users.Messages.Get("me", msg.Id).
				Format("full").
				Do()

			if err != nil {
				log.Println("❌ Message fetch error:", err)
				continue
			}

			body := extractEmailBody(fullMsg)
			subject := extractEmailSubject(fullMsg)

			log.Println("📨 Processing email ID:", msg.Id)
			log.Println("📧 Subject:", subject)
			log.Println("📄 Email preview:", firstN(body, 120))

			// Create email event (simplified - you can add NLP classification here)
			emailEvent := &models.EmailEvent{
				UserID:          token.UserID,
				EmailID:         msg.Id,
				DetectedCompany: extractCompanyFromSubject(subject),
				DetectedRole:    "Position",
				SuggestedStatus: "Applied",
				Confidence:      75,
				ProcessedAt:     time.Now(),
				Confirmed:       false,
			}

			// Save email event
			if err := s.db.Create(emailEvent).Error; err != nil {
				log.Println("❌ Failed to save email event:", err)
				continue
			}

			// Create notification for user
			notification := &models.Notification{
				UserID:    token.UserID,
				Type:      "email",
				Title:     "New Job Email Detected",
				Message:   "Email from " + emailEvent.DetectedCompany + " about " + emailEvent.DetectedRole + " position",
				Read:      false,
				ActionURL: "/emails",
			}

			if err := s.notificationRepo.Create(notification); err != nil {
				log.Println("❌ Failed to create notification:", err)
			} else {
				log.Println("✅ Notification created for user", token.UserID, "- Email from", emailEvent.DetectedCompany)
			}

			// Mark email as read
			_, err = gmailService.Users.Messages.Modify("me", msg.Id,
				&gmail.ModifyMessageRequest{
					RemoveLabelIds: []string{"UNREAD"},
				}).Do()

			if err != nil {
				log.Println("❌ Failed to mark as read:", err)
			}
		}
	}
}

// Manual sync
func (s *EmailService) SyncUserEmails(userID uint) (int, error) {
	s.PollUnreadEmails()
	return 1, nil
}

// Extract email body
func extractEmailBody(msg *gmail.Message) string {

	if msg.Payload == nil {
		return ""
	}

	if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		return string(data)
	}

	for _, part := range msg.Payload.Parts {
		if part.Body != nil && part.Body.Data != "" {
			data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
			return string(data)
		}
	}

	return ""
}

// Extract email subject
func extractEmailSubject(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			return header.Value
		}
	}
	return "No Subject"
}

// Extract company name from subject (simple heuristic)
func extractCompanyFromSubject(subject string) string {
	// Common patterns in job emails
	// "Interview with Google", "Your application at Microsoft", "Amazon - Next Steps"
	
	// Try to extract company name from common patterns
	subject = strings.ToLower(subject)
	
	// Pattern: "from [Company]"
	if strings.Contains(subject, "from ") {
		parts := strings.Split(subject, "from ")
		if len(parts) > 1 {
			company := strings.TrimSpace(parts[1])
			if len(company) > 0 && len(company) < 50 {
				return strings.Title(strings.Split(company, " ")[0])
			}
		}
	}
	
	// Pattern: "at [Company]"
	if strings.Contains(subject, " at ") {
		parts := strings.Split(subject, " at ")
		if len(parts) > 1 {
			company := strings.TrimSpace(parts[1])
			if len(company) > 0 && len(company) < 50 {
				return strings.Title(strings.Split(company, " ")[0])
			}
		}
	}
	
	// Pattern: "[Company] -" or "[Company]:"
	if strings.Contains(subject, " - ") {
		parts := strings.Split(subject, " - ")
		if len(parts) > 0 {
			company := strings.TrimSpace(parts[0])
			if len(company) > 0 && len(company) < 50 {
				return strings.Title(company)
			}
		}
	}
	
	// Default: return first 30 chars of subject
	if len(subject) > 30 {
		return strings.Title(subject[:30]) + "..."
	}
	return strings.Title(subject)
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Existing suggestion logic
func (s *EmailService) GetPendingSuggestions(userID uint) ([]models.EmailEvent, error) {
	return s.emailRepo.FindPendingByUserID(userID)
}

func (s *EmailService) ConfirmSuggestion(
	userID,
	emailEventID uint,
	applicationID *uint,
	status string,
) (*models.Application, error) {

	var resultApp *models.Application

	err := s.db.Transaction(func(tx *gorm.DB) error {

		event, err := s.emailRepo.FindByID(emailEventID)
		if err != nil {
			return err
		}

		if event.UserID != userID {
			return errors.New("unauthorized")
		}

		var app *models.Application

		if applicationID != nil {

			app, err = s.appRepo.FindByID(*applicationID)
			if err != nil {
				return err
			}

			app.Status = status
			app.LastUpdated = time.Now()

			if err := tx.Save(app).Error; err != nil {
				return err
			}

		} else {

			app = &models.Application{
				UserID:      userID,
				Company:     event.DetectedCompany,
				Role:        event.DetectedRole,
				AppliedDate: time.Now(),
				Status:      status,
				LastUpdated: time.Now(),
				Source:      "Email",
			}

			if err := tx.Create(app).Error; err != nil {
				return err
			}
		}

		event.Confirmed = true
		event.ApplicationID = &app.ID

		if err := tx.Save(event).Error; err != nil {
			return err
		}

		// Create notification for confirmed email
		notification := &models.Notification{
			UserID:    userID,
			Type:      "application",
			Title:     "Email Confirmed",
			Message:   "Application for " + app.Role + " at " + app.Company + " created from email",
			Read:      false,
			ActionURL: "/applications",
		}
		if err := s.notificationRepo.Create(notification); err != nil {
			log.Println("⚠️  Failed to create notification:", err)
		}

		resultApp = app
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resultApp, nil
}

func (s *EmailService) IgnoreSuggestion(userID, emailEventID uint) error {

	event, err := s.emailRepo.FindByID(emailEventID)
	if err != nil {
		return err
	}

	if event.UserID != userID {
		return errors.New("unauthorized")
	}

	event.Confirmed = true
	return s.emailRepo.Update(event)
}

// CreateConnectionNotification creates a notification when Gmail is connected
func (s *EmailService) CreateConnectionNotification(notification *models.Notification) error {
	return s.notificationRepo.Create(notification)
}
