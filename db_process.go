// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

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

	err = h.db.QueryRow("SELECT COUNT(*) FROM hashes WHERE id NOT IN (select vectors_id from vectors_ann_index)").Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("error getting total count of hashes: %w", err)
	}

	log.Printf("Pending embeddings: %d", totalCount)

	startTime := time.Now()
	var problematicIDs []int
	for {
		tx, err := h.db.Begin()
		if err != nil {
			return fmt.Errorf("error starting transaction: %w", err)
		}

		query := `SELECT id, hash, text FROM hashes WHERE id NOT IN (select vectors_id from vectors_ann_index)`
		if len(problematicIDs) > 0 {
			idList := make([]string, 0, len(problematicIDs))
			for _, id := range problematicIDs {
				idList = append(idList, fmt.Sprintf("%d", id))
			}
			query += " AND id NOT IN (" + strings.Join(idList, ", ") + ")"
		}
		query += " LIMIT ?"

		rows, err := tx.Query(query, batchSize)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error querying hashes: %w", err)
		}

		type HashData struct {
			ID   int
			Hash string
			Text string
		}
		var hashesData []HashData
		for rows.Next() {
			var data HashData
			if err := rows.Scan(&data.ID, &data.Hash, &data.Text); err != nil {
				rows.Close()
				tx.Rollback()
				return fmt.Errorf("error scanning row: %w", err)
			}
			hashesData = append(hashesData, data)
		}
		rows.Close()

		if err := rows.Err(); err != nil {
			tx.Rollback()
			return fmt.Errorf("error iterating rows: %w", err)
		}

		if len(hashesData) == 0 {
			tx.Commit()
			break
		}

		var ann_chunk_data []byte
		var ann_chunk_position int
		var ann_chunk_rowid int
		if err := tx.QueryRow("SELECT id FROM vectors_ann_chunks ORDER BY id DESC LIMIT 1").Scan(&ann_chunk_rowid); err != nil && err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		ann_chunk_rowid++

		for _, hashData := range hashesData {
			embedding, err := aiEmbeddings(hashData.Text)
			if err != nil {
				log.Printf("Embedding generation error for hash %s: %v", hashData.Hash, err)
				problematicIDs = append(problematicIDs, hashData.ID)
				continue
			}

			if _, err := tx.Exec("INSERT OR REPLACE INTO vectors (rowid, embedding) VALUES (?, ?)", hashData.ID, aiFloat32ToBytes(embedding)); err != nil {
				log.Printf("Error inserting vectors for hash %s: %v", hashData.Hash, err)
				problematicIDs = append(problematicIDs, hashData.ID)
				continue
			}
			if _, err := tx.Exec("INSERT INTO vectors_ann_index (vectors_id, chunk_id, chunk_position) VALUES (?, ?, ?)", hashData.ID, ann_chunk_rowid, ann_chunk_position); err != nil {
				log.Printf("Error inserting vectors_ann for hash %s: %v", hashData.Hash, err)
				problematicIDs = append(problematicIDs, hashData.ID)
				continue
			}
			ann_chunk_data = append(ann_chunk_data, aiQuantizeBinary(embedding)...)
			ann_chunk_position++
		}
		if _, err := tx.Exec("INSERT INTO vectors_ann_chunks (id, chunk) VALUES (?, ?)", ann_chunk_rowid, ann_chunk_data); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
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
