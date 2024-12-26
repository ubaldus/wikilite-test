// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

type SearchResult struct {
	Article int     `json:"article"`
	Title   string  `json:"title"`
	Entity  string  `json:"entity"`
	Text    string  `json:"text"`
	Section string  `json:"section"`
	Type    string  `json:"type"`
	Power   float64 `json:"power"`
}

type ArticleResult struct {
	Title   string `json:"title"`
	Entity  string `json:"entity"`
	Section string `json:"section"`
	Text    string `json:"text"`
	Article int    `json:"article"`
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
