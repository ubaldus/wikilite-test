// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func Search(query string, limit int) ([]SearchResult, error) {
	var results []SearchResult

	log.Println("FTS title searching", query)
	titles, err := db.SearchTitle(query, limit)
	if err != nil {
		return nil, err
	}
	for _, title := range titles {
		results = append(results, title)
	}

	log.Println("FTS content searching", query)
	contents, err := db.SearchContent(query, limit)
	if err != nil {
		return nil, err
	}
	for _, content := range contents {
		results = append(results, content)
	}

	if ai {
		log.Println("Vectors searching", query)
		vectors, err := db.SearchVectors(query, limit)
		if err != nil {
			return nil, err
		}
		for _, vector := range vectors {
			results = append(results, vector)
		}
	}

	return searchOptimize(results), nil
}

func SearchCli() error {
	reader := bufio.NewReader(os.Stdin)
	articles := make(map[int]int)

	for {
		fmt.Print("> ")
		query, _ := reader.ReadString('\n')
		query = strings.TrimSpace(query)
		if query == "" {
			return nil
		}

		queryIdx, err := strconv.Atoi(query)
		if err == nil {
			if articleID, exists := articles[queryIdx]; exists {
				article, err := db.ArticleGet(articleID)
				if err != nil {
					log.Fatal("CLI error: ", err)
				}

				for _, entry := range article {
					fmt.Printf("%s\n\n", entry.Title)

					for _, section := range entry.Sections {
						fmt.Printf("%s\n\n", section.Title)
						for _, text := range section.Texts {
							fmt.Println(text)
						}
						fmt.Println()
					}
				}
				continue
			}
		}

		results, err := Search(query, options.limit)
		if err != nil {
			log.Fatal("CLI error: ", err)
		}

		articles = make(map[int]int)
		for i, result := range results {
			articles[i+1] = result.ArticleID
			fmt.Printf("% 3d [%s] %s\n", i+1, result.Type, result.Title)
		}
	}
}

func searchOptimize(results []SearchResult) []SearchResult {
	seen := make(map[int]bool)
	accumulatedResults := []SearchResult{}

	for _, result := range results {
		if !seen[result.ArticleID] {
			seen[result.ArticleID] = true
			accumulatedResults = append(accumulatedResults, result)
		} else {
			for i := range accumulatedResults {
				if accumulatedResults[i].ArticleID == result.ArticleID {
					accumulatedResults[i].Power += result.Power
					break
				}
			}
		}
	}

	return accumulatedResults
}
