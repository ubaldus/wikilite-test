// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func autoStart() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exeDir := filepath.Dir(exePath)
	if err := os.Chdir(exeDir); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(options.dbPath); err != nil {
		options.setup = true
		fmt.Printf("When the setup is ready navigate to http://localhost:%d\n\n", options.webPort)
	} else {
		fmt.Printf("Navigate to http://localhost:%d\n", options.webPort)
	}

	options.web = true
}
