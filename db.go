// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

type DBHandler struct {
	db *sql.DB
}

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
			pow INTEGER DEFAULT 0,
			FOREIGN KEY(article_id) REFERENCES articles(id)
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
			hash,
			text,
			content='hashes',
			content_rowid='id'
		)`,

		`CREATE TABLE IF NOT EXISTS content (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id INTEGER,
			hash_id INTEGER NOT NULL,
			FOREIGN KEY(section_id) REFERENCES sections(id),
			FOREIGN KEY(hash_id) REFERENCES hashes(id)
		)`,
	}

	for _, query := range queries {
		if _, err := h.db.Exec(query); err != nil {
			return fmt.Errorf("error executing query %s: %v", query, err)
		}
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

	return handler, nil
}

func (h *DBHandler) Close() error {
	return h.db.Close()
}

func (h *DBHandler) SaveArticle(article OutputArticle) error {
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

	_, err = tx.Exec("INSERT INTO article_search(rowid, title) VALUES (?, ?)", article.ID, article.Title)
	if err != nil {
		return fmt.Errorf("error inserting into article_search: %v", err)
	}

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
			_, err = tx.Exec(
				"INSERT INTO hash_search(rowid, hash, text) VALUES (?, ?, ?)",
				hashID, hash, text,
			)

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

func (h *DBHandler) GetArticle(articleID int) ([]ArticleResult, error) {
	sqlQuery := `
		SELECT 
			a.id AS article_id,
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
			&result.Article,
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

func (h *DBHandler) SearchTitle(searchQuery string, limit int) ([]SearchResult, error) {
	sqlQuery := `
		WITH matched_titles AS (
  		SELECT rowid, title, bm25(article_search) AS relevance
			FROM article_search
			WHERE article_search MATCH ?
		)
    SELECT DISTINCT
			a.id as article_id,
			a.title,
			a.entity,
			'' as section,
			'' as text,
			relevance
		FROM matched_titles mt
		JOIN articles a ON a.id = mt.rowid
    ORDER BY mt.relevance ASC
    LIMIT ?
	`

	rows, err := h.db.Query(sqlQuery, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search error: %v", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(
			&result.Article,
			&result.Title,
			&result.Entity,
			&result.Section,
			&result.Text,
			&result.Power,
		); err != nil {
			return nil, fmt.Errorf("error scanning result: %v", err)
		}

		var text string
		textQuery := `
			SELECT h.text AS content
			FROM articles a
			JOIN sections s ON a.id = s.article_id
			JOIN content c ON s.id = c.section_id
			JOIN hashes h ON c.hash_id = h.id
			WHERE a.id = ?
			LIMIT 1
		`
		err = h.db.QueryRow(textQuery, result.Article).Scan(&text)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("error fetching text for article %d: %v", result.Article, err)
		}
		result.Text = text
		result.Type = "T"
		results = append(results, result)
	}

	return results, nil
}

func (h *DBHandler) SearchHash(hashes []string, scores []float64, limit int) ([]SearchResult, error) {
	if len(hashes) != len(scores) {
		return nil, fmt.Errorf("hashes and scores arrays must have the same length")
	}

	hashScoreMap := make(map[string]float64, len(hashes))
	for i, hash := range hashes {
		hashScoreMap[hash] = scores[i]
	}

	hashString := ""
	for i, hash := range hashes {
		if i > 0 {
			hashString += ", "
		}
		hashString += "'" + hash + "'"
	}

	sqlQuery := fmt.Sprintf(`
    SELECT DISTINCT
      a.id as article_id,
      a.title,
      a.entity,
      s.sub as section,
      h.text,
      h.hash
    FROM hashes h
    JOIN content c ON c.hash_id = h.id
    JOIN sections s ON s.id = c.section_id
    JOIN articles a ON a.id = s.article_id
    WHERE h.hash IN (%s)
	`, hashString)

	rows, err := h.db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("error executing hash search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	seenArticleIds := make(map[int]bool)
	for rows.Next() {
		if limit > 0 && len(results) >= limit {
			break
		}
		var result SearchResult
		var hash string
		var articleId int
		if err := rows.Scan(
			&articleId,
			&result.Title,
			&result.Entity,
			&result.Section,
			&result.Text,
			&hash,
		); err != nil {
			return nil, fmt.Errorf("error scanning result: %v", err)
		}

		if _, seen := seenArticleIds[articleId]; seen {
			continue
		}

		seenArticleIds[articleId] = true
		result.Article = articleId
		result.Type = "V"
		result.Power = hashScoreMap[hash]
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Power > results[j].Power
	})

	return results, nil
}

func (h *DBHandler) SearchContent(searchQuery string, limit int) ([]SearchResult, error) {
	sqlQuery := `
  	SELECT rowid, text, bm25(hash_search) as relevance
		FROM hash_search
		WHERE hash_search MATCH ?
		ORDER BY relevance ASC
		LIMIT ?;
	`

	rows, err := h.db.Query(sqlQuery, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("hash search error: %v", err)
	}
	defer rows.Close()

	type HashResult struct {
		RowID     int
		Text      string
		Relevance float64
	}
	hashResults := make([]HashResult, 0)
	for rows.Next() {
		var result HashResult
		if err := rows.Scan(&result.RowID, &result.Text, &result.Relevance); err != nil {
			return nil, fmt.Errorf("error scanning hash result: %v", err)
		}
		hashResults = append(hashResults, result)
	}
	if len(hashResults) == 0 {
		return []SearchResult{}, nil
	}
	placeholders := ""
	params := make([]interface{}, len(hashResults))

	for i, hash := range hashResults {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		params[i] = hash.RowID
	}
	sqlQuery = fmt.Sprintf(`
		SELECT
			a.id as article_id,
			a.title,
			a.entity,
			s.sub as section,
			h.id as hash_id
		FROM hashes h
		JOIN content c ON c.hash_id = h.id
		JOIN sections s ON s.id = c.section_id
		JOIN articles a ON a.id = s.article_id
		WHERE h.id IN (%s)
  	`, placeholders)

	rows, err = h.db.Query(sqlQuery, params...)
	if err != nil {
		return nil, fmt.Errorf("error retrieving article info: %v", err)
	}
	defer rows.Close()

	resultsMap := make(map[int]SearchResult, len(hashResults))
	for rows.Next() {
		var result SearchResult
		var hash_id int
		if err := rows.Scan(
			&result.Article,
			&result.Title,
			&result.Entity,
			&result.Section,
			&hash_id,
		); err != nil {
			return nil, fmt.Errorf("error scanning article info: %v", err)
		}
		result.Type = "C"
		resultsMap[hash_id] = result
	}

	results := make([]SearchResult, 0, len(hashResults))
	for _, hashResult := range hashResults {
		if result, ok := resultsMap[hashResult.RowID]; ok {
			result.Text = hashResult.Text
			result.Power = hashResult.Relevance
			results = append(results, result)
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (h *DBHandler) ProcessEmbeddings() error {
	batchSize := 1000
	offset := 0
	totalCount := 0

	err := h.db.QueryRow("SELECT COUNT(*) FROM hashes").Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("error getting total count of hashes: %w", err)
	}

	for {
		rows, err := h.db.Query(`SELECT hash, text FROM hashes LIMIT ? OFFSET ?`, batchSize, offset)
		if err != nil {
			return fmt.Errorf("error querying hashes: %w", err)
		}
		defer rows.Close()

		type HashData struct {
			Hash string
			Text string
		}
		var hashesData []HashData
		for rows.Next() {
			var data HashData
			if err := rows.Scan(&data.Hash, &data.Text); err != nil {
				return fmt.Errorf("error scanning row: %w", err)
			}
			hashesData = append(hashesData, data)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating rows: %w", err)
		}

		if len(hashesData) == 0 {
			// No more hashes to process
			break
		}

		hashEmbeddings := make(map[string][]float32)
		var hashesToUpdate []string

		for _, hashData := range hashesData {
			hash := hashData.Hash
			text := hashData.Text

			exists, err := qdrantHashExists(qd.PointsClient, options.qdrantCollection, hash)
			if err != nil {
				log.Printf("Error checking qdrant existence for hash %s: %v", hash, err)
				continue
			}
			if exists {
				log.Printf("Hash %s already exists in qdrant", hash)
				continue
			}

			embedding, err := aiEmbeddings(text)
			if err != nil {
				log.Printf("Embedding generation error for hash %s: %v", hash, err)
				continue
			}
			hashEmbeddings[hash] = embedding
			hashesToUpdate = append(hashesToUpdate, hash)
		}

		if len(hashEmbeddings) > 0 {
			err = qdrantUpsertPoints(qd.PointsClient, options.qdrantCollection, hashEmbeddings)
			if err != nil {
				log.Printf("Error upserting batch to qdrant: %v", err)
			}
		}

		processedCount := offset + len(hashesData)
		progress := float64(processedCount) / float64(totalCount) * 100
		log.Printf("Embedding progress: %.2f%%", progress)

		offset += batchSize
	}

	return nil
}
