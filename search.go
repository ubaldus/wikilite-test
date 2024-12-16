package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func search(query string, limit int) ([]SearchResult, error) {
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

func searchCli() error {
	reader := bufio.NewReader(os.Stdin)
	var articles map[int]int
	for {
		fmt.Print("> ")
		query, _ := reader.ReadString('\n')
		query = strings.TrimSpace(query)
		queryIdx, err := strconv.Atoi(query)

		if err == nil && articles[queryIdx] > 0 {
			article, err := db.GetArticle(articles[queryIdx])
			if err != nil {
				log.Fatal("cli error", err)
			}

			title := ""
			section := ""
			for _, entry := range article {
				if entry.Title != title {
					title = entry.Title
					fmt.Printf("%s\n\n", title)
				}
				if entry.Section != section {
					section = entry.Section
					fmt.Printf("\n%s\n\n", section)
				}
				fmt.Println(entry.Text)
			}
			fmt.Println()

		} else {

			results, err := search(query, options.limit)
			if err != nil {
				log.Fatal("cli error", err)
			}

			articles = make(map[int]int)
			for i, result := range results {
				articles[i+1] = result.Article
				fmt.Printf("% 3d [%s] %s\n", i+1, result.Type, result.Title)
			}
		}
	}
}
