# Wikilite

Wikilite is a tool that allows you to create a local SQLite database of Wikipedia articles, indexed with FTS5 for fast and efficient searching. Additionally, it supports using Qdrant as a vector database for enhanced AI capabilities. Built with Go, Wikilite provides a command-line interface (CLI) and an optional web interface for accessing the database, enabling offline searching and browsing of Wikipedia content.

## Features

*   **Fast and Flexible Searching**: Leverages FTS5 for efficient text-based searching within the SQLite database.
*   **Vector Database Support**: Integrates with Qdrant for advanced AI-powered searches and semantic understanding using embeddings.
*   **Offline Access**: Access Wikipedia articles without an active internet connection.
*   **Command-Line Interface (CLI)**: Search and query the database directly from your terminal.
*   **Web Interface (Optional)**: Browse and search articles through a user-friendly web interface.
*   **AI Integration (Optional)**: Utilizes AI models for embeddings and advanced query understanding.

## Getting Started

1.  **Clone the repository**: `git clone https://github.com/eja/wikilite.git`
2.  **Build the Wikilite binary**: `make`
3.  **Import Wikipedia data**: `./wikilite --import <url> --db <file>`

### Web Interface

1.  **Start the web server**: `./wikilite --web --db <file>`
2.  **Access the web interface**: Open a web browser and navigate to `http://localhost:35248`

## Acknowledgments

*   **Wikipedia**: For providing the valuable data that powers Wikilite.
*   **SQLite**: For providing the robust database engine that enables fast and efficient local data storage.
*   **Qdrant**: For the high-performance vector database, used for semantic search capabilities.
