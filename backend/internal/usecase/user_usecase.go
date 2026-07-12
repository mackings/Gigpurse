package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"gigpurse/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type userUsecase struct {
	userRepo        domain.UserRepository
	resetTokenRepo  domain.PasswordResetRepository
	emailVerifyRepo domain.EmailVerificationRepository
}

func NewUserUsecase(repo domain.UserRepository, resetRepos ...domain.PasswordResetRepository) domain.UserUsecase {
	var resetRepo domain.PasswordResetRepository
	if len(resetRepos) > 0 {
		resetRepo = resetRepos[0]
	}
	var emailVerifyRepo domain.EmailVerificationRepository
	if len(resetRepos) > 1 {
		if repo, ok := any(resetRepos[1]).(domain.EmailVerificationRepository); ok {
			emailVerifyRepo = repo
		}
	}
	return &userUsecase{
		userRepo:        repo,
		resetTokenRepo:  resetRepo,
		emailVerifyRepo: emailVerifyRepo,
	}
}

func NewUserUsecaseWithVerification(repo domain.UserRepository, resetRepo domain.PasswordResetRepository, emailVerifyRepo domain.EmailVerificationRepository) domain.UserUsecase {
	return &userUsecase{
		userRepo:        repo,
		resetTokenRepo:  resetRepo,
		emailVerifyRepo: emailVerifyRepo,
	}
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "gigpurse-fallback-secret-key-12345"
	}
	return []byte(secret)
}

func (u *userUsecase) SignUp(ctx context.Context, email, password, role, name string, acceptedTerms bool) (*domain.User, error) {
	if email == "" || password == "" || role == "" || name == "" {
		return nil, errors.New("missing required signup fields")
	}
	if !acceptedTerms {
		return nil, errors.New("you must accept the Terms and Conditions to sign up")
	}

	if role != "client" && role != "musician" && role != "admin" && role != "moderator" {
		return nil, errors.New("role must be 'client', 'musician', 'moderator', or 'admin'")
	}
	if role == "admin" && os.Getenv("ALLOW_ADMIN_SIGNUP") != "true" {
		return nil, errors.New("admin signup is disabled")
	}
	if role == "moderator" && os.Getenv("ALLOW_MODERATOR_SIGNUP") != "true" {
		return nil, errors.New("moderator signup is disabled")
	}

	// Check if email already exists
	existing, err := u.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate a unique ID (normally handled by DB, but we will generate one here or in repo)
	// For MongoDB, we can let MongoDB generate it, but we can generate a temporary ID if we want,
	// or leave it empty so MongoDB can populate it. However, since the ID field is a string,
	// we can generate a unique string or handle it in repo. Let's do it in repo.
	newUser := &domain.User{
		Email:           email,
		EmailVerified:   false,
		PasswordHash:    string(hashed),
		Role:            role,
		Name:            name,
		TermsAcceptedAt: time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if role == "musician" {
		newUser.MusicianProfile = &domain.MusicianProfile{
			Portfolio: []domain.PortfolioItem{},
		}
	} else if role == "client" {
		newUser.ClientProfile = &domain.ClientProfile{}
	}

	if err := u.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if u.emailVerifyRepo != nil {
		_ = u.sendEmailVerification(ctx, newUser)
	}

	return newUser, nil
}

func (u *userUsecase) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	if email == "" || password == "" {
		return "", nil, errors.New("email and password are required")
	}

	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}
	if !user.EmailVerified {
		return "", nil, errors.New("email is not verified")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(getJWTSecret())
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, user, nil
}

func (u *userUsecase) ResendEmailVerification(ctx context.Context, email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if u.emailVerifyRepo == nil {
		return errors.New("email verification is not configured")
	}
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}
	if user.EmailVerified {
		return nil
	}
	return u.sendEmailVerification(ctx, user)
}

func (u *userUsecase) VerifyEmail(ctx context.Context, email, code string) error {
	if email == "" || code == "" {
		return errors.New("email and code are required")
	}
	if u.emailVerifyRepo == nil {
		return errors.New("email verification is not configured")
	}
	verifyToken, err := u.emailVerifyRepo.GetByTokenHash(ctx, hashToken(emailVerificationHashInput(email, code)))
	if err != nil {
		return errors.New("invalid or expired email verification code")
	}
	if !verifyToken.UsedAt.IsZero() || time.Now().After(verifyToken.ExpiresAt) {
		return errors.New("invalid or expired email verification code")
	}
	user, err := u.userRepo.GetByID(ctx, verifyToken.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if !strings.EqualFold(user.Email, email) {
		return errors.New("invalid or expired email verification code")
	}
	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	if err := u.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}
	return u.emailVerifyRepo.MarkUsed(ctx, verifyToken.ID, time.Now())
}

func (u *userUsecase) sendEmailVerification(ctx context.Context, user *domain.User) error {
	code, err := secureDigits(6)
	if err != nil {
		return fmt.Errorf("failed to generate email verification code: %w", err)
	}
	now := time.Now()
	verifyToken := &domain.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: hashToken(emailVerificationHashInput(user.Email, code)),
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
	}
	if err := u.emailVerifyRepo.Create(ctx, verifyToken); err != nil {
		return err
	}
	subject := "Verify your Gigpurse email"
	body := fmt.Sprintf("Your Gigpurse verification code is %s. It expires in 15 minutes.", code)
	if err := sendEmail(user.Email, subject, body); err != nil {
		log.Printf("[EMAIL OUTBOX FAILED] To %s: Subject: %s | Code: %s | Error: %v", user.Email, subject, code, err)
		return err
	}
	if emailProviderConfigured() {
		log.Printf("[EMAIL SENT] To %s: Subject: %s", user.Email, subject)
	} else {
		log.Printf("[EMAIL OUTBOX] To %s: Subject: %s | Code: %s", user.Email, subject, code)
	}
	return nil
}

func (u *userUsecase) RequestPasswordReset(ctx context.Context, email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if u.resetTokenRepo == nil {
		return errors.New("password reset is not configured")
	}

	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Keep the endpoint non-enumerating while still succeeding for callers.
		return nil
	}

	token, err := secureToken()
	if err != nil {
		return fmt.Errorf("failed to generate password reset token: %w", err)
	}
	now := time.Now()
	resetToken := &domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		ExpiresAt: now.Add(30 * time.Minute),
		CreatedAt: now,
	}
	if err := u.resetTokenRepo.Create(ctx, resetToken); err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	subject := "Reset your Gigpurse password"
	body := fmt.Sprintf("Use this password reset token: %s. It expires in 30 minutes.", token)
	if err := sendEmail(email, subject, body); err != nil {
		log.Printf("[EMAIL OUTBOX FAILED] To %s: Subject: %s | Token: %s | Error: %v", email, subject, token, err)
		return err
	}
	if emailProviderConfigured() {
		log.Printf("[EMAIL SENT] To %s: Subject: %s", email, subject)
	} else {
		log.Printf("[EMAIL OUTBOX] To %s: Subject: %s | Token: %s", email, subject, token)
	}
	return nil
}

func (u *userUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token == "" || newPassword == "" {
		return errors.New("token and new password are required")
	}
	if u.resetTokenRepo == nil {
		return errors.New("password reset is not configured")
	}

	resetToken, err := u.resetTokenRepo.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		return errors.New("invalid or expired password reset token")
	}
	if !resetToken.UsedAt.IsZero() || time.Now().After(resetToken.ExpiresAt) {
		return errors.New("invalid or expired password reset token")
	}

	user, err := u.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = string(hashed)
	user.UpdatedAt = time.Now()
	if err := u.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return u.resetTokenRepo.MarkUsed(ctx, resetToken.ID, time.Now())
}

func (u *userUsecase) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

func (u *userUsecase) UpdateProfile(ctx context.Context, id string, name, bio, location string, musProfile *domain.MusicianProfile, cliProfile *domain.ClientProfile) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if name != "" {
		user.Name = name
	}
	user.Bio = bio
	user.Location = location
	user.UpdatedAt = time.Now()

	if user.Role == "musician" && musProfile != nil {
		if user.MusicianProfile == nil {
			user.MusicianProfile = &domain.MusicianProfile{}
		}
		user.MusicianProfile.StageName = musProfile.StageName
		user.MusicianProfile.Instruments = musProfile.Instruments
		user.MusicianProfile.Genres = musProfile.Genres
		user.MusicianProfile.ExperienceYears = musProfile.ExperienceYears
		user.MusicianProfile.PriceMin = musProfile.PriceMin
		user.MusicianProfile.PriceMax = musProfile.PriceMax
		user.MusicianProfile.Availability = musProfile.Availability
		user.MusicianProfile.SocialLinks = musProfile.SocialLinks
		user.MusicianProfile.IntroVideoURL = musProfile.IntroVideoURL
		user.MusicianProfile.Portfolio = musProfile.Portfolio
	} else if user.Role == "client" && cliProfile != nil {
		if user.ClientProfile == nil {
			user.ClientProfile = &domain.ClientProfile{}
		}
		user.ClientProfile.CompanyName = cliProfile.CompanyName
	}

	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (u *userUsecase) BrowseMusicians(ctx context.Context, filter domain.MusicianFilter) ([]*domain.User, error) {
	return u.userRepo.ListMusicians(ctx, filter)
}

func secureToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func secureDigits(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, n), nil
}

func emailVerificationHashInput(email, code string) string {
	return strings.ToLower(strings.TrimSpace(email)) + ":" + strings.TrimSpace(code)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func sendEmail(to, subject, body string) error {
	if resendConfigured() {
		return sendResendEmail(to, subject, body)
	}
	if mailjetConfigured() {
		return sendMailjetEmail(to, subject, body)
	}
	if !smtpConfigured() {
		log.Printf("[EMAIL OUTBOX - SMTP NOT CONFIGURED] To %s: Subject: %s | Body: %s", to, subject, body)
		return nil
	}
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=\"UTF-8\"",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", username, password, host)
	return smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(msg))
}

func sendMailjetEmail(to, subject, body string) error {
	apiKey := os.Getenv("MAILJET_API_KEY")
	apiSecret := os.Getenv("MAILJET_API_SECRET")
	fromEmail := os.Getenv("MAILJET_FROM_EMAIL")
	fromName := os.Getenv("MAILJET_FROM_NAME")
	if fromName == "" {
		fromName = "Gigpurse"
	}

	payload := map[string]interface{}{
		"Messages": []map[string]interface{}{
			{
				"From": map[string]string{
					"Email": fromEmail,
					"Name":  fromName,
				},
				"To": []map[string]string{
					{"Email": to},
				},
				"Subject":  subject,
				"TextPart": body,
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.mailjet.com/v3.1/send", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, apiSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("mailjet send failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func sendResendEmail(to, subject, body string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	fromEmail := os.Getenv("RESEND_FROM_EMAIL")

	payload := map[string]interface{}{
		"from":    fromEmail,
		"to":      []string{to},
		"subject": subject,
		"text":    body,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("resend send failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func smtpConfigured() bool {
	return os.Getenv("SMTP_HOST") != "" &&
		os.Getenv("SMTP_PORT") != "" &&
		os.Getenv("SMTP_USERNAME") != "" &&
		os.Getenv("SMTP_PASSWORD") != "" &&
		os.Getenv("SMTP_FROM") != ""
}

func mailjetConfigured() bool {
	return os.Getenv("MAILJET_API_KEY") != "" &&
		os.Getenv("MAILJET_API_SECRET") != "" &&
		os.Getenv("MAILJET_FROM_EMAIL") != ""
}

func resendConfigured() bool {
	return os.Getenv("RESEND_API_KEY") != "" && os.Getenv("RESEND_FROM_EMAIL") != ""
}

func emailProviderConfigured() bool {
	return resendConfigured() || mailjetConfigured() || smtpConfigured()
}
