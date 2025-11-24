package database

import (
	"database/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
)

func ConnectDB() *sql.DB {
	db, err := sql.Open("sqlite", "./mangahub.db")
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	// Create users table
	createUserTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createUserTable)
	if err != nil {
		log.Fatal("Failed to create users table:", err)
	}

	// Create manga table
	createMangaTable := `
	CREATE TABLE IF NOT EXISTS manga (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT,
		genres TEXT,
		status TEXT,
		total_chapters INTEGER DEFAULT 0,
		description TEXT,
		cover_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createMangaTable)
	if err != nil {
		log.Fatal("Failed to create manga table:", err)
	}

	// Create user_library table
	createLibraryTable := `
	CREATE TABLE IF NOT EXISTS user_library (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		manga_id TEXT NOT NULL,
		status TEXT DEFAULT 'plan_to_read',
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE,
		UNIQUE(user_id, manga_id)
	);`
	_, err = db.Exec(createLibraryTable)
	if err != nil {
		log.Fatal("Failed to create user_library table:", err)
	}

	// Create user_progress table
	createProgressTable := `
	CREATE TABLE IF NOT EXISTS user_progress (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		manga_id TEXT NOT NULL,
		chapter INTEGER DEFAULT 0,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (manga_id) REFERENCES manga(id) ON DELETE CASCADE,
		UNIQUE(user_id, manga_id)
	);`
	_, err = db.Exec(createProgressTable)
	if err != nil {
		log.Fatal("Failed to create user_progress table:", err)
	}

	return db
}
