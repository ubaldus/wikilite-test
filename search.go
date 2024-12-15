package main

import (
	"math"
	"sort"
)

func searchCalculatePower(titles []SearchResult, contents []SearchResult) []int {
	type Pair struct {
		Key   int
		Value float64
	}
	articleCounts := make(map[int]float64)
	maxPower := -math.MaxFloat64
	minPower := math.MaxFloat64

	for _, result := range titles {
		if result.Power < minPower {
			minPower = result.Power
		}
		if result.Power > maxPower {
			maxPower = result.Power
		}
	}
	for _, result := range contents {
		if result.Power < minPower {
			minPower = result.Power
		}
		if result.Power > maxPower {
			maxPower = result.Power
		}
	}

	if maxPower < 0 {
		maxPower *= -1
	}
	if minPower < 0 {
		minPower *= -1
	}

	for _, result := range titles {
		articleCounts[result.Article] += minPower + maxPower
	}
	for _, result := range contents {
		articleCounts[result.Article] += minPower
	}

	pairs := make([]Pair, 0, len(articleCounts))
	for k, v := range articleCounts {
		pairs = append(pairs, Pair{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	sortedArticles := make([]int, 0, len(pairs))
	for _, pair := range pairs {
		sortedArticles = append(sortedArticles, pair.Key)
	}

	return sortedArticles
}

func search(query string, db *DBHandler) ([]SearchResult, error) {
	titles, err := db.searchTitle(query, 10)
	if err != nil {
		return nil, err
	}

	contents, err := db.searchContent(query, 10)
	if err != nil {
		return nil, err
	}

	articlePower := searchCalculatePower(titles, contents)

	articleMap := make(map[int]SearchResult)
	for _, res := range titles {
		articleMap[res.Article] = res
	}
	for _, res := range contents {
		if _, found := articleMap[res.Article]; !found {
			articleMap[res.Article] = res
		}
	}

	results := make([]SearchResult, 0, len(articlePower))
	for _, articleID := range articlePower {
		if res, found := articleMap[articleID]; found {
			results = append(results, res)
		}

	}

	return results, nil

}
