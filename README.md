# Wikilite

Wikilite is a tool that allows you to create a local SQLite database of Wikipedia articles, indexed with FTS5 for fast and efficient lexical searching. It further enhances search capabilities by leveraging an existing Qdrant vector database server for semantic search, enabling users to find information even when their queries don't match keywords exactly. Built with Go, Wikilite provides a command-line interface (CLI) and an optional web interface, enabling offline access, browsing, and searching of Wikipedia content.

## Features

*   **Fast and Flexible Lexical Searching**: Leverages FTS5 (Full-Text Search 5) for efficient and fast keyword-based searching within the SQLite database. This is great for finding exact matches of words and phrases in your query.
*  **Enhanced Semantic Search**: Integrates with Qdrant, a vector database, for powerful semantic search capabilities. This complements the FTS5 search, finding results that are *semantically similar* to your query, even if they lack exact keyword matches. This handles issues like misspellings, plurals/singulars, and different verb tenses.
*   **Offline Access**: Access Wikipedia articles without an active internet connection.
*   **Command-Line Interface (CLI)**: Search and query the database directly from your terminal.
*   **Web Interface (Optional)**: Browse and search articles through a user-friendly web interface.
*   **Semantic Search Integration (Optional)**: Leverages text embeddings for improved query understanding and retrieval based on meaning, complementing the lexical search.

## Getting Started

1.  **Clone the repository**: `git clone https://github.com/eja/wikilite.git`
2.  **Build the Wikilite binary**: `make`
3.  **Import Wikipedia data**: `./wikilite --import <url> --db <file>`

### Web Interface

1.  **Start the web server**: `./wikilite --web --db <file>`
2.  **Access the web interface**: Open a web browser and navigate to `http://localhost:35248`

## Semantic Search Details

Wikilite utilizes text embeddings to power its semantic search capabilities. This means that instead of just looking for exact keyword matches (like FTS5 does), it searches for paragraphs that have a *similar meaning* to your query. This is particularly helpful in scenarios where:

*   You have typos in your search query.
*   You are using different wordings to express the same concept.
*   The article uses synonyms or related terms instead of the precise words you searched for.

The semantic search acts as a powerful complement to FTS5, allowing you to get more relevant results even when your query doesn't match directly.

## Acknowledgments

*   **Wikipedia**: For providing the valuable data that powers Wikilite.
*   **SQLite**: For providing the robust database engine that enables fast and efficient local data storage.
*   **Qdrant**: For the high-performance vector database, which enables the semantic search functionality.
