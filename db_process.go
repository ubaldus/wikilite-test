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

	err = h.db.QueryRow("SELECT COUNT(*) FROM hashes WHERE id NOT IN (select id FROM vectors)").Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("error getting total count of hashes: %w", err)
	}

	log.Printf("Pending embeddings: %d", totalCount)

	startTime := time.Now()
	var problematicIDs []int
	for {
		query := `SELECT id, hash, text FROM hashes WHERE id NOT IN (select id FROM vectors)`
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
		}
		processedCount := offset + len(hashesData)
		progress := float64(processedCount) / float64(totalCount) * 100
		elapsed := time.Since(startTime)
		estimatedTotalTime := time.Duration(float64(elapsed) / (progress / 100.0))
		remainingTime := estimatedTotalTime - elapsed

		log.Printf("Embedding progress: %.2f%%, Estimated total time: %s, Remaining: %s", progress, estimatedTotalTime.Truncate(time.Second), remainingTime.Truncate(time.Second))

		offset += batchSize
	}

	if err = h.ProcessEmbeddingsANN(); err != nil {
		return fmt.Errorf("Embedding ANN generation error: %v\n", err)
	}

	return nil
}

func (h *DBHandler) ProcessEmbeddingsANN() (err error) {
	batchSize := 1000

	if _, err = h.db.Exec("DELETE FROM vectors_ann_chunks"); err != nil {
		return
	}

	if _, err = h.db.Exec("DELETE FROM vectors_ann_index"); err != nil {
		return
	}

	var totalVectors int
	err = h.db.QueryRow("SELECT COUNT(*) FROM vectors WHERE id NOT IN (SELECT vectors_id FROM vectors_ann_index)").Scan(&totalVectors)
	if err != nil {
		return err
	}

	ann_chunk_rowid := 1
	type VectorsIndex struct {
		VectorsID     int
		ChunkRowID    int
		ChunkPosition int
	}

	processedVectors := 0
	for {
		log.Printf("Embedding ANN chunk id: %d, Progress: %.2f%%\n", ann_chunk_rowid*batchSize, float64(processedVectors)/float64(totalVectors)*100)
		rows, err := h.db.Query("SELECT id, embedding FROM vectors WHERE id NOT IN (SELECT vectors_id FROM vectors_ann_index) LIMIT ?", batchSize)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		defer rows.Close()

		var ann_chunk_data []byte
		var ann_chunk_position int
		ann_chunk_index := make([]VectorsIndex, 0, batchSize)
		for rows.Next() {
			var id int
			var embedding []byte
			if err := rows.Scan(&id, &embedding); err != nil {
				return err
			}
			ann_chunk_index = append(ann_chunk_index, VectorsIndex{
				VectorsID:     id,
				ChunkRowID:    ann_chunk_rowid,
				ChunkPosition: ann_chunk_position,
			})
			ann_chunk_data = append(ann_chunk_data, aiQuantizeBinary(aiBytesToFloat32(embedding))...)
			ann_chunk_position++
		}
		if len(ann_chunk_index) == 0 {
			return nil
		}
		if _, err := h.db.Exec("INSERT INTO vectors_ann_chunks (id, chunk) VALUES (?, ?)", ann_chunk_rowid, ann_chunk_data); err != nil {
			return err
		}
		for _, idx := range ann_chunk_index {
			if _, err := h.db.Exec("INSERT INTO vectors_ann_index (vectors_id, chunk_id, chunk_position) VALUES (?, ?, ?)", idx.VectorsID, idx.ChunkRowID, idx.ChunkPosition); err != nil {
				return err
			}
		}

		processedVectors += len(ann_chunk_index)
		ann_chunk_rowid++
	}
}
