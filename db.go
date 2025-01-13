// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

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

		`CREATE TABLE IF NOT EXISTS vectors_ann (
			id INTEGER PRIMARY KEY,
			embedding BLOB
		)`,
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

func (h *DBHandler) GetArticle(articleID int) ([]ArticleResult, error) {
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
			&result.ArticleID,
			&result.Title,
			&result.Entity,
			&result.SectionTitle,
			&result.Text,
			&result.Power,
		); err != nil {
			return nil, fmt.Errorf("error scanning result: %v", err)
		}

		textQuery := `
			SELECT text, (
				SELECT id 
				FROM sections 
				WHERE article_id = ? 
				LIMIT 1
			) AS section_title
			FROM hashes
			WHERE id = (
				SELECT hash_id 
				FROM content 
				WHERE section_id = (
					SELECT id 
					FROM sections 
					WHERE article_id = ? 
					LIMIT 1
				) 
				LIMIT 1
			)
			LIMIT 1;
		`
		err = h.db.QueryRow(textQuery, result.ArticleID, result.ArticleID).Scan(&result.Text, &result.SectionID)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("error fetching text for article %d: %v", result.ArticleID, err)
		}
		result.Type = "T"
		results = append(results, result)
	}

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
			s.title as section_title,
			s.id as section_id,
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
			&result.ArticleID,
			&result.Title,
			&result.Entity,
			&result.SectionTitle,
			&result.SectionID,
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

func (h *DBHandler) SearchVectors(query string, limit int) ([]SearchResult, error) {
	queryEmbedding, err := aiEmbeddings(query)
	if err != nil {
		return nil, fmt.Errorf("embeddings generation error: %w", err)
	}

	quantizedQuery := aiQuantizeBinary(queryEmbedding)

	rows, err := h.db.Query("SELECT id, embedding FROM vectors_ann")
	if err != nil {
		return nil, fmt.Errorf("error querying vectors_ann: %w", err)
	}
	defer rows.Close()

	type VectorDistance struct {
		ID       int64
		Distance float32
	}

	topANNResults := make([]VectorDistance, 0, limit*8)
	for rows.Next() {
		var id int64
		var embeddingBlob []byte
		if err := rows.Scan(&id, &embeddingBlob); err != nil {
			return nil, fmt.Errorf("error scanning vector_ann row: %w", err)
		}

		distance, err := aiHammingDistance(quantizedQuery, embeddingBlob)
		if err != nil {
			return nil, fmt.Errorf("error calculating Hamming distance: %w", err)
		}

		if len(topANNResults) < limit*8 {
			topANNResults = append(topANNResults, VectorDistance{ID: id, Distance: distance})
		} else {
			if distance < topANNResults[limit*8-1].Distance {
				topANNResults[limit*8-1] = VectorDistance{ID: id, Distance: distance}
			}
		}
		sort.Slice(topANNResults, func(i, j int) bool {
			return topANNResults[i].Distance < topANNResults[j].Distance
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vector_ann rows: %w", err)
	}

	topResults := make([]VectorDistance, 0, limit)
	for _, annResult := range topANNResults {
		var embeddingBlob []byte
		err := h.db.QueryRow("SELECT embedding FROM vectors WHERE id = ?", annResult.ID).Scan(&embeddingBlob)
		if err != nil {
			if err == sql.ErrNoRows {
				continue // Skip if no matching vector is found
			}
			return nil, fmt.Errorf("error fetching vector embedding: %w", err)
		}

		embedding := aiBytesToFloat32(embeddingBlob)
		distance, err := aiL2Distance(queryEmbedding, embedding)
		if err != nil {
			return nil, fmt.Errorf("error calculating L2 distance: %w", err)
		}

		if len(topResults) < limit {
			topResults = append(topResults, VectorDistance{ID: annResult.ID, Distance: float32(distance)})
		} else {
			if distance < topResults[len(topResults)-1].Distance {
				topResults[len(topResults)-1] = VectorDistance{ID: annResult.ID, Distance: float32(distance)}
			}
		}
	}

	sort.Slice(topResults, func(i, j int) bool {
		return topResults[i].Distance < topResults[j].Distance
	})

	var results []SearchResult
	for _, vd := range topResults {
		sqlQuery := `
			SELECT
				a.id as article_id,
				a.title,
				a.entity,
				s.title as section_title,
				s.id as section_id,
				h.text
			FROM hashes h
			JOIN content c ON c.hash_id = h.id
			JOIN sections s ON s.id = c.section_id
			JOIN articles a ON a.id = s.article_id
			WHERE h.id = ?
		`

		var result SearchResult
		err := h.db.QueryRow(sqlQuery, vd.ID).Scan(
			&result.ArticleID,
			&result.Title,
			&result.Entity,
			&result.SectionTitle,
			&result.SectionID,
			&result.Text,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, fmt.Errorf("error fetching article info: %w", err)
		}

		result.Type = "V"
		result.Power = float64(vd.Distance)
		results = append(results, result)
	}

	return results, nil
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

func (h *DBHandler) ProcessTitles() error {
	_, err := h.db.Exec("INSERT INTO article_search(rowid, title) SELECT id, title FROM articles")
	if err != nil {
		return fmt.Errorf("error populating article_search table: %v", err)
	}

	return nil
}

func (h *DBHandler) ProcessContents() error {
	_, err := h.db.Exec("INSERT INTO hash_search(rowid, text) SELECT id, text FROM hashes")
	if err != nil {
		return fmt.Errorf("error populating hash_search table: %v", err)
	}

	return nil
}

func (h *DBHandler) ProcessEmbeddings() (err error) {
	batchSize := 1000
	totalCount := 0
	offset := 0

	if err = db.SetupPut("model", options.aiModel); err != nil {
		return
	}

	err = h.db.QueryRow("SELECT COUNT(*) FROM hashes WHERE id NOT IN (select id from vectors_ann)").Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("error getting total count of hashes: %w", err)
	}

	log.Printf("Pending embeddings: %d", totalCount)

	startTime := time.Now()
	var problematicIDs []int
	for {
		query := `SELECT id, hash, text FROM hashes WHERE id NOT IN (select id from vectors_ann)`
		if len(problematicIDs) > 0 {
			idList := make([]string, 0, len(problematicIDs))
			for _, id := range problematicIDs {
				idList = append(idList, fmt.Sprintf("%d", id))
			}
			query += " AND id NOT IN (" + strings.Join(idList, ", ") + ")"
		}
		query += " LIMIT ?"

		rows, err := h.db.Query(query, batchSize)
		if err != nil {
			return fmt.Errorf("error querying hashes: %w", err)
		}
		defer rows.Close()

		type HashData struct {
			ID   int
			Hash string
			Text string
		}
		var hashesData []HashData
		for rows.Next() {
			var data HashData
			if err := rows.Scan(&data.ID, &data.Hash, &data.Text); err != nil {
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

		for _, hashData := range hashesData {
			embedding, err := aiEmbeddings(hashData.Text)
			if err != nil {
				log.Printf("Embedding generation error for hash %s: %v", hashData.Hash, err)
				problematicIDs = append(problematicIDs, hashData.ID)
				continue
			}

			if _, err := h.db.Exec("INSERT OR REPLACE INTO vectors (rowid, embedding) VALUES (?, ?)", hashData.ID, aiFloat32ToBytes(embedding)); err != nil {
				log.Printf("Error inserting vectors for hash %s: %v", hashData.Hash, err)
				problematicIDs = append(problematicIDs, hashData.ID)
				continue
			}
			if _, err := h.db.Exec("INSERT OR REPLACE INTO vectors_ann (rowid, embedding) VALUES (?, ?)", hashData.ID, aiQuantizeBinary(embedding)); err != nil {
				log.Printf("Error inserting vectors_ann for hash %s: %v", hashData.Hash, err)
				problematicIDs = append(problematicIDs, hashData.ID)
				continue
			}
		}
		processedCount := offset + len(hashesData)
		progress := float64(processedCount) / float64(totalCount) * 100
		elapsed := time.Since(startTime)
		estimatedTotalTime := time.Duration(float64(elapsed) / (progress / 100.0))
		remainingTime := estimatedTotalTime - elapsed

		log.Printf("Embedding progress: %.2f%%, Estimated total time: %s, Remaining: %s", progress, estimatedTotalTime.Truncate(time.Second), remainingTime.Truncate(time.Second))

		offset += batchSize
	}

	return nil
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
