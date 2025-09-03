package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"

	"armstrong-webapi/cmd/service/auth"
)

// User represents the users table
type User struct {
	UserID    int       `json:"user_id" db:"user_id"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"password,omitempty" db:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	IsAdmin   bool      `json:"is_admin" db:"is_admin"`
}

// handleGetMyArmstrong returns all Armstrong numbers saved by the authenticated user.
func (h *Handler) handleGetMyArmstrong(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(
		"SELECT id, user_id, thennumber, created_at FROM armstrong_numbers WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		http.Error(w, "Error fetching Armstrong numbers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var numbers []ArmstrongNumber
	for rows.Next() {
		var arm ArmstrongNumber
		if err := rows.Scan(&arm.ID, &arm.UserID, &arm.ThenNumber, &arm.CreatedAt); err != nil {
			http.Error(w, "Error scanning Armstrong numbers", http.StatusInternalServerError)
			return
		}
		numbers = append(numbers, arm)
	}

	json.NewEncoder(w).Encode(numbers)
}

// handleGetAllUsers returns all users (admin only).
func (h *Handler) handleGetAllUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		"SELECT user_id, email, created_at, is_admin FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		http.Error(w, "Error fetching users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.UserID, &user.Email, &user.CreatedAt, &user.IsAdmin); err != nil {
			http.Error(w, "Error scanning users", http.StatusInternalServerError)
			return
		}
		user.Password = ""
		users = append(users, user)
	}

	json.NewEncoder(w).Encode(users)
}

// ArmstrongNumber represents the armstrong_numbers table
type ArmstrongNumber struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	ThenNumber int       `json:"thennumber" db:"thennumber"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type Handler struct {
	db       *sql.DB
	validate *validator.Validate
}

// Database operations
func (h *Handler) CreateUser(email string) (*User, error) {
	var user User
	err := h.db.QueryRow(
		"INSERT INTO users (email) VALUES ($1) RETURNING user_id, email, created_at",
		email,
	).Scan(&user.UserID, &user.Email, &user.CreatedAt)
	return &user, err
}

func (h *Handler) GetUserByEmail(email string) (*User, error) {
	var user User
	err := h.db.QueryRow(
		"SELECT user_id, email, created_at FROM users WHERE email = $1",
		email,
	).Scan(&user.UserID, &user.Email, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (h *Handler) SaveArmstrongNumber(userID int, thenNumber int) (*ArmstrongNumber, error) {
	var arm ArmstrongNumber
	err := h.db.QueryRow(
		"INSERT INTO armstrong_numbers (user_id, thennumber) VALUES ($1, $2) RETURNING id, user_id, thennumber, created_at",
		userID, thenNumber,
	).Scan(&arm.ID, &arm.UserID, &arm.ThenNumber, &arm.CreatedAt)
	return &arm, err
}

// Handler registration
func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:       db,
		validate: validator.New(),
	}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/health", h.handleHealthCheck).Methods("GET")
	router.HandleFunc("/users", h.handleCreateUser).Methods("POST")
	router.HandleFunc("/register", h.handleRegister).Methods("POST")
	router.HandleFunc("/login", h.handleLogin).Methods("POST")
	router.HandleFunc("/users/me", h.AuthMiddleware(h.handleGetUser)).Methods("GET")
	router.HandleFunc("/armstrong", h.AuthMiddleware(h.handleCheckArmstrong)).Methods("POST")
	router.HandleFunc("/armstrong/my", h.AuthMiddleware(h.handleGetMyArmstrong)).Methods("GET")
	router.HandleFunc("/admin/users", h.AdminMiddleware(h.handleGetAllUsers)).Methods("GET")
}

func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AdminMiddleware ensures the user is an admin before allowing access.
func (h *Handler) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !claims.IsAdmin {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user by email
	user, err := h.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Compare password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.UserID, user.IsAdmin)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Save user
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING user_id, created_at`
	err = h.db.QueryRow(query, user.Email, hashedPassword).Scan(&user.UserID, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	user.Password = "" // Don't send password back
	json.NewEncoder(w).Encode(user)
}

func (h *Handler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	createdUser, err := h.CreateUser(user.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUser)
}

func (h *Handler) handleArmstrongNumber(w http.ResponseWriter, r *http.Request) {
	var arm ArmstrongNumber
	if err := json.NewDecoder(r.Body).Decode(&arm); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedArm, err := h.SaveArmstrongNumber(arm.UserID, arm.ThenNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(savedArm)
}

func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	err := h.db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "up",
		"database": "connected",
	})
}

// handleGetUser returns the authenticated user's information.
func (h *Handler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	var user User
	err := h.db.QueryRow(
		"SELECT user_id, email, created_at, is_admin FROM users WHERE user_id = $1",
		userID,
	).Scan(&user.UserID, &user.Email, &user.CreatedAt, &user.IsAdmin)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Password = "" // Do not expose password
	json.NewEncoder(w).Encode(user)
}

// handleCheckArmstrong checks if a number is an Armstrong number and saves it if true.
func (h *Handler) handleCheckArmstrong(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Number int `json:"number"`
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Armstrong number check
	n := req.Number
	sum := 0
	numDigits := 0
	for t := n; t > 0; t /= 10 {
		numDigits++
	}
	for t := n; t > 0; t /= 10 {
		d := t % 10
		pow := 1
		for i := 0; i < numDigits; i++ {
			pow *= d
		}
		sum += pow
	}

	if sum != n {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":    n,
			"armstrong": false,
		})
		return
	}

	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}

	// Save Armstrong number
	arm, err := h.SaveArmstrongNumber(userID, n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"number":    n,
		"armstrong": true,
		"record":    arm,
	})
}
