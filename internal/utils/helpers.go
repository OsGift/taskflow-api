package utils

import (
	"bytes" // For building email body
	"encoding/json"
	"fmt"
	"html/template" // For parsing HTML templates
	"math/rand"
	"net/http"
	"net/smtp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	// For models.Permission
)

// Global mailer configuration
var (
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	auth         smtp.Auth
	templates    *template.Template
)

// InitMailer initializes the email sender with SMTP credentials and loads templates
func InitMailer(host, port, username, password string) error {
	smtpHost = host
	smtpPort = port
	smtpUsername = username
	smtpPassword = password
	auth = smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)

	// Load all HTML templates from the 'templates' directory
	var err error
	templates, err = template.ParseGlob("templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse email templates: %w", err)
	}
	fmt.Println("Email templates loaded successfully.")
	return nil
}

// SendEmail sends an HTML email using the specified template and data
func SendEmail(templateName, subject, toEmail string, data interface{}) {
	if templates == nil {
		fmt.Println("Mailer not initialized. Skipping email sending.")
		return
	}

	var body bytes.Buffer
	templatePath := fmt.Sprintf("%s.html", templateName)
	t := templates.Lookup(templatePath)
	if t == nil {
		fmt.Printf("Error: Template %s not found.\n", templatePath)
		return
	}

	err := t.Execute(&body, data)
	if err != nil {
		fmt.Printf("Error executing template %s: %v\n", templateName, err)
		return
	}

	msg := []byte("To: " + toEmail + "\r\n" +
		"From: " + smtpUsername + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
		"\r\n" +
		body.String())

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	err = smtp.SendMail(addr, auth, smtpUsername, []string{toEmail}, msg)
	if err != nil {
		fmt.Printf("Error sending email to %s: %v\n", toEmail, err)
	} else {
		fmt.Printf("Email '%s' sent to %s successfully.\n", subject, toEmail)
	}
}

// HashPassword hashes a plain-text password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain-text password with a hashed password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a new JWT token for the user
func GenerateToken(userID primitive.ObjectID, email string, roleID primitive.ObjectID, secretKey []byte) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.Hex(),
		"email":   email, // Using email in claims
		"role_id": roleID.Hex(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		"iss":     "taskflow-api",
		"aud":     "taskflow-clients",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// GeneratePasswordResetToken generates a JWT for password reset
func GeneratePasswordResetToken(userID primitive.ObjectID, secretKey []byte) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.Hex(),
		"exp":     time.Now().Add(time.Hour).Unix(), // Reset token expires in 1 hour
		"iss":     "taskflow-api-reset",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidatePasswordResetToken validates a password reset token
func ValidatePasswordResetToken(tokenString string, secretKey []byte) (primitive.ObjectID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return primitive.NilObjectID, err
	}

	if !token.Valid {
		return primitive.NilObjectID, fmt.Errorf("invalid password reset token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("invalid password reset token claims")
	}

	userIDHex, ok := claims["user_id"].(string)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("user ID claim missing from reset token")
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid user ID format in reset token")
	}

	return userID, nil
}

// GenerateVerificationToken generates a JWT for email verification
func GenerateVerificationToken(userID string, secretKey []byte) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Verification token expires in 24 hours
		"iss":     "taskflow-api-verify",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RespondWithError sends a JSON error response
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]interface{}{"error": true, "message": message})
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Error marshalling JSON response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
