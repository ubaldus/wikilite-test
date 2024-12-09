package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
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
	db, err := sql.Open("sqlite", dbPath)
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

// initializeDB creates the necessary tables and indexes
func (h *DBHandler) initializeDB() error {
	queries := []string{
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
			hash TEXT NOT NULL,
			text TEXT NOT NULL,
			FOREIGN KEY(section_id) REFERENCES sections(id)
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS content_fts USING fts5(
			text,
			content='content',
			content_rowid='id'
		)`,
		`CREATE TRIGGER IF NOT EXISTS content_ai AFTER INSERT ON content BEGIN
			INSERT INTO content_fts(rowid, text) VALUES (new.id, new.text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS content_ad AFTER DELETE ON content BEGIN
			INSERT INTO content_fts(content_fts, rowid, text) VALUES('delete', old.id, old.text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS content_au AFTER UPDATE ON content BEGIN
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

			_, err = tx.Exec(
				"INSERT INTO content (section_id, hash, text) VALUES (?, ?, ?)",
				sectionID, hash, text,
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
