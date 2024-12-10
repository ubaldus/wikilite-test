// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

//go:embed templates/*
var templates embed.FS

type WebServer struct {
	db       *DBHandler
	template *template.Template
}

func NewWebServer(db *DBHandler) (*WebServer, error) {
	tmpl, err := template.ParseFS(templates, "templates/*")
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	return &WebServer{
		db:       db,
		template: tmpl,
	}, nil
}

func (s *WebServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		query := r.FormValue("query")
		results, err := s.db.SearchArticles(query, 10)
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

func (s *WebServer) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	http.HandleFunc("/", s.handleSearch)
	fmt.Printf("Starting web server at http://%s/\n", addr)
	return http.ListenAndServe(addr, nil)
}
