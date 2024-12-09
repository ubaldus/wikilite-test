package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	importPath string // URL or file path to import from
	dbPath     string // Path to SQLite database (optional)
}

var (
	db  *DBHandler
	cfg *Config
)

// parseConfig handles command line arguments and returns a Config
func parseConfig() (*Config, error) {
	cfg := &Config{}

	// Define flags
	flag.StringVar(&cfg.importPath, "import", "", "URL or file path to import (required)")
	flag.StringVar(&cfg.dbPath, "db", "", "SQLite database path (optional, if not specified outputs JSON)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s --import <url/file> [--db file]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --import <url/file>  URL or file path to import (required)\n")
		fmt.Fprintf(os.Stderr, "  --db <file>         SQLite database path (optional)\n")
		fmt.Fprintf(os.Stderr, "\nIf --db is not specified, JSON output will be printed to stdout\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --import data.tar.gz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --import http://example.com/data.tar.gz --db output.db\n", os.Args[0])
	}

	flag.Parse()

	// Validate required import path
	if cfg.importPath == "" {
		return nil, fmt.Errorf("--import is required")
	}

	return cfg, nil
}

func main() {
	// Parse command line configuration
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Initialize database if path specified
	if cfg.dbPath != "" {
		db, err = NewDBHandler(cfg.dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
	}

	// Process the input based on whether it's a URL or local file
	if strings.HasPrefix(cfg.importPath, "http://") || strings.HasPrefix(cfg.importPath, "https://") {
		err = downloadAndProcessFile(cfg.importPath)
	} else {
		err = processLocalFile(cfg.importPath)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing import: %v\n", err)
		os.Exit(1)
	}
}
