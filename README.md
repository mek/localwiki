# Personal Wiki

A lightweight, portable wiki application built with Go and SQLite. Perfect for personal documentation, note-taking, and knowledge management.

## Features

- **Markdown Support**: Write your documentation in clean, readable markdown
- **Wiki-style Linking**: Use `[[Page Name]]` syntax to link between pages
- **Footnotes**: Add footnotes with `[^1]` syntax for detailed references
- **Real-time Preview**: See your formatted content as you type
- **Search**: Full-text search across all pages
- **Portable**: SQLite database makes it easy to move between machines
- **Docker Ready**: Simple deployment with Docker and docker-compose
- **REST API**: Clean API for programmatic access
- **Responsive Design**: Works on desktop, tablet, and mobile

## Quick Start

### Option 1: Kubernetes with Rancher Desktop

Deploy to your local Kubernetes cluster using Rancher Desktop with Traefik:

```bash
# Build and deploy
cd k8s
./deploy.sh

# Access your wiki at http://wiki.rancher.localhost
```

Or manually:

```bash
# Build Docker image
docker build -t wiki:latest .

# Apply Kubernetes manifests
kubectl apply -k k8s/

# Check deployment status
kubectl get all -n wiki

# Access at http://wiki.rancher.localhost
```

**Features:**
- Automatic ingress with Traefik at `wiki.rancher.localhost`
- Persistent storage using PVC
- Resource limits and health checks
- Easy cleanup with `kubectl delete -k k8s/`

### Option 2: Docker (Simple)

```bash
# Clone or create the project files
mkdir personal-wiki && cd personal-wiki

# Create docker-compose.yml (content provided above)
# Start the application
docker-compose up -d

# Access your wiki at http://localhost:8080
```

**Note:** The backend defaults to port `8083` if the `PORT` environment variable is not set. For consistency, set `PORT=8080` in your environment or Docker configuration.

### Option 2: Local Development

```bash
# Prerequisites: Go 1.21+ installed
git clone <your-repo>
cd personal-wiki

# Install dependencies
go mod tidy

# Run the application
go run main.go

# Access your wiki at http://localhost:8080
```

**Default Port:** If you do not set the `PORT` environment variable, the Go backend will run on port `8083` by default. Set `PORT=8080` to match the frontend and Docker setup.

## Code Structure

The project consists of a Go backend and a JavaScript frontend:

- **main.go**: Go backend server, REST API, SQLite integration
- **static/app.js**: Main frontend logic, markdown editor, API calls
- **static/index.html**: Frontend HTML template
- **static/style.css**: Frontend styles and layout
- **Dockerfile**: Docker build configuration
- **docker-compose.yaml**: Docker Compose setup
- **data/wiki.db**: SQLite database (auto-created)
- **README.md**: Project documentation

## API Documentation

The Go backend is documented with inline doc comments for all main types and functions. See `main.go` for details.

### Endpoints

- `GET /api/pages` - List all pages
- `GET /api/pages/{title}` - Get specific page
- `POST /api/pages` - Create new page
- `PUT /api/pages/{title}` - Update existing page
- `DELETE /api/pages/{title}` - Delete page
- `GET /api/pages/search?q={query}` - Search pages

## Usage

### Creating Pages

1. Type a page title in the "New page title" field
2. Click "Create" or press Enter
3. Start writing your content in markdown

### Wiki Links

Link to other pages using double brackets:

```markdown
Check out the [[Getting Started]] guide for more information.
```

Broken links (to pages that don't exist yet) appear in red. Click them to create the page.

### Footnotes

Add footnotes to provide additional context:

```markdown
This is a statement that needs clarification[^1].

[^1]: This is the footnote explanation.
```

### Markdown Features

All standard markdown features are supported:

- Headers (`#`, `##`, `###`)
- **Bold** and *italic* text
- `Code` and code blocks
- Lists and numbered lists
- Tables
- Blockquotes
- Links and images

## Configuration

### Environment Variables

- `PORT`: Server port (default: 8080)

### Data Persistence

All wiki data is stored in `./data/wiki.db`. To backup your wiki:

```bash
# Copy the database file
cp ./data/wiki.db ./wiki-backup.db

# Or if using Docker
docker cp personal-wiki_wiki_1:/app/data/wiki.db ./wiki-backup.db
```

To restore from backup:

```bash
# Stop the application first
docker-compose down

# Replace the database
cp ./wiki-backup.db ./data/wiki.db

# Restart
docker-compose up -d
```

## Moving Between Machines

### Method 1: Database File

1. Stop the wiki application
2. Copy the `data/wiki.db` file to your new machine
3. Place it in the same location in your new installation
4. Start the wiki application

### Method 2: Docker Volume

The docker-compose setup uses named volumes, making it easy to backup and restore:

```bash
# Create backup
docker run --rm -v personal-wiki_wiki_data:/data -v $(pwd):/backup alpine tar czf /backup/wiki-backup.tar.gz -C /data .

# Restore backup on new machine
docker run --rm -v personal-wiki_wiki_data:/data -v $(pwd):/backup alpine tar xzf /backup/wiki-backup.tar.gz -C /data
```

## Development Workflow

### Local Development

1. Make code changes in `main.go` (backend) or `static/` (frontend).
2. Restart the Go server after backend changes: `go run main.go`.
3. Frontend changes in `static/` are reflected automatically on refresh.
4. Use Docker for consistent environment: `docker-compose up --build`.

### Testing

The project includes comprehensive Go tests. Use the Makefile for easy testing:

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Generate coverage report
make test-coverage

# Generate HTML coverage report
make test-coverage-html

# Run tests with race detection
make test-race

# Run benchmarks
make test-benchmark

# Run all CI checks (format, vet, test)
make ci
```

Test files included:
- `main_test.go` - Tests for API endpoints and handlers
- `db_test.go` - Tests for database operations

Current test coverage: ~63% of statements

## Code Style

- **Go:** Use `gofmt` for formatting. Write clear doc comments for exported functions and types.
- **JavaScript:** Use standard ES6+ syntax. Consider using ESLint for style and error checking.

## Troubleshooting

### Port Already in Use

Change the port in docker-compose.yml:

```yaml
ports:
  - "9000:8080"  # Use port 9000 instead
```

### Database Issues

If you encounter database corruption:

1. Stop the application
2. Delete `data/wiki.db`
3. Restart the application (it will create a new database)
4. Restore from backup if available

### Permission Issues

Ensure the data directory is writable:

```bash
mkdir -p data
chmod 755 data
```

## License

MIT License - feel free to use this for personal or commercial projects.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.
