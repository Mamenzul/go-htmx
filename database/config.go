package database

import (
	"database/sql"
	"log"

	_ "github.com/tursodatabase/go-libsql"
)

// DBConnection wraps the database connection
type DBConnection struct {
	db *sql.DB
}

// NewDBConnection initializes and returns a new database connection
func NewDBConnection() (*DBConnection, error) {
	dbURL := "file:./local.db"

	db, err := sql.Open("libsql", dbURL)
	if err != nil {
		log.Printf("Failed to connect to database %s", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close() // Clean up if ping fails
		return nil, err
	}

	log.Println("Successfully connected to database")
	return &DBConnection{db: db}, nil
}

// GetDB returns the underlying *sql.DB instance
func (c *DBConnection) GetDB() *sql.DB {
	return c.db
}

// Close closes the database connection
func (c *DBConnection) Close() error {
	log.Println("Closing database connection")
	return c.db.Close()
}
