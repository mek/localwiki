package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

// TestValidatePageTitle tests the title validation function
func TestValidatePageTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{"Valid title", "Test Page", ""},
		{"Valid with numbers", "Page 123", ""},
		{"Valid with dash", "Test-Page", ""},
		{"Valid with underscore", "Test_Page", ""},
		{"Empty title", "", "Title is required"},
		{"Only spaces", "   ", "Title is required"},
		{"Too long", string(make([]byte, 101)), "Title too long (max 100 characters)"},
		{"Invalid chars", "Test@Page", "Title contains invalid characters (allowed: letters, numbers, space, dash, underscore)"},
		{"Special chars", "Test#$%", "Title contains invalid characters (allowed: letters, numbers, space, dash, underscore)"},
		{"Unicode chars", "Test™Page", "Title contains invalid characters (allowed: letters, numbers, space, dash, underscore)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePageTitle(tt.title)
			if result != tt.expected {
				t.Errorf("validatePageTitle(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := initDB(db); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return db
}

// TestNewWikiServer tests server initialization
func TestNewWikiServer(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	server, err := NewWikiServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.db.Close()

	// Check that data directory was created
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		t.Error("Data directory was not created")
	}

	// Check that database is accessible
	if err := server.db.Ping(); err != nil {
		t.Errorf("Database is not accessible: %v", err)
	}
}

// TestGetAllPages tests the getAllPages endpoint
func TestGetAllPages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &WikiServer{db: db}

	// Insert test data
	_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Test Page", "Test content")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create request and recorder
	req, err := http.NewRequest("GET", "/api/pages", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	// Call handler
	server.getAllPages(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response body
	var pages []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &pages); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(pages) != 1 {
		t.Errorf("Expected 1 page, got %d", len(pages))
	}

	if pages[0]["title"] != "Test Page" {
		t.Errorf("Expected title 'Test Page', got %v", pages[0]["title"])
	}
}

// TestGetPage tests the getPage endpoint
func TestGetPage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &WikiServer{db: db}

	// Insert test data
	_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Test Page", "Test content")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test successful get
	t.Run("Existing page", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/pages/Test%20Page", nil)
		req = mux.SetURLVars(req, map[string]string{"title": "Test Page"})
		rr := httptest.NewRecorder()

		server.getPage(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var page Page
		if err := json.Unmarshal(rr.Body.Bytes(), &page); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if page.Title != "Test Page" || page.Content != "Test content" {
			t.Errorf("Got unexpected page data: %+v", page)
		}
	})

	// Test non-existent page
	t.Run("Non-existent page", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/pages/NonExistent", nil)
		req = mux.SetURLVars(req, map[string]string{"title": "NonExistent"})
		rr := httptest.NewRecorder()

		server.getPage(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestCreatePage tests the createPage endpoint
func TestCreatePage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &WikiServer{db: db}

	tests := []struct {
		name       string
		payload    map[string]string
		wantStatus int
		wantError  string
	}{
		{
			name:       "Valid page",
			payload:    map[string]string{"title": "New Page", "content": "New content"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Empty title",
			payload:    map[string]string{"title": "", "content": "Content"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Title is required",
		},
		{
			name:       "Invalid title characters",
			payload:    map[string]string{"title": "Bad@Title", "content": "Content"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Title contains invalid characters",
		},
		{
			name:       "Title too long",
			payload:    map[string]string{"title": string(make([]byte, 101)), "content": "Content"},
			wantStatus: http.StatusBadRequest,
			wantError:  "Title too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", "/api/pages", bytes.NewReader(body))
			rr := httptest.NewRecorder()

			server.createPage(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}

			if tt.wantError != "" {
				var response map[string]string
				json.Unmarshal(rr.Body.Bytes(), &response)
				if response["error"] == "" || !bytes.Contains([]byte(response["error"]), []byte(tt.wantError)) {
					t.Errorf("Expected error containing %q, got %q", tt.wantError, response["error"])
				}
			}
		})
	}

	// Test duplicate page
	t.Run("Duplicate page", func(t *testing.T) {
		// First create a page
		body, _ := json.Marshal(map[string]string{"title": "Unique Page", "content": "Content"})
		req, _ := http.NewRequest("POST", "/api/pages", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		server.createPage(rr, req)

		// Try to create same page again
		req, _ = http.NewRequest("POST", "/api/pages", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		server.createPage(rr, req)

		if status := rr.Code; status != http.StatusConflict {
			t.Errorf("Handler returned wrong status code for duplicate: got %v want %v", status, http.StatusConflict)
		}
	})
}

// TestUpdatePage tests the updatePage endpoint
func TestUpdatePage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &WikiServer{db: db}

	// Insert test page
	_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Existing Page", "Old content")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Run("Update existing page", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"content": "Updated content"})
		req, _ := http.NewRequest("PUT", "/api/pages/Existing%20Page", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"title": "Existing Page"})
		rr := httptest.NewRecorder()

		server.updatePage(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify content was updated
		var content string
		err := db.QueryRow("SELECT content FROM pages WHERE title = ?", "Existing Page").Scan(&content)
		if err != nil || content != "Updated content" {
			t.Errorf("Content was not updated correctly: got %q", content)
		}
	})

	t.Run("Update non-existent page", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"content": "New content"})
		req, _ := http.NewRequest("PUT", "/api/pages/NonExistent", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"title": "NonExistent"})
		rr := httptest.NewRecorder()

		server.updatePage(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestDeletePage tests the deletePage endpoint
func TestDeletePage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &WikiServer{db: db}

	// Insert test pages
	_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Deletable Page", "Content")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	_, err = db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", "Home", "Home content")
	if err != nil {
		t.Fatalf("Failed to insert Home page: %v", err)
	}

	t.Run("Delete existing page", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/pages/Deletable%20Page", nil)
		req = mux.SetURLVars(req, map[string]string{"title": "Deletable Page"})
		rr := httptest.NewRecorder()

		server.deletePage(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
		}

		// Verify page was deleted
		var count int
		db.QueryRow("SELECT COUNT(*) FROM pages WHERE title = ?", "Deletable Page").Scan(&count)
		if count != 0 {
			t.Error("Page was not deleted")
		}
	})

	t.Run("Cannot delete Home page", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/pages/Home", nil)
		req = mux.SetURLVars(req, map[string]string{"title": "Home"})
		rr := httptest.NewRecorder()

		server.deletePage(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		// Verify Home page still exists
		var count int
		db.QueryRow("SELECT COUNT(*) FROM pages WHERE title = ?", "Home").Scan(&count)
		if count != 1 {
			t.Error("Home page was deleted (it shouldn't be)")
		}
	})

	t.Run("Delete non-existent page", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/pages/NonExistent", nil)
		req = mux.SetURLVars(req, map[string]string{"title": "NonExistent"})
		rr := httptest.NewRecorder()

		server.deletePage(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

// TestSearchPages tests the searchPages endpoint
func TestSearchPages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &WikiServer{db: db}

	// Insert test data
	testPages := []struct {
		title   string
		content string
	}{
		{"Go Programming", "Learn Go programming language"},
		{"Python Tutorial", "Python is a great language"},
		{"JavaScript Guide", "JavaScript for beginners"},
		{"Go Testing", "Testing in Go is easy"},
	}

	for _, p := range testPages {
		_, err := db.Exec("INSERT INTO pages (title, content) VALUES (?, ?)", p.title, p.content)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	tests := []struct {
		name         string
		query        string
		wantCount    int
		wantContains []string
	}{
		{
			name:         "Search by title",
			query:        "Go",
			wantCount:    2,
			wantContains: []string{"Go Programming", "Go Testing"},
		},
		{
			name:         "Search by content",
			query:        "language",
			wantCount:    2,
			wantContains: []string{"Go Programming", "Python Tutorial"},
		},
		{
			name:         "No results",
			query:        "Rust",
			wantCount:    0,
			wantContains: []string{},
		},
		{
			name:         "Case insensitive",
			query:        "python",
			wantCount:    1,
			wantContains: []string{"Python Tutorial"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/pages/search?q="+tt.query, nil)
			rr := httptest.NewRecorder()

			server.searchPages(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var results []Page
			if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("Expected %d results, got %d", tt.wantCount, len(results))
			}

			for _, want := range tt.wantContains {
				found := false
				for _, result := range results {
					if result.Title == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find page %q in results", want)
				}
			}
		})
	}

	t.Run("Missing query parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/pages/search", nil)
		rr := httptest.NewRecorder()

		server.searchPages(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}
