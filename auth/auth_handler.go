package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthService struct {
	// You can add fields here if needed, e.g., a user store or configuration
	client    *dynamodb.Client
	tableName string
}

type UserPasswordDB struct {
	UserName       string `dynamodbav:"UserName"`
	HashedPassword string `dynamodbav:"HashedPassword"`
}

var jwtSecret = []byte("your-super-secret-jwt-key-change-in-production")

func NewAuthService(client *dynamodb.Client, tableName string) *AuthService {
	return &AuthService{client: client, tableName: tableName}
}

// RegisterLogin registers a new user.
// @Summary Register a new user
// @Description Creates a new user with a hashed password stored in DynamoDB.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User credentials"
// @Success 201 {object} map[string]string "message: User registered successfully"
// @Failure 400 {string} string "Invalid request body / Username and password are required"
// @Failure 405 {string} string "Method not allowed"
// @Failure 409 {string} string "Could not register user (username may already exist)"
// @Failure 500 {string} string "Internal server error"
// @Router /register [post]
func (s *AuthService) RegisterLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	//1. Check if the request method is POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the incoming JSON body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 3. Basic validation (ensure fields are not empty)
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	// 4. Hash the password using bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error generating hashed password", http.StatusInternalServerError)
		return
	}

	user := UserPasswordDB{
		UserName:       req.Username,
		HashedPassword: string(hashedBytes),
	}

	// 5. Marshal user struct into DynamoDB AttributeValues
	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		http.Error(w, "Error marshalling user data", http.StatusInternalServerError)
		return
	}

	// 6. Put item into DynamoDB table (with a condition to prevent overwriting existing users)
	_, err = s.client.PutItem(r.Context(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(UserName)"),
	})

	// 7. Call the AuthService Register method (hashes password and saves to DynamoDB)
	if err != nil {
		// Optional: Check if the error is due to the user already existing
		// (ConditionalCheckFailedException from DynamoDB)
		http.Error(w, "Could not register user (username may already exist)", http.StatusConflict)
		return
	}

	// 8. Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
	})
}

// handleLogin authenticates a user and sets an auth_token cookie.
// @Summary Log in a user
// @Description Verifies credentials against DynamoDB, generates a JWT, and returns it via an HTTP-only cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User credentials"
// @Success 200 {object} map[string]string "message: Login successful"
// @Header 200 {string} Set-Cookie "auth_token=JWT_TOKEN; Path=/; HttpOnly; SameSite=Strict"
// @Failure 400 {string} string "Invalid request payload"
// @Failure 401 {string} string "Invalid username or password"
// @Failure 500 {string} string "Internal server error"
// @Router /login [post]
func (s *AuthService) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	key, err := attributevalue.MarshalMap(map[string]string{
		"UserName": req.Username,
	})
	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key:       key,
	}

	output, err := s.client.GetItem(r.Context(), input)
	if err != nil {
		http.Error(w, "Error consulting DynamoDB", http.StatusInternalServerError)
		return
	}
	if output.Item == nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	var user UserPasswordDB
	err = attributevalue.UnmarshalMap(output.Item, &user)
	// Validate username
	if err != nil {
		http.Error(w, "Error deserializing user data", http.StatusInternalServerError)
		return
	}

	// Validate password against bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create JWT Claims (expires in 24 hours)
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "my-auth-service",
		},
	}

	// Sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set the JWT into a secure HTTP-only Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		Expires:  expirationTime,
		HttpOnly: true,                    // Prevents JavaScript access (XSS defense)
		Secure:   false,                   // Set to true in production (requires HTTPS)
		SameSite: http.SameSiteStrictMode, // CSRF defense
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
}

// handleLogout clears the auth_token cookie to log out the user.
// @Summary Log out a user
// @Description Clears the HTTP-only auth_token cookie.
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string "message: Logged out successfully"
// @Header 200 {string} Set-Cookie "auth_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; HttpOnly; SameSite=Strict"
// @Router /logout [post]
func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the cookie by setting an expired time and empty value
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Expire immediately
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}
