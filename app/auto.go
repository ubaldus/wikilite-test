// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func autoStart() {
	ask := ""
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
	}

	ask = ReadLine("Launch web search? (Y/n): ")
	if ask != "n" {
		options.web = true
		ask = ReadLine("Launch Browser when ready? (Y/n): ")
		if ask != "n" {
			options.webBrowser = true
		}
	} else {
		ask = ReadLine("Launch CLI search? (Y/n): ")
		if ask != "n" {
			options.cli = true
			fmt.Println("")
		} else {
			flag.Usage()
		}
	}
}
