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

func (s *WebServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		query := r.FormValue("query")
		results, err := Search(query, options.limit)

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
			Language string
		}{
			Query:    query,
			Results:  results,
			HasQuery: query != "",
			Language: options.language,
		}

		s.template.ExecuteTemplate(w, "search.html", data)
		return
	}

	s.template.ExecuteTemplate(w, "search.html", nil)
}

func (s *WebServer) handleArticle(w http.ResponseWriter, r *http.Request) {
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

		s.template.ExecuteTemplate(w, "article.html", data)
	}
}

func (s *WebServer) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request APIRequest
	var query string
	var limit int = options.limit
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Invalid JSON request",
			})
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
				json.NewEncoder(w).Encode(APIResponse{
					Status:  "error",
					Message: "Invalid limit parameter",
				})
				return
			}
		}
	}

	if query == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Query parameter is required",
		})
		return
	}

	results, err := Search(query, limit)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: fmt.Sprintf("Search error: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Results: results,
	})
}

func (s *WebServer) handleAPISearchTitle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request APIRequest
	var query string
	var limit int = options.limit
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Invalid JSON request",
			})
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
				json.NewEncoder(w).Encode(APIResponse{
					Status:  "error",
					Message: "Invalid limit parameter",
				})
				return
			}
		}
	}

	if query == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Query parameter is required",
		})
		return
	}

	results, err := db.SearchTitle(query, limit)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: fmt.Sprintf("Title search error: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Results: results,
	})
}

func (s *WebServer) handleAPISearchContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request APIRequest
	var query string
	var limit int = options.limit
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Invalid JSON request",
			})
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
				json.NewEncoder(w).Encode(APIResponse{
					Status:  "error",
					Message: "Invalid limit parameter",
				})
				return
			}
		}
	}

	if query == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Query parameter is required",
		})
		return
	}

	results, err := db.SearchContent(query, limit)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: fmt.Sprintf("Content search error: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Results: results,
	})
}

func (s *WebServer) handleAPISearchVectors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !options.ai || !options.qdrant {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Vector search is not enabled",
		})
		return
	}

	var request APIRequest
	var query string
	var limit int = options.limit
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Invalid JSON request",
			})
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
				json.NewEncoder(w).Encode(APIResponse{
					Status:  "error",
					Message: "Invalid limit parameter",
				})
				return
			}
		}
	}

	if query == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: "Query parameter is required",
		})
		return
	}

	results, err := db.SearchVectors(query, limit)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: fmt.Sprintf("Vector search error: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Results: results,
	})
}

func (s *WebServer) handleAPIArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request APIRequest
	var id int
	var err error

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Invalid JSON request",
			})
			return
		}
		id = request.ID
	} else {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "ID parameter is required",
			})
			return
		}
		id, err = strconv.Atoi(idStr)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Status:  "error",
				Message: "Invalid ID parameter",
			})
			return
		}
	}

	article, err := db.GetArticle(id)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Status:  "error",
			Message: fmt.Sprintf("Error retrieving article: %v", err),
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Article: article,
	})
}

func (s *WebServer) Start(host string, port int) error {
	http.HandleFunc("/", s.handleSearch)
	http.HandleFunc("/article", s.handleArticle)

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
