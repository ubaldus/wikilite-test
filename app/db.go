// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

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
		`CREATE VIRTUAL TABLE IF NOT EXISTS article_search USING fts5(
			title,
			content='articles',
			content_rowid='id'
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS article_search_vocabulary USING fts5vocab(article_search, row)`,

		`CREATE TABLE IF NOT EXISTS sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			article_id INTEGER,
			title TEXT,
			content TEXT,
			pow INTEGER DEFAULT 0,
			FOREIGN KEY(article_id) REFERENCES articles(id)
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS section_search USING fts5(
			title, content,
			content='sections',
			content_rowid='id'
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS section_search_vocabulary USING fts5vocab(section_search, row)`,

		`CREATE TABLE IF NOT EXISTS vocabulary (term TEXT)`,

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

	if options.aiModel == "" {
		if model, err := handler.SetupGet("model"); err == nil && model != "" {
			options.aiModel = model
		}
	}

	if annMode, err := handler.SetupGet("annMode"); err == nil && annMode != "" {
		options.aiAnnMode = annMode
	}

	if annSize, err := handler.SetupGet("annSize"); err == nil && annSize != "" {
		options.aiAnnSize = extractNumberFromString(annSize)
	}

	if modelPrefixSearch, err := handler.SetupGet("modelPrefixSearch"); err == nil && modelPrefixSearch != "" {
		options.aiModelPrefixSearch = modelPrefixSearch
	}

	if modelPrefixSave, err := handler.SetupGet("modelPrefixSave"); err == nil && modelPrefixSave != "" {
		options.aiModelPrefixSave = modelPrefixSave
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

func (h *DBHandler) HasAI() bool {
	var id int
	err := h.db.QueryRow("SELECT id FROM vectors LIMIT 1").Scan(&id)
	return err != sql.ErrNoRows
}

func (h *DBHandler) HasANN() bool {
	var id int
	err := h.db.QueryRow("SELECT id FROM vectors_ann_index LIMIT 1").Scan(&id)
	return err != sql.ErrNoRows
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
		content, _ := item["content"].(string)

		_, err := tx.Exec(
			"INSERT INTO sections (article_id, title, content, pow) VALUES (?, ?, ?, ?)",
			article.ID, title, content, pow,
		)
		if err != nil {
			return fmt.Errorf("error inserting section: %v", err)
		}
	}

	return tx.Commit()
}

func (h *DBHandler) ArticleGet(articleID int) (ArticleResult, error) {
	article := ArticleResult{
		Sections: []ArticleResultSection{},
	}

	sqlQuery := `
		SELECT
			a.id,
			a.title,
			a.entity,
			s.id,
			s.title,
			s.content
		FROM
			articles a
		JOIN
			sections s ON a.id = s.article_id
		WHERE
			a.id = ?
		ORDER BY
			s.id ASC;
	`

	rows, err := h.db.Query(sqlQuery, articleID)
	if err != nil {
		return article, fmt.Errorf("article query error: %v", err)
	}
	defer rows.Close()

	var isFirstRow = true
	for rows.Next() {
		var (
			artID     int
			artTitle  string
			artEntity string
			section   ArticleResultSection
		)

		if err := rows.Scan(
			&artID,
			&artTitle,
			&artEntity,
			&section.ID,
			&section.Title,
			&section.Content,
		); err != nil {
			return article, fmt.Errorf("error scanning result: %v", err)
		}

		if isFirstRow {
			article.ID = artID
			article.Title = artTitle
			article.Entity = artEntity
			isFirstRow = false
		}

		article.Sections = append(article.Sections, section)
	}

	if article.ID == 0 {
		return article, fmt.Errorf("article not found")
	}

	return article, nil
}
