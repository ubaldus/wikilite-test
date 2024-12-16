package main

import (
	"fmt"
	"log"
)

func search(query string, db *DBHandler) ([]SearchResult, error) {
	limit := QueryLimit
	var results []SearchResult

	log.Println("FTS title searching", query)
	titles, err := db.searchTitle(query, limit)
	if err != nil {
		return nil, err
	}
	for _, title := range titles {
		results = append(results, title)
	}

	log.Println("FTS content searching", query)
	contents, err := db.searchContent(query, limit)
	if err != nil {
		return nil, err
	}
	for _, content := range contents {
		results = append(results, content)
	}

	if options.ai && options.qdrant {
		log.Println("AI searching", query)
		vectorsQuery, err := aiEmbeddings(query)
		if err != nil {
			return nil, fmt.Errorf("embeddings generation error: %w", err)
		}
		hashes, scores, err := qdrantSearch(qd.PointsClient, options.qdrantCollection, vectorsQuery, limit*limit)
		embeddingsResults, err := db.SearchHash(hashes, scores, limit)
		if err != nil {
			return nil, fmt.Errorf("database hashes search error: %w", err)
		}
		for _, result := range embeddingsResults {
			results = append(results, result)
		}
	}

	return results, nil
}
