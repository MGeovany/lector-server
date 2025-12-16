# PDF Text Reader

A complete web application that transforms illegible PDF documents into highly readable, customizable text displays with full user management and document library features.

## Architecture

- **Backend**: Go API server with Supabase integration
- **Frontend**: SvelteKit with TypeScript and Tailwind CSS
- **Database**: Supabase (PostgreSQL with real-time features)
- **Authentication**: Supabase Auth
- **File Storage**: Supabase Storage

## Quick Start

### Prerequisites

- Go 1.21 or later
- Node.js 18 or later
- pnpm package manager
- Supabase account

## Development

### Backend (Go)

```bash
cd reader-go

# Development with hot reload
make dev

# Run without hot reload
make run

# Build
make build

# Run tests
make test

# Check environment
make check-env
```

## Project Structure

```
pdf-text-reader/
├── reader-go/                 # Go backend
│   ├── cmd/server/           # Application entry point
│   ├── internal/             # Internal packages
│   │   ├── config/          # Configuration management
│   │   ├── domain/          # Domain models and interfaces
│   │   ├── handler/         # HTTP handlers
│   │   ├── repository/      # Data access layer
│   │   └── service/         # Business logic
│   ├── pkg/                 # Shared packages
│   ├── scripts/             # Development scripts
│   └── supabase_schema.sql  # Database schema
├── reader-app/               # SvelteKit frontend
│   ├── src/
│   │   ├── lib/            # Shared libraries
│   │   │   ├── api/        # API client
│   │   │   └── stores/     # Svelte stores
│   │   └── routes/         # Application routes
│   └── scripts/            # Development scripts
└── dev-setup.sh            # Master setup script
```

## API Endpoints

### Authentication
All protected endpoints require a valid Supabase JWT token in the Authorization header.

### Documents
- `GET /api/v1/documents` - List user documents
- `POST /api/v1/documents` - Upload new document
- `GET /api/v1/documents/{id}` - Get specific document
- `DELETE /api/v1/documents/{id}` - Delete document
- `GET /api/v1/documents/search` - Search documents

### Preferences
- `GET /api/v1/preferences` - Get user preferences
- `PUT /api/v1/preferences` - Update preferences
- `GET /api/v1/preferences/reading-position/{documentId}` - Get reading position
- `PUT /api/v1/preferences/reading-position/{documentId}` - Update reading position

## Database Schema

The application uses three main tables:

- **documents**: Store PDF metadata and extracted content
- **user_preferences**: User reading preferences and settings
- **reading_positions**: Track reading progress across documents

All tables implement Row Level Security (RLS) to ensure users can only access their own data.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request


