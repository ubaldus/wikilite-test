// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"sort"
)

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

	rows, err := h.db.Query("SELECT id, chunk FROM vectors_ann_chunks")
	if err != nil {
		return nil, fmt.Errorf("error querying vectors_ann: %w", err)
	}
	defer rows.Close()

	type VectorDistance struct {
		ID            int64
		ChunkRowID    int64
		ChunkPosition int
		Distance      float32
	}

	topANNResults := make([]VectorDistance, 0, limit*8)
	for rows.Next() {
		var chunkRowID int64
		var chunkBlob []byte
		chunkSize := len(quantizedQuery)
		if err := rows.Scan(&chunkRowID, &chunkBlob); err != nil {
			return nil, fmt.Errorf("error scanning vector_ann_chunks row: %w", err)
		}
		for position := 0; position < len(chunkBlob); position += chunkSize {
			var result VectorDistance
			embeddingBlob := chunkBlob[position:(position + chunkSize)]
			distance, err := aiHammingDistance(quantizedQuery, embeddingBlob)
			if err != nil {
				return nil, fmt.Errorf("error calculating Hamming distance: %w", err)
			}
			result.ChunkRowID = chunkRowID
			result.ChunkPosition = position / chunkSize
			result.Distance = distance

			if len(topANNResults) < limit*8 {
				topANNResults = append(topANNResults, result)
			} else {
				if distance < topANNResults[limit*8-1].Distance {
					topANNResults[limit*8-1] = result
				}
			}
			sort.Slice(topANNResults, func(i, j int) bool {
				return topANNResults[i].Distance < topANNResults[j].Distance
			})
		}
	}

	for k, v := range topANNResults {
		var vectors_id int64
		if err := h.db.QueryRow("SELECT vectors_id FROM vectors_ann_index WHERE chunk_id = ? AND chunk_position = ? LIMIT 1", v.ChunkRowID, v.ChunkPosition).Scan(&vectors_id); err != nil {
			return nil, err
		}
		topANNResults[k].ID = vectors_id
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
