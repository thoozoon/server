package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID               int       `json:"id"`
	Email            string    `json:"email"`
	Password         string    `json:"-"` // Never include in JSON
	IsAdmin          bool      `json:"is_admin"`
	SetupToken       string    `json:"-"` // Never include in JSON
	SetupTokenExpiry time.Time `json:"-"` // Never include in JSON
	IsSetup          bool      `json:"is_setup"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AuthClaims struct {
	UserID  int    `json:"user_id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

type AuthManager struct {
	db        *sql.DB
	jwtSecret []byte
}

func NewAuthManager(dbPath string) (*AuthManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	am := &AuthManager{
		db:        db,
		jwtSecret: []byte("your-secret-key-change-this-in-production"), // TODO: Use env var
	}

	if err := am.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Create default admin user if no users exist
	if err := am.createDefaultAdmin(); err != nil {
		log.Printf("Warning: Failed to create default admin: %v", err)
	}

	return am, nil
}

func (am *AuthManager) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password TEXT,
		is_admin BOOLEAN DEFAULT FALSE,
		setup_token TEXT,
		setup_token_expiry DATETIME,
		is_setup BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TRIGGER IF NOT EXISTS update_users_updated_at
		AFTER UPDATE ON users
		FOR EACH ROW
		BEGIN
			UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`

	_, err := am.db.Exec(query)
	return err
}

func (am *AuthManager) createDefaultAdmin() error {
	var count int
	err := am.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Create default admin with the old password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("ahsahbeequen"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = am.db.Exec(`
			INSERT INTO users (email, password, is_admin, is_setup)
			VALUES (?, ?, ?, ?)
		`, "admin@comp3007.local", string(hashedPassword), true, true)
		return err
	}
	return nil
}

func (am *AuthManager) CreateUser(email string, isAdmin bool) (*User, error) {
	// Generate setup token
	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate setup token: %w", err)
	}

	expiry := time.Now().Add(7 * 24 * time.Hour) // 7 days

	result, err := am.db.Exec(`
		INSERT INTO users (email, is_admin, setup_token, setup_token_expiry)
		VALUES (?, ?, ?, ?)
	`, email, isAdmin, token, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	user := &User{
		ID:               int(id),
		Email:            email,
		IsAdmin:          isAdmin,
		SetupToken:       token,
		SetupTokenExpiry: expiry,
		IsSetup:          false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return user, nil
}

func (am *AuthManager) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	var password sql.NullString
	var setupToken sql.NullString
	var setupTokenExpiry sql.NullTime

	err := am.db.QueryRow(`
		SELECT id, email, password, is_admin, setup_token, setup_token_expiry, is_setup, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &password, &user.IsAdmin, &setupToken, &setupTokenExpiry, &user.IsSetup, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	if password.Valid {
		user.Password = password.String
	}
	if setupToken.Valid {
		user.SetupToken = setupToken.String
	}
	if setupTokenExpiry.Valid {
		user.SetupTokenExpiry = setupTokenExpiry.Time
	}

	return user, nil
}

func (am *AuthManager) GetUserByID(id int) (*User, error) {
	user := &User{}
	var password sql.NullString
	var setupToken sql.NullString
	var setupTokenExpiry sql.NullTime

	err := am.db.QueryRow(`
		SELECT id, email, password, is_admin, setup_token, setup_token_expiry, is_setup, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Email, &password, &user.IsAdmin, &setupToken, &setupTokenExpiry, &user.IsSetup, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	if password.Valid {
		user.Password = password.String
	}
	if setupToken.Valid {
		user.SetupToken = setupToken.String
	}
	if setupTokenExpiry.Valid {
		user.SetupTokenExpiry = setupTokenExpiry.Time
	}

	return user, nil
}

func (am *AuthManager) GetUserBySetupToken(token string) (*User, error) {
	user := &User{}
	var password sql.NullString
	var setupToken sql.NullString
	var setupTokenExpiry sql.NullTime

	err := am.db.QueryRow(`
		SELECT id, email, password, is_admin, setup_token, setup_token_expiry, is_setup, created_at, updated_at
		FROM users WHERE setup_token = ? AND setup_token_expiry > CURRENT_TIMESTAMP
	`, token).Scan(&user.ID, &user.Email, &password, &user.IsAdmin, &setupToken, &setupTokenExpiry, &user.IsSetup, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	if password.Valid {
		user.Password = password.String
	}
	if setupToken.Valid {
		user.SetupToken = setupToken.String
	}
	if setupTokenExpiry.Valid {
		user.SetupTokenExpiry = setupTokenExpiry.Time
	}

	return user, nil
}

func (am *AuthManager) GetAllUsers() ([]*User, error) {
	rows, err := am.db.Query(`
		SELECT id, email, is_admin, is_setup, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.Email, &user.IsAdmin, &user.IsSetup, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (am *AuthManager) SetupUserPassword(token, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	result, err := am.db.Exec(`
		UPDATE users
		SET password = ?, is_setup = TRUE, setup_token = NULL, setup_token_expiry = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE setup_token = ? AND setup_token_expiry > CURRENT_TIMESTAMP
	`, string(hashedPassword), token)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invalid or expired setup token")
	}

	return nil
}

func (am *AuthManager) UpdateUserPassword(userID int, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = am.db.Exec(`
		UPDATE users
		SET password = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, string(hashedPassword), userID)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	return nil
}

func (am *AuthManager) RegenerateSetupToken(userID int) (string, error) {
	token, err := generateSecureToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate setup token: %w", err)
	}

	expiry := time.Now().Add(7 * 24 * time.Hour) // 7 days

	_, err = am.db.Exec(`
		UPDATE users
		SET setup_token = ?, setup_token_expiry = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, token, expiry, userID)
	if err != nil {
		return "", fmt.Errorf("failed to update setup token: %w", err)
	}

	return token, nil
}

func (am *AuthManager) ValidateCredentials(email, password string) (*User, error) {
	user, err := am.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsSetup {
		return nil, fmt.Errorf("user account not set up")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

func (am *AuthManager) GenerateJWT(user *User) (string, error) {
	claims := AuthClaims{
		UserID:  user.ID,
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.jwtSecret)
}

func (am *AuthManager) ValidateJWT(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (am *AuthManager) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.AuthDisabled {
			next(w, r)
			return
		}
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		claims, err := am.ValidateJWT(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user info to request context
		r = r.WithContext(WithUserContext(r.Context(), claims))
		next(w, r)
	}
}

func (am *AuthManager) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return am.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if config.AuthDisabled {
			next(w, r)
			return
		}
		claims := GetUserFromContext(r.Context())
		if claims == nil || !claims.IsAdmin {
			http.Error(w, "Access denied: admin required", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func (am *AuthManager) SendSetupEmail(user *User, baseURL string) error {
	setupURL := fmt.Sprintf("%s/setup?token=%s", baseURL, user.SetupToken)

	// For development, log the setup URL
	log.Printf("Setup email for %s: %s", user.Email, setupURL)

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>COMP 3007 Account Setup</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f4f4f4; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; margin-top: 20px; }
        .header { background: #2563eb; color: white; padding: 20px; border-radius: 8px 8px 0 0; text-align: center; margin: -20px -20px 20px -20px; }
        .button { display: inline-block; background: #2563eb; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; margin: 10px 0; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #eee; font-size: 12px; color: #666; }
        .url-box { background: #f8f9fa; padding: 10px; border-radius: 4px; font-family: monospace; font-size: 12px; word-break: break-all; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2 style="margin: 0;">Welcome to COMP 3007</h2>
            <p style="margin: 5px 0 0 0;">Programming Paradigms</p>
        </div>

        <p>Hello!</p>

        <p>Your account has been created for the COMP 3007 course website. To get started, you need to set up your password.</p>

        <p style="text-align: center;">
            <a href="%s" class="button">Set Up Your Account</a>
        </p>

        <p><strong>Important:</strong> This setup link will expire in 7 days. If you don't set up your account within this time, please contact your instructor.</p>

        <p>If the button above doesn't work, you can copy and paste this URL into your browser:</p>
        <div class="url-box">%s</div>

        <div class="footer">
            <p>This email was sent automatically by the COMP 3007 course management system. If you believe you received this email in error, please contact your instructor.</p>
            <p>COMP 3007 - Programming Paradigms</p>
        </div>
    </div>
</body>
</html>`, setupURL, setupURL)

	return am.sendEmailWithResend(user.Email, "COMP 3007 Account Setup", htmlBody)
}

// ResendEmailRequest represents the JSON payload for Resend API
type ResendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

// ResendEmailResponse represents the response from Resend API
type ResendEmailResponse struct {
	Id string `json:"id"`
}

func (am *AuthManager) sendEmailWithResend(to, subject, htmlBody string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	fromEmail := os.Getenv("FROM_EMAIL")

	// For development/testing, just log the email if API key is not configured
	if apiKey == "" {
		log.Printf("Resend API key not configured, logging email instead:")
		log.Printf("TO: %s", to)
		log.Printf("SUBJECT: %s", subject)
		log.Printf("BODY: %s", htmlBody)
		return nil
	}

	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev" // Resend's test domain
	}

	// Prepare the email payload
	emailRequest := ResendEmailRequest{
		From:    fromEmail,
		To:      []string{to},
		Subject: subject,
		Html:    htmlBody,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(emailRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		// Read the response body for more detailed error information
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		log.Printf("Resend API error response: %s", bodyString)
		return fmt.Errorf("Resend API returned status %d: %s - %s", resp.StatusCode, resp.Status, bodyString)
	}

	// Parse response
	var emailResponse ResendEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Email sent successfully to %s via Resend API (ID: %s)", to, emailResponse.Id)
	return nil
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Context helpers for user information
type contextKey string

const userContextKey contextKey = "user"

func WithUserContext(ctx context.Context, claims *AuthClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

func GetUserFromContext(ctx context.Context) *AuthClaims {
	if claims, ok := ctx.Value(userContextKey).(*AuthClaims); ok {
		return claims
	}
	return nil
}
