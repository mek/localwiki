package main

import (
	"database/sql"
	"fmt"
)

// initDB creates the pages table and index if they do not exist.
func initDB(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT UNIQUE NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS page_versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		page_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (page_id) REFERENCES pages(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_title ON pages(title);
	CREATE INDEX IF NOT EXISTS idx_page_versions_page_id ON page_versions(page_id);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create trigger to save page versions on update
	triggerSQL := `
	CREATE TRIGGER IF NOT EXISTS save_page_version
	AFTER UPDATE ON pages
	FOR EACH ROW
	BEGIN
		INSERT INTO page_versions (page_id, title, content, created_at)
		VALUES (OLD.id, OLD.title, OLD.content, OLD.updated_at);
	END;
	`

	if _, err := db.Exec(triggerSQL); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}
	return nil
}

// Additional DB logic (CRUD) can be moved here for modularity.
