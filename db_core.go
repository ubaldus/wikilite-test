// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type DBHandler struct {
	db *sql.DB
}

func (h *DBHandler) initializeDB() error {
	if err := h.PragmaInitMode(); err != nil {
		return err
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS setup (
			key TEXT PRIMARY KEY,
			value TEXT DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			entity TEXT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER,
			title TEXT,
			pow INTEGER DEFAULT 0,
			FOREIGN KEY(article_id) REFERENCES articles(id)
		)`,

		`CREATE TABLE IF NOT EXISTS content (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id INTEGER,
			hash_id INTEGER NOT NULL,
			FOREIGN KEY(section_id) REFERENCES sections(id),
			FOREIGN KEY(hash_id) REFERENCES hashes(id)
		)`,

		`CREATE TABLE IF NOT EXISTS hashes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT UNIQUE NOT NULL,
			text TEXT NOT NULL,
			pow INTEGER DEFAULT 0
		)`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS article_search USING fts5(
			title,
			content='articles',
			content_rowid='id'
		)`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS hash_search USING fts5(
			text,
			content='hashes',
			content_rowid='id'
		)`,

		`CREATE TABLE IF NOT EXISTS vectors (
			id INTEGER PRIMARY KEY,
			embedding BLOB
		)`,

		`CREATE TABLE IF NOT EXISTS vectors_ann_chunks (
			id INTEGER PRIMARY KEY,
			chunk BLOB
		)`,

		`CREATE TABLE IF NOT EXISTS vectors_ann_index (
			id INTEGER PRIMARY KEY,
			vectors_id INTEGER NOT NULL,
			chunk_id INTEGER NOT NULL,
			chunk_position INTEGER NOT NULL
		)`,

		`CREATE INDEX IF NOT EXISTS idx_vectors_ann_index_chunk_id_position ON vectors_ann_index (chunk_id, chunk_position)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_article_id ON sections(article_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_section_id ON content(section_id)`,
		`CREATE INDEX IF NOT EXISTS idx_hashes_id ON hashes(id)`,
	}
	for _, query := range queries {
		if _, err := h.db.Exec(query); err != nil {
			return fmt.Errorf("error executing query %s: %v", query, err)
		}
	}

	if err := h.PragmaReadMode(); err != nil {
		return err
	}

	return nil
}

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

	if language, err := handler.SetupGet("language"); err == nil && language != "" {
		options.language = language
	}

	if model, err := handler.SetupGet("model"); err == nil && model != "" {
		options.aiModel = model
	}

	return handler, nil
}

func (h *DBHandler) Close() error {
	return h.db.Close()
}

func (h *DBHandler) Pragma(pragmas []string) error {
	for _, pragma := range pragmas {
		if _, err := h.db.Exec(pragma); err != nil {
			return fmt.Errorf("error executing PRAGMA %s: %v", pragma, err)
		}
	}
	return nil
}

func (h *DBHandler) PragmaInitMode() error {
	pragmas := []string{
		"PRAGMA synchronous = OFF",
		"PRAGMA journal_mode = OFF",
		"PRAGMA foreign_keys = OFF",
		"PRAGMA cache_size = -10000",
		"PRAGMA mmap_size = 268435456",
		"PRAGMA temp_store = MEMORY",
	}
	return h.Pragma(pragmas)
}

func (h *DBHandler) PragmaReadMode() error {
	pragmas := []string{
		"PRAGMA locking_mode = NORMAL",
		"PRAGMA query_only = ON",
	}
	return h.Pragma(pragmas)
}

func (h *DBHandler) PragmaImportMode() error {
	pragmas := []string{
		"PRAGMA locking_mode = EXCLUSIVE",
		"PRAGMA query_only = OFF",
	}
	return h.Pragma(pragmas)
}

func (h *DBHandler) Optimize() error {
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	log.Println("Deleting duplicate sections")
	_, err = tx.Exec(`
		DELETE FROM sections
		WHERE id NOT IN (
			SELECT MAX(id)
			FROM sections
			GROUP BY article_id, title
		)`)
	if err != nil {
		return fmt.Errorf("error deleting duplicate sections: %v", err)
	}

	log.Println("Updating hashes for orphaned content")
	_, err = tx.Exec(`
		UPDATE hashes
		SET pow = pow - 1
		WHERE id IN (
			SELECT hash_id
			FROM content
			WHERE section_id NOT IN (SELECT id FROM sections)
		)`)
	if err != nil {
		return fmt.Errorf("error updating hashes for orphaned content: %v", err)
	}

	log.Println("Deleting orphaned content")
	_, err = tx.Exec("DELETE FROM content WHERE section_id NOT IN (SELECT id FROM sections)")
	if err != nil {
		return fmt.Errorf("error deleting orphaned content: %v", err)
	}

	log.Println("Deleting unused hashes")
	_, err = tx.Exec("DELETE FROM hashes WHERE pow <= 0")
	if err != nil {
		return fmt.Errorf("error deleting hashes with pow <= 0: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	log.Println("Running VACUUM")
	_, err = h.db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("error executing VACUUM: %v", err)
	}

	return nil
}

func (h *DBHandler) SetupPut(key, value string) (err error) {
	_, err = h.db.Exec("INSERT OR REPLACE INTO setup (key, value) VALUES (?, ?)", key, value)
	return
}

func (h *DBHandler) SetupGet(key string) (value string, err error) {
	err = h.db.QueryRow("SELECT value FROM setup WHERE key = ? LIMIT 1", key).Scan(&value)
	return
}

func (h *DBHandler) ArticlePut(article OutputArticle) error {
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT OR REPLACE INTO articles (id, title, entity) VALUES (?, ?, ?)",
		article.ID, article.Title, article.Entity,
	)
	if err != nil {
		return fmt.Errorf("error inserting article: %v", err)
	}

	for _, item := range article.Items {
		title, _ := item["title"].(string)
		pow, _ := item["pow"].(int)

		result, err := tx.Exec(
			"INSERT INTO sections (article_id, title, pow) VALUES (?, ?, ?)",
			article.ID, title, pow,
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

			result, err = tx.Exec(
				"INSERT OR IGNORE INTO hashes (hash, text) VALUES (?, ?)",
				hash, text,
			)
			if err != nil {
				return fmt.Errorf("error inserting hash: %v", err)
			}
			var hashID int
			err = tx.QueryRow("SELECT id FROM hashes WHERE hash = ?", hash).Scan(&hashID)
			if err != nil {
				return fmt.Errorf("error getting hash ID: %v", err)
			}

			if err != nil {
				return fmt.Errorf("error inserting hash to fts: %v", err)
			}

			_, err = tx.Exec("UPDATE hashes SET pow = pow + 1 WHERE id = ?", hashID)
			if err != nil {
				return fmt.Errorf("error updating hash pow: %v", err)
			}

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

func (h *DBHandler) ArticleGet(articleID int) ([]ArticleResult, error) {
	sqlQuery := `
		SELECT 
			a.id AS article_id,
			a.title AS article_title,
			a.entity AS article_entity,
			s.title AS section_title,
			s.id AS section_id,
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
			s.id ASC, c.id ASC;
	`

	rows, err := h.db.Query(sqlQuery, articleID)
	if err != nil {
		return nil, fmt.Errorf("article query error: %v", err)
	}
	defer rows.Close()

	articleMap := make(map[int]*ArticleResult)

	for rows.Next() {
		var (
			articleID    int
			title        string
			entity       string
			sectionID    int
			sectionTitle string
			content      string
		)

		if err := rows.Scan(
			&articleID,
			&title,
			&entity,
			&sectionTitle,
			&sectionID,
			&content,
		); err != nil {
			return nil, fmt.Errorf("error scanning result: %v", err)
		}

		article, exists := articleMap[articleID]
		if !exists {
			article = &ArticleResult{
				Title:    title,
				Entity:   entity,
				ID:       articleID,
				Sections: []ArticleResultSection{},
			}
			articleMap[articleID] = article
		}

		var section *ArticleResultSection
		for i, sec := range article.Sections {
			if sec.Title == sectionTitle {
				section = &article.Sections[i]
				break
			}
		}

		if section == nil {
			article.Sections = append(article.Sections, ArticleResultSection{
				Title: sectionTitle,
				ID:    sectionID,
				Texts: []string{},
			})
			section = &article.Sections[len(article.Sections)-1]
		}

		section.Texts = append(section.Texts, content)
	}

	results := make([]ArticleResult, 0, len(articleMap))
	for _, article := range articleMap {
		results = append(results, *article)
	}

	return results, nil
}
