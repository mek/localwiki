package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

// loggingMiddleware logs all incoming HTTP requests (method, path, remote address)
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// Page represents a wiki page stored in the database.
type Page struct {
	ID        int       `json:"id"`         // Unique page ID
	Title     string    `json:"title"`      // Page title
	Content   string    `json:"content"`    // Markdown content
	CreatedAt time.Time `json:"created_at"` // Creation timestamp
	UpdatedAt time.Time `json:"updated_at"` // Last update timestamp
}

const HomePageTitle = "Home"

// WikiServer handles HTTP requests and database operations for the wiki.
type WikiServer struct {
	db *sql.DB // SQLite database connection
}

// is DacFormat - check if a page is a DAC page
func isDacFormat(content string) bool {
	matched, _ := regexp.MatchString(`<<[^>]+>>=`, content)
	hasTerminator := strings.Contains(content, "\n@")
	return matched && hasTerminator
}

// renderDac - shells out to run dac for weaving
func (s *WikiServer) renderDac(dacSource string) (string, error) {

	// create a temp file for the DAC source
	tmpfile, err := os.CreateTemp("", "dac-*.dac")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	// write the DAC source to the temp file
	if _, err := tmpfile.Write([]byte(dacSource)); err != nil {
		return "", fmt.Errorf("failed to write DAC source to temporary file: %w", err)
	}

	cmd := exec.Command("dac", "-w", tmpfile.Name())
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run dac: %w", err)
	}

	return string(out), nil
}

func (s *WikiServer) handleWeavePage(w http.ResponseWriter, r *http.Request) {

	s.enableCORS(w, r)

	vars := mux.Vars(r)
	title := vars["title"]

	var content string
	err := s.db.QueryRow(
		"SELECT content FROM pages WHERE title = ?",
		title,
	).Scan(&content)

	if err == sql.ErrNoRows {
		s.writeJSONError(w, "Page not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.writeJSONError(w, "Failed to fetch page", http.StatusInternalServerError)
		return
	}

	if !isDacFormat(content) {
		s.writeJSONError(w, "Page is not a DAC page", http.StatusBadRequest)
		return
	}

	rendered, err := s.renderDac(content)
	if err != nil {
		s.writeJSONError(w, fmt.Sprintf("Failed to weave page %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(rendered))

}
func (s *WikiServer) tangleChunk(dacSource, chunkName string) (string, error) {

	// create a temp file for the DAC source
	tmpfile, err := os.CreateTemp("", "dac-*.dac")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	// write the DAC source to the temp file
	if _, err := tmpfile.Write([]byte(dacSource)); err != nil {
		return "", fmt.Errorf("failed to write DAC source to temporary file: %w", err)
	}

	cmd := exec.Command("dac", "-t", "-R", chunkName, tmpfile.Name())
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run dac: %w", err)
	}

	return string(out), nil
}

// handleRawPage - returns raw page content
func (s *WikiServer) handleRawPage(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	vars := mux.Vars(r)
	title := vars["title"]

	var page Page
	err := s.db.QueryRow(
		"SELECT content FROM pages WHERE title = ?",
		title,
	).Scan(&page.Content)

	if err == sql.ErrNoRows {
		s.writeJSONError(w, "Page not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.writeJSONError(w, "Failed to fetch page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, page.Content)
}

// handleTangelChunk - extract a specific check from a DAC page
func (s *WikiServer) handleTangleChunk(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	vars := mux.Vars(r)
	title := vars["title"]
	chunk := vars["chunk"]

	var content string
	err := s.db.QueryRow(
		"SELECT content FROM pages WHERE title = ?",
		title,
	).Scan(&content)

	if err == sql.ErrNoRows {
		s.writeJSONError(w, "Page not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.writeJSONError(w, "Failed to fetch page", http.StatusInternalServerError)
		return
	}

	if !isDacFormat(content) {
		s.writeJSONError(w, "Page is not a DAC page", http.StatusBadRequest)
		return
	}

	tangled, err := s.tangleChunk(content, chunk)
	if err != nil {
		s.writeJSONError(w, fmt.Sprintf("Failed to tangle chunk %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, tangled)
}

// validatePageTitle checks if a page title is valid.
// Returns an error message if invalid, empty string if valid.
func validatePageTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "Title is required"
	}
	if len(title) > 100 {
		return "Title too long (max 100 characters)"
	}
	for _, c := range title {
		if !(c == ' ' || c == '-' || c == '_' ||
			(c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9')) {
			return "Title contains invalid characters (allowed: letters, numbers, space, dash, underscore)"
		}
	}
	return ""
}

// NewWikiServer initializes the WikiServer, database, and seeds default data if needed.
func NewWikiServer() (*WikiServer, error) {
	// Ensure data directory exists
	if err := os.MkdirAll("./data", 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", "./data/wiki.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test database connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	server := &WikiServer{db: db}
	// Initialize database schema
	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	server.seedDefaultData()
	return server, nil
}

// DB schema initialization moved to db.go for modularity.

// seedDefaultData creates the default Home page if it does not exist.
// Reads the content from static/home.md.
func (s *WikiServer) seedDefaultData() {
	// Check if Home page exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM pages WHERE title = ?", HomePageTitle).Scan(&count)
	if err != nil {
		log.Printf("Error checking for Home page: %v", err)
		return
	}

	if count == 0 {
		// Read default content from static/home.md
		contentBytes, err := os.ReadFile("./static/home.md")
		if err != nil {
			log.Printf("Error reading default Home page content: %v", err)
			return
		}
		defaultContent := string(contentBytes)
		_, err = s.db.Exec(
			"INSERT INTO pages (title, content) VALUES (?, ?)",
			HomePageTitle, defaultContent,
		)
		if err != nil {
			log.Printf("Error creating default Home page: %v", err)
		}
	}
}

// enableCORS sets CORS headers for API responses.
func (s *WikiServer) enableCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// handleOptions responds to CORS preflight requests.
func (s *WikiServer) handleOptions(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)
	w.WriteHeader(http.StatusOK)
}

// getAllPages returns a list of all wiki pages (metadata only).
// Responds with JSON array of Page objects (without content).
func (s *WikiServer) getAllPages(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	rows, err := s.db.Query("SELECT id, title, created_at, updated_at FROM pages ORDER BY title")
	if err != nil {
		s.writeJSONError(w, "Failed to fetch pages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PageMeta struct {
		ID        int       `json:"id"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	var pages []PageMeta
	for rows.Next() {
		var page PageMeta
		err := rows.Scan(&page.ID, &page.Title, &page.CreatedAt, &page.UpdatedAt)
		if err != nil {
			continue
		}
		pages = append(pages, page)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pages)
}

// getPage returns the full content of a specific wiki page.
// Responds with a JSON Page object.
func (s *WikiServer) getPage(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	vars := mux.Vars(r)
	title := vars["title"]

	var page Page
	err := s.db.QueryRow(
		"SELECT id, title, content, created_at, updated_at FROM pages WHERE title = ?",
		title,
	).Scan(&page.ID, &page.Title, &page.Content, &page.CreatedAt, &page.UpdatedAt)

	if err == sql.ErrNoRows {
		s.writeJSONError(w, "Page not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.writeJSONError(w, "Failed to fetch page", http.StatusInternalServerError)
		return
	}

	isDac := isDacFormat(page.Content)

	response := map[string]interface{}{
		"id":         page.ID,
		"title":      page.Title,
		"content":    page.Content,
		"created_at": page.CreatedAt,
		"updated_at": page.UpdatedAt,
		"is_dac":     isDac,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createPage creates a new wiki page from JSON request body.
// Validates title and content, returns created Page as JSON.
func (s *WikiServer) createPage(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	var page Page
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		s.writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	page.Title = strings.TrimSpace(page.Title)
	if errMsg := validatePageTitle(page.Title); errMsg != "" {
		s.writeJSONError(w, errMsg, http.StatusBadRequest)
		return
	}

	result, err := s.db.Exec(
		"INSERT INTO pages (title, content) VALUES (?, ?)",
		page.Title, page.Content,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			s.writeJSONError(w, "Page already exists", http.StatusConflict)
			return
		}
		s.writeJSONError(w, "Failed to create page", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	page.ID = int(id)
	page.CreatedAt = time.Now()
	page.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(page)
}

// updatePage updates the content of an existing wiki page.
// Only updates content, title cannot be changed (uses title from URL).
func (s *WikiServer) updatePage(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	vars := mux.Vars(r)
	title := vars["title"]

	var requestBody struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		s.writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := s.db.Exec(
		"UPDATE pages SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE title = ?",
		requestBody.Content, title,
	)
	if err != nil {
		s.writeJSONError(w, "Failed to update page", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.writeJSONError(w, "Page not found", http.StatusNotFound)
		return
	}

	// Return updated page
	s.getPage(w, r)
}

// deletePage deletes a wiki page by title, except the Home page.
// Responds with 204 No Content on success.
func (s *WikiServer) deletePage(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	vars := mux.Vars(r)
	title := vars["title"]

	if title == HomePageTitle {
		s.writeJSONError(w, "Cannot delete Home page", http.StatusBadRequest)
		return
	}

	result, err := s.db.Exec("DELETE FROM pages WHERE title = ?", title)
	if err != nil {
		s.writeJSONError(w, "Failed to delete page", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.writeJSONError(w, "Page not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// searchPages searches for pages by title or content matching the query.
// Responds with JSON array of Page objects (without content).
func (s *WikiServer) searchPages(w http.ResponseWriter, r *http.Request) {
	s.enableCORS(w, r)

	query := r.URL.Query().Get("q")
	if query == "" {
		s.writeJSONError(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	searchQuery := "%" + query + "%"
	rows, err := s.db.Query(
		"SELECT id, title, created_at, updated_at FROM pages WHERE title LIKE ? OR content LIKE ? ORDER BY title",
		searchQuery, searchQuery,
	)
	if err != nil {
		s.writeJSONError(w, "Failed to search pages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var page Page
		err := rows.Scan(&page.ID, &page.Title, &page.CreatedAt, &page.UpdatedAt)
		if err != nil {
			continue
		}
		pages = append(pages, page)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pages)
}

// writeJSONError writes a standardized JSON error response for API handlers.
func (s *WikiServer) writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// main is the entry point for the wiki server application.
func main() {
	server, err := NewWikiServer()
	if err != nil {
		log.Printf("Error initializing wiki server: %v\n", err)
		log.Println("Please check that:")
		log.Println("  - You have write permissions to the ./data directory")
		log.Println("  - SQLite is properly installed")
		log.Println("  - No other process is using the database file")
		os.Exit(1)
	}
	defer server.db.Close()

	r := mux.NewRouter()

	// API routes - must be registered first and more specific
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/pages", server.getAllPages).Methods("GET")
	api.HandleFunc("/pages", server.createPage).Methods("POST")
	api.HandleFunc("/pages/search", server.searchPages).Methods("GET")
	api.HandleFunc("/pages/{title}", server.getPage).Methods("GET")
	api.HandleFunc("/pages/{title}", server.updatePage).Methods("PUT")
	api.HandleFunc("/pages/{title}", server.deletePage).Methods("DELETE")

	// Dac-specific endpoints
	api.HandleFunc("/pages/{title}/raw", server.handleRawPage).Methods("GET")
	api.HandleFunc("/pages/{title}/weave", server.handleWeavePage).Methods("GET")
	api.HandleFunc("/pages/{title}/tangle/{chunk}", server.handleTangleChunk).Methods("GET")

	// Handle preflight requests
	api.HandleFunc("/pages", server.handleOptions).Methods("OPTIONS")
	api.HandleFunc("/pages/{title}", server.handleOptions).Methods("OPTIONS")

	// Serve static files - this should be last to catch all non-API routes
	staticDir := "./static/"
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		log.Println("Static directory not found, serving API only")
	} else {
		// Serve index.html for root path
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		}).Methods("GET")

		// Serve all other static files
		r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir))).Methods("GET")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Wiki server starting on port %s...\n", port)
	fmt.Printf("Access your wiki at: http://localhost:%s\n", port)
	fmt.Printf("API available at: http://localhost:%s/api/pages\n", port)

	// Add logging middleware to router
	loggedRouter := loggingMiddleware(r)
	if err := http.ListenAndServe(":"+port, loggedRouter); err != nil {
		log.Printf("Server failed to start: %v\n", err)
		log.Printf("Please check that port %s is not already in use\n", port)
		os.Exit(1)
	}
}
