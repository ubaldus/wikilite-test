# WikiLite API Documentation

## Base URL
```
http://{host}:{port}/api
```
Default: `http://localhost:35248/api`

## API Endpoints

### 1. Combined Search
Performs a search across lexical and semantic (if enabled).

**Endpoint:** `/search`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search?query=linux&limit=5
```

#### POST Request
```json
POST /api/search
Content-Type: application/json

{
  "query": "linux",
  "limit": 5
}
```

#### Response
```json
{
  "status": "success",
  "time": 1.234,
  "results": [
    {
      "article_id": 123,
      "title": "Linux",
      "text": "Linux was created in 1991...",
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
GET /api/search/title?query=linux&limit=5
```

#### POST Request
```json
POST /api/search/title
Content-Type: application/json

{
  "query": "linux",
  "limit": 5
}
```

### 3. Lexical Search
Searches using full-text search.

**Endpoint:** `/search/lexical`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search/lexical?query=linux&limit=5
```

#### POST Request
```json
POST /api/search/lexical
Content-Type: application/json

{
  "query": "linux",
  "limit": 5
}
```

### 4. Semantic Search
Searches using vector embeddings. Requires AI to be enabled.

**Endpoint:** `/search/semantic`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search query string
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search/semantic?query=linux&limit=5
```

#### POST Request
```json
POST /api/search/semantic
Content-Type: application/json

{
  "query": "linux",
  "limit": 5
}
```

### 5. Search Word Distance
Calculate the Levenshtein distance between the provided word and the entries in the internal database vocabulary to find the closest match.

**Endpoint:** `/search/distance`  
**Methods:** GET, POST

#### Parameters
- `query` (required): Search word
- `limit` (optional): Maximum number of results (default: 5)

#### GET Request
```
GET /api/search/distance?query=linux&limit=5
```

#### POST Request
```json
POST /api/search/distance
Content-Type: application/json

{
  "query": "linux",
  "limit": 5
}
```


### 6. Get Article
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
  "time": 1.234,
  "article": {
    "id": 123,
    "title": "Linux",
    "entity": "Q388",
    "sections": [
      {
        "id": 1234,
        "title": "History",
        "texts": [
          "Linux was created in 1991...",
          "Linus Torvalds began working on Linux while studying..."
        ]
      },
      {
        "id": 12345,
        "title": "Design Philosophy",
        "texts": [
          "Linux follows Unix philosophy...",
          "The kernel is designed to be modular..."
        ]
      }
    ]
  }
}
```

## Common Response Format

### Success Response
```json
{
  "status": "success",
  "time": 1.234,
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
curl 'http://localhost:35248/api/search?query=linux&limit=5'
```

2. Title Search:
```bash
curl -X POST http://localhost:35248/api/search/title \
  -H 'Content-Type: application/json' \
  -d '{"query": "linux"}'
```

3. Get Article:
```bash
curl 'http://localhost:35248/api/article?id=123'
```

## Configuration Requirements

### Sematic Search
To use semantic search, you have two options:
1. **Remote Server**:  
  You can pass the remote server URL using the `--ai-api-url` flag. This allows the system to connect to a remote server where the vector search functionality is hosted.
2. **Local Model**:
  Alternatively, you can use a local GGUF model file. The model file must have the same name as the AI model (`--ai-model`) with `.gguf` extension. The file should be located in the same directory as the executable or as stated by `--ai-model-path`.

## Notes
- All search endpoints support both GET and POST methods
- The `limit` parameter is shared across all search types
- Semantic search requires additional configuration and services
- Results are deduplicated across search types in combined search

