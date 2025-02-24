// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

type SearchResult struct {
	ArticleID int     `json:"article_id"`
	Title     string  `json:"title"`
	Text      string  `json:"text"`
	Type      string  `json:"type"`
	Power     float64 `json:"power"`
}

type ArticleResultSection struct {
	ID    int      `json:"id"`
	Title string   `json:"title"`
	Texts []string `json:"texts"`
}

type ArticleResult struct {
	ID       int                    `json:"id"`
	Title    string                 `json:"title"`
	Entity   string                 `json:"entity"`
	Sections []ArticleResultSection `json:"sections"`
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
