// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"
)

func (h *DBHandler) SearchTitle(searchQuery string, limit int) ([]SearchResult, error) {
	start := time.Now()
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

		contentQuery := `SELECT content FROM sections WHERE article_id = ? LIMIT 1`
		err = h.db.QueryRow(contentQuery, result.ArticleID).Scan(&result.Text)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		result.Type = "T"
		results = append(results, result)
	}

	log.Printf("Search title: %s (%v)", searchQuery, time.Since(start))

	return results, nil
}

func (h *DBHandler) SearchContent(searchQuery string, limit int) ([]SearchResult, error) {
	start := time.Now()
	sqlQuery := `
		SELECT
			s.article_id,
			a.title,
			s.content,
			bm25(section_search) as power
		FROM section_search
		JOIN sections s ON section_search.rowid = s.id
		JOIN articles a ON s.article_id = a.id
		WHERE section_search MATCH ?
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
		if err := rows.Scan(&result.ArticleID, &result.Title, &result.Text, &result.Power); err != nil {
			return nil, err
		}
		result.Type = "C"
		results = append(results, result)
	}

	log.Printf("Search content: %s (%v)", searchQuery, time.Since(start))

	return results, nil
}

func (h *DBHandler) SearchWordDistance(inputWord string, limit int) ([]SearchResult, error) {
	start := time.Now()
	var allMatches []SearchResult
	seen := make(map[string]bool)

	batchSize := 100 * 1000
	offset := 0

	for {
		rows, err := h.db.Query("SELECT term FROM vocabulary LIMIT ? OFFSET ?", batchSize, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		processed := 0
		for rows.Next() {
			var word string
			if err := rows.Scan(&word); err != nil {
				return nil, err
			}

			if seen[word] {
				continue
			}
			seen[word] = true

			distance := LevenshteinDistance(inputWord, word)

			allMatches = append(allMatches, SearchResult{Text: word, Power: float64(distance)})

			if len(allMatches) > limit {
				sort.Slice(allMatches, func(i, j int) bool {
					return allMatches[i].Power < allMatches[j].Power
				})
				allMatches = allMatches[:limit]
			}

			processed++
		}

		if processed < batchSize {
			break
		}
		offset += batchSize
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Power < allMatches[j].Power
	})

	log.Printf("Search word distance: %s (%v)", inputWord, time.Since(start))

	return allMatches, nil
}

func (h *DBHandler) SearchVectors(query string, limit int) ([]SearchResult, error) {
	start := time.Now()
	queryEmbedding, err := aiEmbeddings(options.aiModelPrefixSearch + query)
	if err != nil {
		return nil, err
	}

	sqlQuery := "SELECT id, embedding FROM vectors"
	type VectorDistance struct {
		ID            int64
		ChunkRowID    int64
		ChunkPosition int
		Distance      float32
	}

	if !options.aiAnnOff && db.HasANN() {
		chunkSize := 0
		annLimit := limit * limit
		if options.aiAnnMode == "mrl" {
			chunkSize = options.aiAnnSize * 4
		} else if options.aiAnnMode == "binary" {
			chunkSize = (len(queryEmbedding) + 7) / 8
		} else {
			return nil, fmt.Errorf("invalid ANN mode")
		}

		rows, err := h.db.Query("SELECT id, chunk FROM vectors_ann_chunks")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		topANNResults := make([]VectorDistance, 0, annLimit)

		for rows.Next() {
			var chunkRowID int64
			var chunkBlob []byte
			if err := rows.Scan(&chunkRowID, &chunkBlob); err != nil {
				return nil, err
			}

			for position := 0; position < len(chunkBlob); position += chunkSize {
				var result VectorDistance
				embeddingBlob := chunkBlob[position:(position + chunkSize)]

				var distance float32
				if options.aiAnnMode == "mrl" {
					mrlQuery := queryEmbedding
					if len(mrlQuery) > options.aiAnnSize {
						mrlQuery = mrlQuery[:options.aiAnnSize]
					}

					storedMRL := BytesToFloat32(embeddingBlob)
					distance, err = EuclideanDistance(mrlQuery, storedMRL)
				}
				if options.aiAnnMode == "binary" {
					quantizedQuery := QuantizeBinary(queryEmbedding)
					distance, err = HammingDistance(quantizedQuery, embeddingBlob)
				}

				result.ChunkRowID = chunkRowID
				result.ChunkPosition = position / chunkSize
				result.Distance = distance

				if len(topANNResults) <= annLimit {
					topANNResults = append(topANNResults, result)
				} else {
					maxIndex := -1
					maxDistance := float32(-1)
					for i := range topANNResults {
						if topANNResults[i].Distance > maxDistance {
							maxDistance = topANNResults[i].Distance
							maxIndex = i
						}
					}
					if maxIndex >= 0 && distance < maxDistance {
						topANNResults[maxIndex] = result
					}
				}
			}
		}

		sqlQuery += " WHERE id IN (0"
		for _, v := range topANNResults {
			var vectors_id int64
			if err := h.db.QueryRow("SELECT vectors_id FROM vectors_ann_index WHERE chunk_id = ? AND chunk_position = ? LIMIT 1", v.ChunkRowID, v.ChunkPosition).Scan(&vectors_id); err != nil {
				return nil, err
			}
			sqlQuery += "," + strconv.FormatInt(vectors_id, 10)
		}
		sqlQuery += ")"
	}

	topResults := make([]VectorDistance, 0, limit)
	rows, err := h.db.Query(sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var embeddingBlob []byte
		var ID int64
		if err := rows.Scan(&ID, &embeddingBlob); err != nil {
			return nil, err
		}
		embedding := BytesToFloat32(embeddingBlob)
		distance, err := EuclideanDistance(queryEmbedding, embedding)
		if err != nil {
			return nil, err
		}

		if len(topResults) <= limit {
			topResults = append(topResults, VectorDistance{ID: ID, Distance: float32(distance)})
		} else {
			maxIndex := -1
			maxDistance := float32(-1)
			for i := range topResults {
				if topResults[i].Distance > maxDistance {
					maxDistance = topResults[i].Distance
					maxIndex = i
				}
			}
			if maxIndex >= 0 && float32(distance) < maxDistance {
				topResults[maxIndex] = VectorDistance{ID: ID, Distance: float32(distance)}
			}
		}
	}

	var results []SearchResult
	for _, vd := range topResults {
		sqlQuery := `
			SELECT
				a.id,
				a.title,
				s.title
			FROM articles a
			JOIN sections s ON a.id = s.article_id
			WHERE s.id = ?
		`

		var result SearchResult
		var sectionTitle string
		err := h.db.QueryRow(sqlQuery, vd.ID).Scan(
			&result.ArticleID,
			&result.Title,
			&sectionTitle,
		)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		result.Text = sectionTitle
		result.Type = "V"
		result.Power = float64(vd.Distance)
		results = append(results, result)
	}

	log.Printf("Search vector: %s (%v)", query, time.Since(start))

	return results, nil
}
