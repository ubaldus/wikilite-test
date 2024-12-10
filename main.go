// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const Version = "0.0.6"

type Config struct {
	importPath string // URL or file path to import from (https://dumps.wikimedia.org/other/enterprise_html/runs/...)
	dbPath     string // Path to SQLite database
	web        bool   // Enable web interface
	webHost    string // Web server host
	webPort    int    // Web server port
}

var (
	db      *DBHandler
	options *Config
)

func parseConfig() (*Config, error) {
	options := &Config{}

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

	// Process import
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

	// Start web server if requested
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
