# WikiLite API Documentation

## Base URL
```
http://{host}:{port}/api
```
Default: `http://localhost:35248/api`

## API Endpoints

### 1. Combined Search
Performs a search across titles, content, and vectors (if enabled).

**Endpoint:** `/search`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search?query=python&limit=5
```

#### POST Request
```json
POST /api/search
Content-Type: application/json

{
  "query": "python",
  "limit": 5
}
```

#### Response
```json
{
  "status": "success",
  "results": [
    {
      "article": 123,
      "title": "Python (programming language)",
      "entity": "Python",
      "section": "History",
      "text": "Python was conceived in the late 1980s...",
      "type": "T",
      "power": 1.234
    }
  ]
}
```

### 2. Title Search
Searches only article titles using full-text search.

**Endpoint:** `/search/title`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search/title?query=python&limit=5
```

#### POST Request
```json
POST /api/search/title
Content-Type: application/json

{
  "query": "python",
  "limit": 5
}
```

### 3. Content Search
Searches article content using full-text search.

**Endpoint:** `/search/content`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search/content?query=python&limit=5
```

#### POST Request
```json
POST /api/search/content
Content-Type: application/json

{
  "query": "python",
  "limit": 5
}
```

### 4. Vector Search
Searches using vector embeddings. Requires AI and Qdrant to be enabled.

**Endpoint:** `/search/vectors`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search/vectors?query=python&limit=5
```

#### POST Request
```json
POST /api/search/vectors
Content-Type: application/json

{
  "query": "python",
  "limit": 5
}
```

### 5. Get Article
Retrieves a complete article by ID.

**Endpoint:** `/article`  
**Methods:** GET, POST

#### Parameters
- `id` (required): Article ID

#### GET Request
```
GET /api/article?id=123
```

#### POST Request
```json
POST /api/article
Content-Type: application/json

{
  "id": 123
}
```

#### Response
```json
{
  "status": "success",
  "article": [
    {
      "article": 123,
      "title": "Python (programming language)",
      "entity": "Python",
      "section": "History",
      "text": "Python was conceived in the late 1980s..."
    }
  ]
}
```

## Common Response Format

### Success Response
```json
{
  "status": "success",
  "results": [...],  // For search endpoints
  "article": [...]   // For article endpoint
}
```

### Error Response
```json
{
  "status": "error",
  "message": "Error description"
}
```

## Result Types
Search results include a `type` field indicating the source:
- `T`: Title match
- `C`: Content match
- `V`: Vector match

## Error Codes
The API uses standard HTTP status codes:
- `200`: Success
- `400`: Bad Request (invalid parameters)
- `500`: Internal Server Error

## Command Line Examples

### Using curl

1. Combined Search:
```bash
curl 'http://localhost:35248/api/search?query=python&limit=5'
```

2. Title Search:
```bash
curl -X POST http://localhost:35248/api/search/title \
  -H 'Content-Type: application/json' \
  -d '{"query": "python"}'
```

3. Get Article:
```bash
curl 'http://localhost:35248/api/article?id=123'
```

## Configuration Requirements

### Vector Search
To use vector search:
1. Start the server with AI enabled: `--ai`
2. Enable Qdrant: `--qdrant`
3. Configure AI/Qdrant host/port if not using defaults

Example:
```bash
./wikilite --ai --qdrant --web
```


## Notes
- All search endpoints support both GET and POST methods
- The `limit` parameter is shared across all search types
- Vector search requires additional configuration and services
- Results are deduplicated across search types in combined search
