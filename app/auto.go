// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"log"
	"os"
	"path/filepath"
)

func autoStart() {
	exePath, err := os.Executable()
	if err != nil {
		log.Println(err)
	}
	exeDir := filepath.Dir(exePath)
	if err := os.Chdir(exeDir); err != nil {
		log.Println(err)
	}
	if _, err := os.Stat(options.dbPath); err != nil {
		log.Println("path", err)
		options.setup = true
	}

	options.web = true
	options.log = true
}
