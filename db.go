package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
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

func (h *DBHandler) initializeDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS hashes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            hash TEXT UNIQUE NOT NULL,
            text TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS articles (
            id INTEGER PRIMARY KEY,
            title TEXT NOT NULL,
            entity TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS sections (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            article_id INTEGER,
            sub TEXT,
            pow INTEGER,
            FOREIGN KEY(article_id) REFERENCES articles(id)
        )`,
		`CREATE TABLE IF NOT EXISTS content (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            section_id INTEGER,
            hash_id INTEGER NOT NULL,
            FOREIGN KEY(section_id) REFERENCES sections(id),
            FOREIGN KEY(hash_id) REFERENCES hashes(id)
        )`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS content_fts USING fts5(
            text,
            content='hashes',
            content_rowid='id'
        )`,
		`CREATE TRIGGER IF NOT EXISTS hashes_ai AFTER INSERT ON hashes BEGIN
            INSERT INTO content_fts(rowid, text) VALUES (new.id, new.text);
        END`,
		`CREATE TRIGGER IF NOT EXISTS hashes_ad AFTER DELETE ON hashes BEGIN
            INSERT INTO content_fts(content_fts, rowid, text) VALUES('delete', old.id, old.text);
        END`,
		`CREATE TRIGGER IF NOT EXISTS hashes_au AFTER UPDATE ON hashes BEGIN
            INSERT INTO content_fts(content_fts, rowid, text) VALUES('delete', old.id, old.text);
            INSERT INTO content_fts(rowid, text) VALUES (new.id, new.text);
        END`,
	}

	for _, query := range queries {
		if _, err := h.db.Exec(query); err != nil {
			return fmt.Errorf("error executing query %s: %v", query, err)
		}
	}

	return nil
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
