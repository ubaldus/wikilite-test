// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
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

func (h *DBHandler) ProcessVocabulary() error {
	_, err := h.db.Exec("INSERT OR IGNORE INTO vocabulary SELECT term FROM article_search_vocabulary")
	if err != nil {
		return fmt.Errorf("error populating vocabulary table: %v", err)
	}

	_, err = h.db.Exec("INSERT OR IGNORE INTO vocabulary SELECT term FROM hash_search_vocabulary")
	if err != nil {
		return fmt.Errorf("error populating vocabulary table: %v", err)
	}

	return nil
}

func (h *DBHandler) ProcessEmbeddings() (err error) {
	batchSize := 250
	totalCount := 0
	offset := 0

	aiModelBasename := filepath.Base(options.aiModel)
	aiModelName := strings.TrimSuffix(aiModelBasename, ".gguf")
	if err = db.SetupPut("model", aiModelName); err != nil {
		return
	}

	if err = db.SetupPut("modelPrefixSave", options.aiModelPrefixSave); err != nil {
		return
	}

	if err = db.SetupPut("modelPrefixSearch", options.aiModelPrefixSearch); err != nil {
		return
	}

	err = h.db.QueryRow("SELECT COUNT(*) FROM sections WHERE id NOT IN (SELECT id FROM vectors)").Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("error getting total count of sections: %w", err)
	}

	log.Printf("Pending section embeddings: %d", totalCount)

	startTime := time.Now()
	var problematicIDs []int
	for {
		tx, err := h.db.Begin()
		if err != nil {
			return fmt.Errorf("error starting transaction: %w", err)
		}

		query := `SELECT s.id, s.title, a.title 
          FROM sections s 
          JOIN articles a ON s.article_id = a.id 
          WHERE s.id NOT IN (SELECT id FROM vectors)`
		if len(problematicIDs) > 0 {
			idList := make([]string, 0, len(problematicIDs))
			for _, id := range problematicIDs {
				idList = append(idList, fmt.Sprintf("%d", id))
			}
			query += " AND s.id NOT IN (" + strings.Join(idList, ", ") + ")"
		}
		query += " LIMIT ?"

		rows, err := tx.Query(query, batchSize)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error querying sections: %w", err)
		}

		var sectionIDs []int
		var sectionTitles []string
		var articleTitles []string

		for rows.Next() {
			var sectionID int
			var sectionTitle, articleTitle string
			if err := rows.Scan(&sectionID, &sectionTitle, &articleTitle); err != nil {
				rows.Close()
				tx.Rollback()
				return fmt.Errorf("error scanning section row: %w", err)
			}
			sectionIDs = append(sectionIDs, sectionID)
			sectionTitles = append(sectionTitles, sectionTitle)
			articleTitles = append(articleTitles, articleTitle)
		}
		rows.Close()

		if err := rows.Err(); err != nil {
			tx.Rollback()
			return fmt.Errorf("error iterating section rows: %w", err)
		}

		if len(sectionIDs) == 0 {
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

		for i, sectionID := range sectionIDs {
			var texts []string
			textRows, err := tx.Query("SELECT h.text FROM hashes h JOIN content c ON h.id = c.hash_id WHERE c.section_id = ? ORDER BY c.id", sectionID)
			if err != nil {
				log.Printf("Error getting text for section %d: %v", sectionID, err)
				problematicIDs = append(problematicIDs, sectionID)
				continue
			}
			for textRows.Next() {
				var text string
				if err := textRows.Scan(&text); err != nil {
					textRows.Close()
					log.Printf("Error scanning text for section %d: %v", sectionID, err)
					problematicIDs = append(problematicIDs, sectionID)
					continue
				}
				texts = append(texts, text)
			}
			textRows.Close()
			fullSectionText := articleTitles[i] + " - " + sectionTitles[i] + "\n\n" + strings.Join(texts, "\n\n")

			embedding, err := aiEmbeddings(options.aiModelPrefixSave + fullSectionText)
			if err != nil {
				log.Printf("Embedding generation error for section %d: %v", sectionID, err)
				problematicIDs = append(problematicIDs, sectionID)
				continue
			}

			if _, err := tx.Exec("INSERT OR REPLACE INTO vectors (id, embedding) VALUES (?, ?)", sectionID, Float32ToBytes(embedding)); err != nil {
				log.Printf("Error inserting vector for section %d: %v", sectionID, err)
				problematicIDs = append(problematicIDs, sectionID)
				continue
			}
			if _, err := tx.Exec("INSERT INTO vectors_ann_index (vectors_id, chunk_id, chunk_position) VALUES (?, ?, ?)", sectionID, ann_chunk_rowid, ann_chunk_position); err != nil {
				log.Printf("Error inserting vectors_ann for section %d: %v", sectionID, err)
				problematicIDs = append(problematicIDs, sectionID)
				continue
			}
			ann_chunk_data = append(ann_chunk_data, QuantizeBinary(embedding)...)
			ann_chunk_position++
		}
		if _, err := tx.Exec("INSERT INTO vectors_ann_chunks (id, chunk) VALUES (?, ?)", ann_chunk_rowid, ann_chunk_data); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
		}

		processedCount := offset + len(sectionIDs)
		progress := float64(processedCount) / float64(totalCount) * 100
		elapsed := time.Since(startTime)
		estimatedTotalTime := time.Duration(float64(elapsed) / (progress / 100.0))
		remainingTime := estimatedTotalTime - elapsed

		log.Printf("Embedding progress: %.2f%%, Estimated total time: %s, Remaining: %s", progress, estimatedTotalTime.Truncate(time.Second), remainingTime.Truncate(time.Second))

		offset += len(sectionIDs)
	}

	return nil
}
