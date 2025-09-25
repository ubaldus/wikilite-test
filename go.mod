module wikilite

go 1.24.0

toolchain go1.24.7

require (
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/ollama/ollama v0.12.2
	golang.org/x/net v0.44.0
)

replace github.com/ollama/ollama => ./ollama
