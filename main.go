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
	web        bool   // Enable web interface
	host       string // Web server host
	port       int    // Web server port
}

var (
	db  *DBHandler
	cfg *Config
)

func parseConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.importPath, "import", "", "URL or file path to import")
	flag.StringVar(&cfg.dbPath, "db", "", "SQLite database path (optional, if not specified outputs JSON)")
	flag.BoolVar(&cfg.web, "web", false, "Enable web interface")
	flag.StringVar(&cfg.host, "host", "localhost", "Web server host")
	flag.IntVar(&cfg.port, "port", 8080, "Web server port")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s --import <url/file> [--db file] [--web] [--host host] [--port port]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --import <url/file>  URL or file path to import\n")
		fmt.Fprintf(os.Stderr, "  --db <file>         SQLite database path\n")
		fmt.Fprintf(os.Stderr, "  --web              Enable web interface\n")
		fmt.Fprintf(os.Stderr, "  --host             Web server host (default: localhost)\n")
		fmt.Fprintf(os.Stderr, "  --port             Web server port (default: 8080)\n")
	}

	flag.Parse()

	return cfg, nil
}

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	if cfg.dbPath != "" {
		db, err = NewDBHandler(cfg.dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
	}

	// Process import
	if cfg.importPath != "" {
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

	// Start web server if requested
	if cfg.web && db != nil {
		server, err := NewWebServer(db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating web server: %v\n", err)
			os.Exit(1)
		}

		if err := server.Start(cfg.host, cfg.port); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
			os.Exit(1)
		}
	}
}
