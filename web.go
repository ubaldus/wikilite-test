// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
)

type APIRequest struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
	ID    int    `json:"id,omitempty"`
}

type APIResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Results []SearchResult  `json:"results,omitempty"`
	Article []ArticleResult `json:"article,omitempty"`
}

type WebServer struct {
	template *template.Template
}

func NewWebServer() (*WebServer, error) {
	tmpl, err := template.ParseFS(assets, "assets/templates/*")
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	return &WebServer{
		template: tmpl,
	}, nil
}

func (s *WebServer) executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	err := s.template.ExecuteTemplate(w, templateName, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("error executing template: %v", err), http.StatusInternalServerError)
		return
	}
}
func (s *WebServer) handleHTMLSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		query := r.FormValue("query")
		results, err := Search(query, options.limit)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Query    string
			Results  []SearchResult
			HasQuery bool
			Language string
		}{
			Query:    query,
			Results:  results,
			HasQuery: query != "",
			Language: options.language,
		}
		s.executeTemplate(w, "search.html", data)
		return
	}
	s.executeTemplate(w, "search.html", nil)
}

func (s *WebServer) handleHTMLArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		value := r.FormValue("id")
		id, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results, err := db.GetArticle(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Language string
			Results  []ArticleResult
		}{
			Language: options.language,
			Results:  results,
		}
		s.executeTemplate(w, "article.html", data)
	}
}

func (s *WebServer) sendAPIError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Status:  "error",
		Message: message,
	})
}

func (s *WebServer) handleGenericAPISearch(w http.ResponseWriter, r *http.Request, searchFunc func(query string, limit int) ([]SearchResult, error), searchType string) {
	w.Header().Set("Content-Type", "application/json")

	var request APIRequest
	var query string
	var limit int = options.limit
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			s.sendAPIError(w, "Invalid JSON request", http.StatusBadRequest)
			return
		}
		query = request.Query
		if request.Limit > 0 {
			limit = request.Limit
		}
	} else {
		query = r.URL.Query().Get("query")
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				s.sendAPIError(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
		}
	}

	if query == "" {
		s.sendAPIError(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	results, err := searchFunc(query, limit)
	if err != nil {
		s.sendAPIError(w, fmt.Sprintf("%s search error: %v", searchType, err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Results: results,
	})

}

func (s *WebServer) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	s.handleGenericAPISearch(w, r, Search, "Search")
}

func (s *WebServer) handleAPISearchTitle(w http.ResponseWriter, r *http.Request) {
	s.handleGenericAPISearch(w, r, db.SearchTitle, "Title")
}

func (s *WebServer) handleAPISearchContent(w http.ResponseWriter, r *http.Request) {
	s.handleGenericAPISearch(w, r, db.SearchContent, "Content")
}

func (s *WebServer) handleAPISearchVectors(w http.ResponseWriter, r *http.Request) {
	if !options.ai || !options.qdrant {
		s.sendAPIError(w, "Vector search is not enabled", http.StatusBadRequest)
		return
	}
	s.handleGenericAPISearch(w, r, db.SearchVectors, "Vector")
}

func (s *WebServer) handleAPIArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var request APIRequest
	var id int
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			s.sendAPIError(w, "Invalid JSON request", http.StatusBadRequest)
			return
		}
		id = request.ID
	} else {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			s.sendAPIError(w, "ID parameter is required", http.StatusBadRequest)
			return
		}
		id, err = strconv.Atoi(idStr)
		if err != nil {
			s.sendAPIError(w, "Invalid ID parameter", http.StatusBadRequest)
			return
		}
	}

	article, err := db.GetArticle(id)
	if err != nil {
		s.sendAPIError(w, fmt.Sprintf("Error retrieving article: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Article: article,
	})
}

func (s *WebServer) Start(host string, port int) error {
	http.HandleFunc("/", s.handleHTMLSearch)
	http.HandleFunc("/article", s.handleHTMLArticle)

	http.HandleFunc("/api/search", s.handleAPISearch)
	http.HandleFunc("/api/search/title", s.handleAPISearchTitle)
	http.HandleFunc("/api/search/content", s.handleAPISearchContent)
	http.HandleFunc("/api/search/vectors", s.handleAPISearchVectors)
	http.HandleFunc("/api/article", s.handleAPIArticle)

	subFS, err := fs.Sub(assets, "assets/static")
	if err != nil {
		panic(fmt.Errorf("failed to access embedded subdirectory: %w", err))
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Starting web server at http://%s/\n", addr)
	return http.ListenAndServe(addr, nil)
}
