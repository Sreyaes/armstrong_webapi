package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Connection parameters
	connStr := "host=localhost port=5432 user=postgres password=Sreya@2004 dbname=armstrong sslmode=disable"

	fmt.Println("Attempting to connect to PostgreSQL...")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database: %v\n", err)
	}
	defer db.Close()

	// Set connection timeout
	db.SetConnMaxLifetime(5 * time.Second)

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v\n", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")
}
