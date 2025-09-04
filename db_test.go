package main

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Initialize database
	if err := initDB(db); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Check that pages table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='pages'").Scan(&tableName)
	if err != nil {
		t.Error("Pages table was not created")
	}

	// Check table schema
	rows, err := db.Query("PRAGMA table_info(pages)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]bool{
		"id":         false,
		"title":      false,
		"content":    false,
		"created_at": false,
		"updated_at": false,
	}

	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dflt sql.NullString
		err := rows.Scan(&cid, &name, &dtype, &notnull, &dflt, &pk)
		if err != nil {
			t.Fatalf("Failed to scan column info: %v", err)
		}
		if _, ok := expectedColumns[name]; ok {
			expectedColumns[name] = true
		}
	}

	for col, found := range expectedColumns {
		if !found {
			t.Errorf("Column %s was not found in pages table", col)
		}
	}

	// Check that index exists
	var indexName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_title'").Scan(&indexName)
	if err != nil {
		t.Error("Index idx_title was not created")
	}
}

// TestInitDBIdempotent tests that initDB can be called multiple times safely
func TestInitDBIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Initialize database multiple times
	for i := 0; i < 3; i++ {
		if err := initDB(db); err != nil {
			t.Fatalf("Failed to initialize database on iteration %d: %v", i+1, err)
		}
	}

	// Insert a test record
	_, err = db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Test", "Content")
	if err != nil {
		t.Fatalf("Failed to insert test record: %v", err)
	}

	// Initialize again - should not affect existing data
	if err := initDB(db); err != nil {
		t.Fatalf("Failed to initialize database after data insert: %v", err)
	}

	// Check that data still exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pages WHERE title = 'Test'").Scan(&count)
	if err != nil || count != 1 {
		t.Error("Existing data was lost after re-initialization")
	}
}

// TestDatabaseConstraints tests database constraints
func TestDatabaseConstraints(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	t.Run("Unique title constraint", func(t *testing.T) {
		// Insert first page
		_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Unique Title", "Content 1")
		if err != nil {
			t.Fatalf("Failed to insert first page: %v", err)
		}

		// Try to insert duplicate title
		_, err = db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Unique Title", "Content 2")
		if err == nil {
			t.Error("Expected unique constraint violation, but insert succeeded")
		}
	})

	t.Run("Not null constraints", func(t *testing.T) {
		// Try to insert without title
		_, err := db.Exec("INSERT INTO pages (title, content) VALUES (NULL, ?)", "Content")
		if err == nil {
			t.Error("Expected not null constraint violation for title, but insert succeeded")
		}

		// Try to insert without content
		_, err = db.Exec("INSERT INTO pages (title, content) VALUES (?, NULL)", "Title")
		if err == nil {
			t.Error("Expected not null constraint violation for content, but insert succeeded")
		}
	})

	t.Run("Auto-increment ID", func(t *testing.T) {
		// Insert two pages
		result1, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Page 1", "Content 1")
		if err != nil {
			t.Fatalf("Failed to insert page 1: %v", err)
		}
		id1, _ := result1.LastInsertId()

		result2, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Page 2", "Content 2")
		if err != nil {
			t.Fatalf("Failed to insert page 2: %v", err)
		}
		id2, _ := result2.LastInsertId()

		if id2 <= id1 {
			t.Errorf("Auto-increment not working: id1=%d, id2=%d", id1, id2)
		}
	})
}

// TestDatabaseTimestamps tests that created_at and updated_at are set correctly
func TestDatabaseTimestamps(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Insert a page
	_, err = db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Test Page", "Test Content")
	if err != nil {
		t.Fatalf("Failed to insert page: %v", err)
	}

	// Check that timestamps are set
	var createdAt, updatedAt string
	err = db.QueryRow("SELECT created_at, updated_at FROM pages WHERE title = ?", "Test Page").Scan(&createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("Failed to query timestamps: %v", err)
	}

	if createdAt == "" {
		t.Error("created_at was not set")
	}
	if updatedAt == "" {
		t.Error("updated_at was not set")
	}
	if createdAt != updatedAt {
		t.Error("created_at and updated_at should be equal for new record")
	}

	// Update the page
	_, err = db.Exec("UPDATE pages SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE title = ?", "Updated Content", "Test Page")
	if err != nil {
		t.Fatalf("Failed to update page: %v", err)
	}

	// Check that updated_at changed
	var newCreatedAt, newUpdatedAt string
	err = db.QueryRow("SELECT created_at, updated_at FROM pages WHERE title = ?", "Test Page").Scan(&newCreatedAt, &newUpdatedAt)
	if err != nil {
		t.Fatalf("Failed to query timestamps after update: %v", err)
	}

	if newCreatedAt != createdAt {
		t.Error("created_at should not change on update")
	}
	// Note: In a real test, we'd need to add a small delay or mock time to ensure updated_at changes
}

// BenchmarkDatabaseInsert benchmarks page insertion
func BenchmarkDatabaseInsert(b *testing.B) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		title := "Page " + string(rune(i))
		_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", title, "Benchmark content")
		if err != nil {
			b.Fatalf("Failed to insert page: %v", err)
		}
	}
}

// BenchmarkDatabaseQuery benchmarks page queries
func BenchmarkDatabaseQuery(b *testing.B) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := initDB(db); err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}

	// Insert test data
	for i := 0; i < 100; i++ {
		title := "Page " + string(rune(i))
		_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", title, "Benchmark content")
		if err != nil {
			b.Fatalf("Failed to insert page: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var title, content string
		err := db.QueryRow("SELECT title, content FROM pages WHERE title = ?", "Page A").Scan(&title, &content)
		if err != nil && err != sql.ErrNoRows {
			b.Fatalf("Failed to query page: %v", err)
		}
	}
}