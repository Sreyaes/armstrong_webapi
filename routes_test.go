package user

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	connStr := "host=localhost port=5432 user=postgres password=Sreya@2004 dbname=armstrong_test sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Add connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return db
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	h := NewHandler(db)
	user, err := h.CreateUser("test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email %s, got %s", "test@example.com", user.Email)
	}
}
