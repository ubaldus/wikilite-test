module wikilite

go 1.24.0

toolchain go1.24.8

require (
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/ollama/ollama v0.12.5
	golang.org/x/net v0.46.0
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/emirpasic/gods/v2 v2.0.0-alpha // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/image v0.22.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
)

replace github.com/ollama/ollama => ./ollama
