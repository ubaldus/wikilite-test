// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func autoWeb() {
	ask := ReadLine("Launch web search? (Y/n): ")
	if ask != "n" {
		options.web = true
		ask = ReadLine("Launch browser when ready? (Y/n): ")
		if ask != "n" {
			options.webBrowser = true
		}
	}
}

func autoCli() {
	ask := ReadLine("Launch CLI search? (Y/n): ")
	if ask != "n" {
		options.cli = true
	}
}

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
	}

	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		autoWeb()
		if options.web == false {
			autoCli()
		}
	} else {
		autoCli()
		if options.cli == false {
			autoWeb()
		}
	}

	if os.Getenv("TERMUX_VERSION") != "" {
		options.dbPath = "/data/data/com.termux/files/home/wikilite.db"
	}

	if options.web == false && options.cli == false {
		flag.Usage()
		os.Exit(0)
	}
}
