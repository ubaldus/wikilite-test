// Copyright (C) 2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

const Version = "0.6.0"

type Config struct {
	aiApiKey      string
	aiApiUrl      string
	aiModel       string
	aiModelPath   string
	aiSync        bool
	cli           bool
	dbPath        string
	language      string
	limit         int
	log           bool
	logFile       string
	setup         bool
	web           bool
	webHost       string
	webPort       int
	webTlsPrivate string
	webTlsPublic  string
	wikiImport    string //https://dumps.wikimedia.org/other/enterprise_html/runs/...
}

var (
	ai      bool
	db      *DBHandler
	options *Config
)

func parseConfig() (*Config, error) {
	options = &Config{}
	flag.StringVar(&options.aiApiKey, "ai-api-key", "", "AI API key")
	flag.StringVar(&options.aiApiUrl, "ai-api-url", "", "AI API base url")
	flag.StringVar(&options.aiModel, "ai-model", "all-minilm", "AI embedding model")
	flag.StringVar(&options.aiModelPath, "ai-model-path", "", "AI embedding local model file (default \"all-minilm.gguf\")")
	flag.BoolVar(&options.aiSync, "ai-sync", false, "AI generate embeddings")

	flag.BoolVar(&options.cli, "cli", false, "Interactive search")

	flag.StringVar(&options.dbPath, "db", "wikilite.db", "SQLite database path")

	flag.StringVar(&options.language, "language", "en", "Language")
	flag.IntVar(&options.limit, "limit", 5, "Maximum number of search results")
	flag.BoolVar(&options.log, "log", false, "Enable logging")
	flag.StringVar(&options.logFile, "log-file", "", "Log file path")
	flag.BoolVar(&options.setup, "setup", false, "Download a ready made database and embeddings model")

	flag.BoolVar(&options.web, "web", false, "Enable web interface")
	flag.StringVar(&options.webHost, "web-host", "localhost", "Web server host")
	flag.IntVar(&options.webPort, "web-port", 35248, "Web server port")
	flag.StringVar(&options.webTlsPrivate, "web-tls-private", "", "TLS private certificate")
	flag.StringVar(&options.webTlsPublic, "web-tls-public", "", "TLS public certificate")

	flag.StringVar(&options.wikiImport, "wiki-import", "", "URL or file path for wikipedia import")

	flag.Usage = func() {
		fmt.Println("Copyright:", "2024-2025 by Ubaldo Porcheddu <ubaldo@eja.it>")
		fmt.Println("Version:", Version)
		fmt.Printf("Usage: %s [options]\n", os.Args[0])
		fmt.Println("Options:\n")
		flag.PrintDefaults()
		fmt.Println()
	}

	flag.Parse()

	return options, nil
}

func main() {
	options, err := parseConfig()
	if err != nil {
		log.Fatalf("Error parsing command line: %v\n", err)
	}

	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if options.setup {
		Setup()
	}

	if options.log || options.logFile != "" {
		if options.logFile != "" {
			logFile, err := os.OpenFile(options.logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
			}
			log.SetOutput(logFile)
		}
	} else {
		log.SetOutput(io.Discard)
	}

	db, err = NewDBHandler(options.dbPath)
	if err != nil {
		log.Fatalf("Error initializing database: %v\n", err)
	}
	defer db.Close()

	if err := aiInit(); err != nil {
		log.Printf("AI initialization error: %v\n", err)
	} else {
		ai = true
	}

	if options.aiSync || options.wikiImport != "" {
		if err := db.PragmaImportMode(); err != nil {
			log.Fatalf("Error setting database in import mode: %v\n", err)
		}

		if options.wikiImport != "" {
			if err = WikiImport(options.wikiImport); err != nil {
				log.Fatalf("Error processing import: %v\n", err)
			}
		}

		if ai && options.aiSync {
			if err := db.ProcessEmbeddings(); err != nil {
				log.Fatalf("Error processing embeddings: %v\n", err)
			}
		}

		if err := db.PragmaReadMode(); err != nil {
			log.Fatalf("Error setting database in read mode: %v\n", err)
		}
	}

	if options.cli {
		SearchCli()
	}

	if options.web {
		server, err := NewWebServer()
		if err != nil {
			log.Fatalf("Error creating web server: %v\n", err)
		}

		if err := server.Start(options.webHost, options.webPort); err != nil {
			log.Fatalf("Error starting web server: %v\n", err)
		}
	}

}
