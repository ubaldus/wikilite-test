// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math"
)

// DBHandler manages database operations
type DBHandler struct {
	db *sql.DB
}

// Article output structure matching the JSON output
type OutputArticle struct {
	Title  string                   `json:"title"`
	Entity string                   `json:"entity"`
	Items  []map[string]interface{} `json:"items"`
	ID     int                      `json:"id"`
}

// TextEntry represents the hash-text pairs in items
type TextEntry struct {
	Hash string `json:"hash"`
	Text string `json:"text"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Article int
	Title   string
	Entity  string
	Text    string
	Section string
}

type ArticleResult struct {
	Title   string
	Entity  string
	Section string
	Text    string
	Article int
	BM25    float64
}

// NewDBHandler creates and initializes a new database connection
func NewDBHandler(dbPath string) (*DBHandler, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	handler := &DBHandler{db: db}
	if err := handler.initializeDB(); err != nil {
		db.Close()
		return nil, err
	}

	return handler, nil
}

// SaveArticle stores an article and its content in the database
func (h *DBHandler) SaveArticle(article OutputArticle) error {
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Insert article
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO articles (id, title, entity) VALUES (?, ?, ?)",
		article.ID, article.Title, article.Entity,
	)
	if err != nil {
		return fmt.Errorf("error inserting article: %v", err)
	}

	// Insert sections and their content
	for _, item := range article.Items {
		sub, _ := item["sub"].(string)
		pow, _ := item["pow"].(int)

		result, err := tx.Exec(
			"INSERT INTO sections (article_id, sub, pow) VALUES (?, ?, ?)",
			article.ID, sub, pow,
		)
		if err != nil {
			return fmt.Errorf("error inserting section: %v", err)
		}

		sectionID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("error getting section ID: %v", err)
		}

		textEntries, ok := item["text"].([]map[string]string)
		if !ok {
			return fmt.Errorf("invalid type for item['text']: %T", item["text"])
		}

		for _, entry := range textEntries {
			hash := entry["hash"]
			text := entry["text"]

			// Try to insert hash first, ignore if already exists
			result, err = tx.Exec(
				"INSERT OR IGNORE INTO hashes (hash, text) VALUES (?, ?)",
				hash, text,
			)
			if err != nil {
				return fmt.Errorf("error inserting hash: %v", err)
			}

			// Get hash_id (whether newly inserted or existing)
			var hashID int
			err = tx.QueryRow("SELECT id FROM hashes WHERE hash = ?", hash).Scan(&hashID)
			if err != nil {
				return fmt.Errorf("error getting hash ID: %v", err)
			}

			// Increment pow in hashes table
			_, err = tx.Exec("UPDATE hashes SET pow = pow + 1 WHERE id = ?", hashID)
			if err != nil {
				return fmt.Errorf("error updating hash pow: %v", err)
			}

			// Insert content with hash_id reference
			_, err = tx.Exec(
				"INSERT INTO content (section_id, hash_id) VALUES (?, ?)",
				sectionID, hashID,
			)
			if err != nil {
				return fmt.Errorf("error inserting content: %v", err)
			}
		}
	}

	return tx.Commit()
}

// Close closes the database connection
func (h *DBHandler) Close() error {
	return h.db.Close()
}

func (h *DBHandler) initializeDB() error {
	queries := []string{
		// Create 'articles' table
		`CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			entity TEXT NOT NULL
		)`,

		// Create 'sections' table with foreign key to 'articles'
		`CREATE TABLE IF NOT EXISTS sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER,
			sub TEXT,
			pow INTEGER DEFAULT 0,
			FOREIGN KEY(article_id) REFERENCES articles(id)
		)`,

		// Create 'hashes' table for basic storage
		`CREATE TABLE IF NOT EXISTS hashes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT UNIQUE NOT NULL,
			text TEXT NOT NULL,
			pow INTEGER DEFAULT 0,
			vectors BLOB
		)`,

		// Create FTS5 virtual table for text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS hash_search USING fts5(
			hash,
			text,
			content='hashes',
			content_rowid='id'
		)`,

		// Create 'content' table with foreign keys
		`CREATE TABLE IF NOT EXISTS content (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id INTEGER,
			hash_id INTEGER NOT NULL,
			FOREIGN KEY(section_id) REFERENCES sections(id),
			FOREIGN KEY(hash_id) REFERENCES hashes(id)
		)`,

		// Create triggers to maintain FTS index
		`CREATE TRIGGER IF NOT EXISTS hashes_ai AFTER INSERT ON hashes BEGIN
			INSERT INTO hash_search(rowid, hash, text) VALUES (new.id, new.hash, new.text);
		END`,

		`CREATE TRIGGER IF NOT EXISTS hashes_ad AFTER DELETE ON hashes BEGIN
			INSERT INTO hash_search(hash_search, rowid, hash, text) VALUES('delete', old.id, old.hash, old.text);
		END`,

		`CREATE TRIGGER IF NOT EXISTS hashes_au AFTER UPDATE ON hashes BEGIN
			INSERT INTO hash_search(hash_search, rowid, hash, text) VALUES('delete', old.id, old.hash, old.text);
			INSERT INTO hash_search(rowid, hash, text) VALUES (new.id, new.hash, new.text);
		END`,
	}

	for _, query := range queries {
		if _, err := h.db.Exec(query); err != nil {
			return fmt.Errorf("error executing query %s: %v", query, err)
		}
	}

	return nil
}

// SearchArticles performs a full-text search and returns the top results
func (h *DBHandler) SearchArticles(searchQuery string, limit int) ([]SearchResult, error) {
	sqlQuery := `
		WITH matched_hashes AS (
			SELECT rowid, text, bm25(hash_search) as relevance
			FROM hash_search
			WHERE hash_search MATCH ?
		)
		SELECT DISTINCT
			a.id as article_id,
			a.title,
			a.entity,
			s.sub as section,
			mh.text
		FROM matched_hashes mh
		JOIN hashes h ON h.id = mh.rowid
		JOIN content c ON c.hash_id = h.id
		JOIN sections s ON s.id = c.section_id
		JOIN articles a ON a.id = s.article_id
		ORDER BY h.pow ASC
		LIMIT ?
	`

	rows, err := h.db.Query(sqlQuery, searchQuery, limit*5)
	if err != nil {
		return nil, fmt.Errorf("search error: %v", err)
	}
	defer rows.Close()

	seenArticles := make(map[int]bool)
	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(
			&result.Article,
			&result.Title,
			&result.Entity,
			&result.Section,
			&result.Text,
		); err != nil {
			return nil, fmt.Errorf("error scanning result: %v", err)
		}
		if _, exists := seenArticles[result.Article]; !exists {
			results = append(results, result)
			seenArticles[result.Article] = true
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

func (h *DBHandler) GetArticle(articleID int) ([]ArticleResult, error) {
	sqlQuery := `
		SELECT 
    	a.title AS article_title,
			a.entity AS article_entity,
    	s.sub AS section_title,
    	h.text AS content
		FROM 
    	articles a
		JOIN 
    	sections s ON a.id = s.article_id
		JOIN 
    	content c ON s.id = c.section_id
		JOIN 
    	hashes h ON c.hash_id = h.id
		WHERE 
    	a.id = ?
		ORDER BY 
    	c.id;
    `

	rows, err := h.db.Query(sqlQuery, articleID)
	if err != nil {
		return nil, fmt.Errorf("article query error: %v", err)
	}
	defer rows.Close()

	var results []ArticleResult
	for rows.Next() {
		var result ArticleResult
		if err := rows.Scan(
			&result.Title,
			&result.Entity,
			&result.Section,
			&result.Text,
		); err != nil {
			return nil, fmt.Errorf("error scanning result: %v", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func Float32ToBlob(floats []float32) ([]byte, error) {
	bytes := make([]byte, len(floats)*4) // 4 bytes per float32
	for i, float := range floats {
		binary.LittleEndian.PutUint32(bytes[i*4:(i+1)*4], uint32(math.Float32bits(float)))
	}
	return bytes, nil
}

func BlobToFloat32(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("length of input bytes is not a multiple of 4")
	}
	numFloats := len(data) / 4
	floats := make([]float32, numFloats)

	for i := 0; i < numFloats; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		floats[i] = math.Float32frombits(bits)
	}

	return floats, nil
}

func (h *DBHandler) ProcessEmbeddings() error {
	for {
		sqlQuery := `SELECT hash, text FROM hashes WHERE vectors IS NULL LIMIT 1;`
		row := h.db.QueryRow(sqlQuery)

		var hash string
		var text string
		err := row.Scan(&hash, &text)
		if err != nil {
			if err == sql.ErrNoRows {
				// No more rows with NULL vectors, exit gracefully
				return nil
			}
			return fmt.Errorf("error fetching next row: %w", err)
		}

		log.Printf("Processing embeddings for %s", hash)
		embeddings, err := aiEmbeddings(text)
		if err != nil {
			return fmt.Errorf("embeddings generation error: %w", err)
		}

		blob, err := Float32ToBlob(embeddings)
		if err != nil {
			return fmt.Errorf("error converting embeddings to blob: %w", err)
		}

		tx, err := h.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		defer tx.Rollback() // Rollback if we don't commit

		updateQuery := `UPDATE hashes SET vectors = ? WHERE hash = ?;`
		_, err = tx.Exec(updateQuery, blob, hash)
		if err != nil {
			return fmt.Errorf("embeddings update error: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

	}
}
