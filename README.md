# Wikilite
Wikilite is a SQLite database of Wikipedia articles, indexed with FTS5 for fast and efficient searching. Built with Go, Wikilite provides a command-line interface and a web interface for accessing the database, allowing you to search and browse Wikipedia articles offline.

## Features
* **Fast and flexible searching**: Utilize FTS5 for efficient searching
* **Offline access**: Access Wikipedia articles without an internet connection
* **Command-line interface**: Search and query the database from the terminal
* **Web interface**: Browse and search articles through a user-friendly web interface

## Getting Started
1. **Clone the repository**: `git clone https://github.com/eja/wikilite.git`
2. **Build the Wikilite binary**: `make`
3. **Import Wikipedia data**: `./wikilite --import <url>`

## Usage
### Command-line Interface
* **Search for articles**: `./wikilite --search "your search query"`

### Web Interface
1. **Start the web server**: `./wikilite --web --db <file>`
2. **Access the web interface**: Open a web browser and navigate to `http://localhost:35248`

## Acknowledgments
* **Wikipedia**: For providing the data that powers Wikilite
* **SQLite**: For providing the database engine that makes Wikilite possible
