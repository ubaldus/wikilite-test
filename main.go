// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const Version = "0.0.8"

type Config struct {
	importPath       string //https://dumps.wikimedia.org/other/enterprise_html/runs/...
	dbPath           string
	web              bool
	webHost          string
	webPort          int
	ai               bool
	aiApiKey         string
	aiModelEmbedding string
	aiModelLLM       string
	aiUrl            string
}

var (
	db      *DBHandler
	options *Config
)

func parseConfig() (*Config, error) {
	options = &Config{}
	flag.BoolVar(&options.ai, "ai", false, "Enable AI")
	flag.StringVar(&options.aiModelEmbedding, "ai-model-embedding", "bge-m3", "AI embedding model")
	flag.StringVar(&options.aiModelLLM, "ai-model-llm", "gemma2", "AI LLM model")
	flag.StringVar(&options.aiUrl, "ai-url", "http://localhost:11434/v1/", "AI base url")
	flag.StringVar(&options.aiApiKey, "ai-api-key", "", "AI API key")
	flag.StringVar(&options.dbPath, "db", "", "SQLite database path")
	flag.StringVar(&options.importPath, "import", "", "URL or file path to import (default to jsonl if no db is provided)")
	flag.BoolVar(&options.web, "web", false, "Enable web interface")
	flag.StringVar(&options.webHost, "web-host", "localhost", "Web server host")
	flag.IntVar(&options.webPort, "web-port", 35248, "Web server port")

	flag.Usage = func() {
		fmt.Println("Copyright:", "2024 by Ubaldo Porcheddu <ubaldo@eja.it>")
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
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	if options.dbPath != "" {
		db, err = NewDBHandler(options.dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
	}

	if options.ai {
		_, err := aiInstruct("ping")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error contacting the AI engine: %v\n", err)
			os.Exit(1)
		}
	}

	if options.importPath != "" {
		if strings.HasPrefix(options.importPath, "http://") || strings.HasPrefix(options.importPath, "https://") {
			err = downloadAndProcessFile(options.importPath)
		} else {
			err = processLocalFile(options.importPath)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing import: %v\n", err)
			os.Exit(1)
		}
	}

	if options.web && db != nil {
		server, err := NewWebServer(db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating web server: %v\n", err)
			os.Exit(1)
		}

		if err := server.Start(options.webHost, options.webPort); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
			os.Exit(1)
		}
	}

	if db == nil {
		flag.Usage()
		os.Exit(1)
	}
}
