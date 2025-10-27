// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"fmt"
	"os"
)

func (h *DBHandler) AiModelImport(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	_, err = h.db.Exec(`INSERT OR REPLACE INTO setup (key, value) VALUES ("gguf", ?)`, data)

	if err != nil {
		return fmt.Errorf("failed to import model into database: %v", err)
	}

	return nil
}

func (h *DBHandler) AiModelLoad() []byte {
	var data []byte

	row := h.db.QueryRow(`SELECT value FROM setup WHERE key = "gguf" LIMIT 1`)

	err := row.Scan(&data)
	if err != nil {
		return nil
	}

	return data
}
