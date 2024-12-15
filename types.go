// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

type SearchResult struct {
	Article int
	Title   string
	Entity  string
	Text    string
	Section string
	Type    string
	Power   float64
}

type ArticleResult struct {
	Title   string
	Entity  string
	Section string
	Text    string
	Article int
	BM25    float64
}

type OutputArticle struct {
	Title  string                   `json:"title"`
	Entity string                   `json:"entity"`
	Items  []map[string]interface{} `json:"items"`
	ID     int                      `json:"id"`
}

type InputArticle struct {
	MainEntity struct {
		Identifier string `json:"identifier"`
	} `json:"main_entity"`
	Name        string `json:"name"`
	ArticleBody struct {
		HTML string `json:"html"`
	} `json:"article_body"`
	Identifier int `json:"identifier"`
}

type EmbeddingData struct {
	Hash    string
	Vectors []float32
	Status  int
}
