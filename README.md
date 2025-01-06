# Wikilite

Wikilite is a tool that allows you to create a local SQLite database of Wikipedia articles, indexed with FTS5 for fast and efficient lexical searching and optional embeddings for semantic searching. Built with Go, Wikilite provides a command-line interface (CLI) and an optional web interface, enabling offline access, browsing, and searching of Wikipedia content.

## Features

*   **Fast and Flexible Lexical Searching**: Leverages FTS5 (Full-Text Search 5) for efficient and fast keyword-based searching within the SQLite database. This is great for finding exact matches of words and phrases in your query.
*  **Enhanced Semantic Search**: Integrates ANN quantization for powerful semantic search capabilities. This complements the FTS5 search, finding results that are *semantically similar* to your query, even if they lack exact keyword matches. This handles issues like misspellings, plurals/singulars, and different verb tenses.
*   **Offline Access**: Access Wikipedia articles without an active internet connection.
*   **Command-Line Interface (CLI)**: Search and query the database directly from your terminal.
*   **Web Interface (Optional)**: Browse and search articles through a user-friendly web interface.
*   **Semantic Search Integration (Optional)**: Leverages text embeddings for improved query understanding and retrieval based on meaning, complementing the lexical search.

## Getting Started

1.  **Clone the repository**: `git clone https://github.com/eja/wikilite.git`
2.  **Build the Wikilite binary**: `make`
3.  **Import Wikipedia data**:  `./wikilite --import <url> --db <file.db>`

### Web Interface

1.  **Start the web server**: `./wikilite --web --db <file.db>`
2.  **Access the web interface**: Open a web browser and navigate to `http://localhost:35248`

## API Overview

Wikilite provides a comprehensive RESTful API that supports both GET and POST methods for all endpoints. The main endpoints include:

* `/api/search`: Combined search across titles, content, and vectors (if enabled)
* `/api/search/title`: Search only article titles
* `/api/search/content`: Search article content
* `/api/search/vectors`: Vector-based semantic search
* `/api/article`: Retrieve complete articles by ID

All search endpoints support pagination through the `limit` parameter and return results in a consistent JSON format. For detailed API documentation and examples, please refer to the [API Documentation](API.md).

## Semantic Search Details

Wikilite utilizes text embeddings to power its semantic search capabilities. This means that instead of just looking for exact keyword matches (like FTS5 does), it searches for paragraphs that have a *similar meaning* to your query. This is particularly helpful in scenarios where:

*   You have typos in your search query.
*   You are using different wordings to express the same concept.
*   The article uses synonyms or related terms instead of the precise words you searched for.

The semantic search acts as a powerful complement to FTS5, allowing you to get more relevant results even when your query doesn't match directly.

## Pre-built Databases

Pre-built databases for several languages are also available on [Hugging Face](https://huggingface.co/datasets/eja/wikilite/tree/main). You can use these databases directly with Wikilite by downloading and decompressing them.

## Supported Data Formats

Wikilite expects Wikipedia dump files in XML format.

## Acknowledgments

*   **Wikipedia**: For providing the valuable data that powers Wikilite.
*   **SQLite**: For providing the robust database engine that enables fast and efficient local data storage.
