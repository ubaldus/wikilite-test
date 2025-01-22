// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"sort"
)

func (h *DBHandler) SearchTitle(searchQuery string, limit int) ([]SearchResult, error) {
	sqlQuery := `
		SELECT rowid, title, bm25(article_search) AS power
		FROM article_search
		WHERE article_search MATCH ?
		ORDER BY power ASC
		LIMIT ?
	`

	rows, err := h.db.Query(sqlQuery, searchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(
			&result.ArticleID,
			&result.Title,
			&result.Power,
		); err != nil {
			return nil, err
		}

		contentQuery := `
					SELECT text
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
					)
				`
		err = h.db.QueryRow(contentQuery, result.ArticleID, result.ArticleID).Scan(&result.Text)
		if err != nil {
			return nil, err
		}
		result.Type = "T"
		results = append(results, result)
	}

	return results, nil
}

func (h *DBHandler) SearchContent(searchQuery string, limit int) ([]SearchResult, error) {
	sqlQuery := `
		SELECT rowid, text, bm25(hash_search) as power
		FROM hash_search
		WHERE hash_search MATCH ?
		ORDER BY power
		LIMIT ?;
	`
	rows, err := h.db.Query(sqlQuery, searchQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var contentID int
		if err := rows.Scan(&contentID, &result.Text, &result.Power); err != nil {
			return nil, err
		}
		articleQuery := `
			SELECT id, title
			FROM articles
			WHERE articles.id = (
				SELECT article_id 
				FROM sections 
				WHERE sections.id = 
					(SELECT section_id FROM content WHERE content.id=?)
			)
		`
		err = h.db.QueryRow(articleQuery, contentID).Scan(&result.ArticleID, &result.Title)
		if err != nil {
			return nil, err
		}
		result.Type = "C"
		results = append(results, result)
	}

	return results, nil
}

func (h *DBHandler) SearchVectors(query string, limit int) ([]SearchResult, error) {
	annLimit := limit * 8
	queryEmbedding, err := aiEmbeddings(query)
	if err != nil {
		return nil, err
	}

	quantizedQuery := aiQuantizeBinary(queryEmbedding)

	rows, err := h.db.Query("SELECT id, chunk FROM vectors_ann_chunks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type VectorDistance struct {
		ID            int64
		ChunkRowID    int64
		ChunkPosition int
		Distance      float32
	}

	topANNResults := make([]VectorDistance, 0, annLimit)
	for rows.Next() {
		var chunkRowID int64
		var chunkBlob []byte
		chunkSize := len(quantizedQuery)
		if err := rows.Scan(&chunkRowID, &chunkBlob); err != nil {
			return nil, err
		}
		for position := 0; position < len(chunkBlob); position += chunkSize {
			var result VectorDistance
			embeddingBlob := chunkBlob[position:(position + chunkSize)]
			distance, err := aiHammingDistance(quantizedQuery, embeddingBlob)
			if err != nil {
				return nil, err
			}
			result.ChunkRowID = chunkRowID
			result.ChunkPosition = position / chunkSize
			result.Distance = distance

			if len(topANNResults) < annLimit {
				topANNResults = append(topANNResults, result)
			} else {
				if distance < topANNResults[annLimit-1].Distance {
					topANNResults[annLimit-1] = result
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
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		embedding := aiBytesToFloat32(embeddingBlob)
		distance, err := aiL2Distance(queryEmbedding, embedding)
		if err != nil {
			return nil, err
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
			&result.Text,
		)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		result.Type = "V"
		result.Power = float64(vd.Distance)
		results = append(results, result)
	}

	return results, nil
}
