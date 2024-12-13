// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
)

//go:embed assets
var assets embed.FS

// WebServer represents a web server
type WebServer struct {
	db       *DBHandler
	template *template.Template
}

func assetsDir(path string) fs.FS {
	subFS, err := fs.Sub(assets, path)
	if err != nil {
		panic(fmt.Errorf("failed to access embedded subdirectory %s: %w", path, err))
	}
	return subFS
}

// NewWebServer creates a new web server
func NewWebServer(db *DBHandler) (*WebServer, error) {
	tmpl, err := template.ParseFS(assets, "assets/templates/*")
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	return &WebServer{
		db:       db,
		template: tmpl,
	}, nil
}

// handleSearch handles search requests
func (s *WebServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		query := r.FormValue("query")
		results, err := s.db.SearchArticles(query, 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
			return
		}

		data := struct {
			Query    string
			Results  []SearchResult
			HasQuery bool
		}{
			Query:    query,
			Results:  results,
			HasQuery: query != "",
		}

		s.template.ExecuteTemplate(w, "search.html", data)
		return
	}

	s.template.ExecuteTemplate(w, "search.html", nil)
}

// handleArticle handles article requests
func (s *WebServer) handleArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		value := r.FormValue("id")
		id, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results, err := s.db.GetArticle(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Results []ArticleResult
		}{
			Results: results,
		}

		s.template.ExecuteTemplate(w, "article.html", data)
	}
}

// Start starts the web server
func (s *WebServer) Start(host string, port int) error {
	http.HandleFunc("/", s.handleSearch)
	http.HandleFunc("/article", s.handleArticle)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(assetsDir("assets/static")))))

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting web server at http://%s/\n", addr)
	return http.ListenAndServe(addr, nil)
}
