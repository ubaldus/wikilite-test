// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type APIRequest struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
	ID    int    `json:"id,omitempty"`
}

type APIResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	Results []SearchResult `json:"results,omitempty"`
	Article ArticleResult  `json:"article,omitempty"`
	Time    float64        `json:"time"`
}

type WebServer struct {
	template *template.Template
}

var SetupProgressMap = struct {
	sync.RWMutex
	m map[string]float64
}{m: make(map[string]float64)}

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
	var err error
	var query string
	var limit int
	var results []SearchResult

	if r.Method == "POST" {
		query = r.FormValue("query")
		limit, _ = strconv.Atoi(r.FormValue("limit"))
	}

	if limit <= 0 {
		limit = options.limit
	}

	if query != "" {
		results, err = Search(query, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s.executeTemplate(w, "search.html", struct {
		Query    string
		Limit    int
		Results  []SearchResult
		HasQuery bool
		Language string
		AI       bool
	}{
		Query:    query,
		Limit:    limit,
		Results:  results,
		HasQuery: query != "",
		Language: options.language,
		AI:       ai,
	})
}

func (s *WebServer) handleHTMLArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		value := r.FormValue("id")
		id, err := strconv.Atoi(value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := db.ArticleGet(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.executeTemplate(w, "article.html", struct {
			Language string
			Result   ArticleResult
		}{
			Language: options.language,
			Result:   result,
		})
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

func (s *WebServer) handleGenericAPISearch(w http.ResponseWriter, r *http.Request, searchFunc func(query string, limit int) ([]SearchResult, error)) {
	w.Header().Set("Content-Type", "application/json")

	var request APIRequest
	var query string
	var limit int = options.limit
	var err error

	startTime := time.Now()

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
	log.Printf("API %s search: %s", r.Method, query)

	if query == "" {
		s.sendAPIError(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	results, err := searchFunc(query, limit)
	if err != nil {
		s.sendAPIError(w, fmt.Sprintf("Search error: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Results: results,
		Time:    time.Since(startTime).Seconds(),
	})

}

func (s *WebServer) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	s.handleGenericAPISearch(w, r, Search)
}

func (s *WebServer) handleAPISearchTitle(w http.ResponseWriter, r *http.Request) {
	s.handleGenericAPISearch(w, r, SearchTitle)
}

func (s *WebServer) handleAPISearchLexical(w http.ResponseWriter, r *http.Request) {
	s.handleGenericAPISearch(w, r, SearchLexical)
}

func (s *WebServer) handleAPISearchSemantic(w http.ResponseWriter, r *http.Request) {
	if !ai {
		s.sendAPIError(w, "Semantic search is not enabled", http.StatusBadRequest)
		return
	}
	s.handleGenericAPISearch(w, r, SearchSemantic)
}

func (s *WebServer) handleAPIArticle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var request APIRequest
	var id int
	var err error

	startTime := time.Now()

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
	log.Printf("API %s article: %d", r.Method, id)

	article, err := db.ArticleGet(id)
	if err != nil {
		s.sendAPIError(w, fmt.Sprintf("Error retrieving article: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  "success",
		Article: article,
		Time:    time.Since(startTime).Seconds(),
	})
}

func handleSetup(w http.ResponseWriter, r *http.Request) {
	if db.IsEmpty() {
		action := r.URL.Query().Get("action")
		switch action {
		case "list":
			setupListHandler(w, r)
		case "install":
			setupInstallHandler(w, r)
		case "progress":
			setupProgressHandler(w, r)
		default:
			http.Error(w, "Invalid action parameter", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Setup not available now", http.StatusForbidden)
	}
}

func setupListHandler(w http.ResponseWriter, r *http.Request) {
	datasetInfo, err := SetupFetchDatasetInfo()
	if err != nil {
		http.Error(w, "Error fetching dataset info", http.StatusInternalServerError)
		return
	}

	dbFiles := SetupFilterDBFiles(datasetInfo.Siblings)
	json.NewEncoder(w).Encode(dbFiles)
}

func setupInstallHandler(w http.ResponseWriter, r *http.Request) {
	dbFile := r.URL.Query().Get("file")
	if dbFile == "" {
		http.Error(w, "dbFile parameter is required", http.StatusBadRequest)
		return
	}

	go func() {
		err := SetupGunzipFile(dbFile, "wikilite.db", func(progress float64) {
			SetupProgressMap.Lock()
			SetupProgressMap.m[dbFile] = progress
			SetupProgressMap.Unlock()
		})
		if err != nil {
			log.Println("Error downloading and extracting file:", err)
			return
		}

		ggufFile := SetupGetGGUFFileName(dbFile)
		if ggufFile != "" {
			err = SetupDownloadFile("models/"+ggufFile, ggufFile)
			if err != nil {
				fmt.Println("Error downloading .gguf file:", err)
				return
			}
		}
	}()

	fmt.Fprintf(w, "Downloading and extracting %s", dbFile)
}

func setupProgressHandler(w http.ResponseWriter, r *http.Request) {
	dbFile := r.URL.Query().Get("file")
	if dbFile == "" {
		http.Error(w, "dbFile parameter is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		SetupProgressMap.RLock()
		progress, exists := SetupProgressMap.m[dbFile]
		SetupProgressMap.RUnlock()
		if exists {
			fmt.Fprintf(w, "data: %.2f\n\n", progress)
			w.(http.Flusher).Flush()
			if progress >= 100 {
				SetupProgressMap.Lock()
				delete(SetupProgressMap.m, dbFile)
				SetupProgressMap.Unlock()
				return
			}
		}
	}
}

func (s *WebServer) handleHome(w http.ResponseWriter, r *http.Request) {
	if db.IsEmpty() {
		http.Redirect(w, r, "/static/setup.html", http.StatusFound)
	} else {
		s.handleHTMLSearch(w, r)
	}
}

func (s *WebServer) Start(host string, port int) error {
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/article", s.handleHTMLArticle)

	http.HandleFunc("/api/search", s.handleAPISearch)
	http.HandleFunc("/api/search/title", s.handleAPISearchTitle)
	http.HandleFunc("/api/search/lexical", s.handleAPISearchLexical)
	http.HandleFunc("/api/search/semantic", s.handleAPISearchSemantic)
	http.HandleFunc("/api/article", s.handleAPIArticle)
	http.HandleFunc("/api/setup", handleSetup)

	subFS, err := fs.Sub(assets, "assets/static")
	if err != nil {
		panic(fmt.Errorf("failed to access embedded subdirectory: %w", err))
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	address := fmt.Sprintf("%s:%d", host, port)
	if options.webTlsPrivate != "" && options.webTlsPublic != "" {
		if _, err := os.Stat(options.webTlsPrivate); err != nil {
			return fmt.Errorf("failed to open private certificate")
		} else if _, err := os.Stat(options.webTlsPublic); err != nil {
			return fmt.Errorf("failed to open public certificate")
		} else {
			log.Println("Starting server on https://" + address)
			if err := http.ListenAndServeTLS(address, options.webTlsPublic, options.webTlsPrivate, nil); err != nil {
				return err
			}
		}
	} else {
		log.Println("Starting server on http://" + address)
		if err := http.ListenAndServe(address, nil); err != nil {
			return err
		}
	}

	return nil
}

func WebStart(host string, port int) error {
	server, err := NewWebServer()
	if err != nil {
		return err
	}

	if err := server.Start(host, port); err != nil {
		return err
	}

	return nil
}
