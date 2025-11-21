# Wikilite

Wikilite is a self-contained tool for creating a local SQLite database of Wikipedia articles, indexed with FTS5 for efficient lexical searching with optional semantic search capabilities through embedded embeddings. Built with Go, Wikilite provides command-line tools, a web interface, and a native Android application for offline access, browsing, and searching of Wikipedia content.

## Features

* **Lexical Search**: Utilizes FTS5 for efficient keyword-based searching within the SQLite database, ideal for exact word and phrase matching.
* **Optional Semantic Search**: Implements ANN quantization and MRL (Matryoshka Representation Learning) with text embeddings to find semantically similar content, effectively handling misspellings, morphological variations, and synonymy.
* **Complete llama.cpp Integration**: Full AI engine embedded directly into Wikilite with GGUF models contained within database files.
* **Cross-Platform & Android**: Available for Linux, macOS, Windows, Termux, and as a native Android application.
* **Minimal Deployment**: Requires only the Wikilite executable and the database file on POSIX platforms.
* **Offline Operation**: Complete functionality without internet connectivity.
* **Dual Interfaces**: Command-line interface for terminal usage and web interface for browser-based access.
* **Interactive Wizard**: When started without command-line options, Wikilite enters an interactive mode that guides users through database setup and search operations.

## Installation

### Source Compilation
* Clone the repository: `git clone --recursive https://github.com/eja/wikilite.git`
* Build the binary: `make`
* Check the available options: `./build/bin/wikilite --help`

### Pre-built Binaries & Android App
Pre-compiled binaries for Linux, macOS, Windows, and Termux are available in the [latest release](https://github.com/eja/wikilite/releases/latest).

A native [Android](https://github.com/eja/wikilite/releases/latest/download/wikilite-android.apk) application is also available in the releases.
*   **External Storage Support**: If a `wikilite.db` file is already present in the external SD card, the Android app will detect and use it directly.
*   **In-App Download**: If no database is found on launch, the app provides an option to download a pre-built database.

### Pre-built Database Installation
Run Wikilite without arguments to launch the interactive wizard:
```bash
./wikilite
```

## Usage

Wikilite can be used in several ways:

**Interactive Mode** (recommended for new users):
```bash
./wikilite
```
This launches a wizard that guides you through database installation, CLI search, and web interface setup.

**Direct Command Line**:
```bash
./wikilite --cli --db <file.db>
```

**Web Interface Only**:
```bash
./wikilite --web --db <file.db>
```
Access the interface at `http://localhost:35248`

## API Documentation

Wikilite provides a comprehensive RESTful API supporting both GET and POST methods. Key endpoints include:

* `/api/search`: Combined search across titles, content, and vectors
* `/api/search/title`: Title-specific search
* `/api/search/lexical`: Full-text search of titles and content
* `/api/search/semantic`: Vector-based semantic search
* `/api/search/distance`: Vocabulary distance search
* `/api/article`: Article retrieval by ID

All search endpoints support pagination via the `limit` parameter and return consistent JSON formatting. Complete API documentation is available in the [API specification](API.md).

## Semantic Search Implementation

The semantic search functionality employs text embeddings with GGUF models embedded directly in the database. This approach identifies content with similar semantic meaning rather than relying solely on lexical matching, providing enhanced search capabilities for:

* Query misspellings and typographical errors
* Conceptual similarity despite different terminology
* Synonym and related term matching
* Morphological variations (plurals, verb tenses)

Semantic search complements the FTS5 lexical search to deliver more comprehensive results.

## Pre-built Databases

Pre-configured databases for multiple languages are available on [Hugging Face](https://huggingface.co/datasets/eja/wikilite/tree/main). These can be installed directly through the setup command, the interactive wizard, or downloaded and extracted manually.

Databases in the "lexical" directory support full-text search only, while others include both lexical and semantic search capabilities.

## Acknowledgments

* **Wikipedia**: For providing the valuable data that powers Wikilite.
* **SQLite**: For providing the robust database engine that enables fast and efficient local data storage.
* **LLaMA.cpp**: For enabling the internal generation of embeddings, enhancing the standalone semantic search capabilities of Wikilite.
