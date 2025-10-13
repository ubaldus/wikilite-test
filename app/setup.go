// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
)

const SetupDBListUrl = "https://huggingface.co/api/datasets/eja/wikilite"
const SetupDBBaseUrl = "https://huggingface.co/datasets/eja/wikilite/resolve/main/"

type SetupSibling struct {
	Rfilename string `json:"rfilename"`
}

type SetupDatasetInfo struct {
	Siblings []SetupSibling `json:"siblings"`
}

type SetupProgressReader struct {
	totalSize        int64
	bytesRead        int64
	progressCallback func(float64)
}

func SetupFetchDatasetInfo() (*SetupDatasetInfo, error) {
	resp, err := http.Get(SetupDBListUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var datasetInfo SetupDatasetInfo
	err = json.NewDecoder(resp.Body).Decode(&datasetInfo)
	if err != nil {
		return nil, err
	}

	return &datasetInfo, nil
}

func SetupFilterDBFiles(siblings []SetupSibling) []SetupSibling {
	var dbFiles []SetupSibling
	partRegex := regexp.MustCompile(`\.db(-\d+)?\.gz$`)
	for _, sibling := range siblings {
		if partRegex.MatchString(sibling.Rfilename) {
			dbFiles = append(dbFiles, sibling)
		}
	}
	return dbFiles
}

func SetupGetGGUFFileName(dbFile string) string {
	name := strings.TrimSuffix(dbFile, ".db.gz")
	prefixIndex := strings.Index(name, ".")
	if prefixIndex > 0 {
		return name[prefixIndex+1:] + ".gguf"
	}
	return ""
}

func SetupDownloadFile(file string, outputPath string, progressCallback func(float64)) error {
	resp, err := http.Get(SetupDBBaseUrl + file)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: server returned status %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	teeReader := io.TeeReader(resp.Body, &SetupProgressReader{totalSize: totalSize, progressCallback: progressCallback})

	_, err = io.Copy(out, teeReader)
	return err
}

func SetupGunzipFile(file string, outputPath string, progressCallback func(float64)) error {
	resp, err := http.Get(SetupDBBaseUrl + file)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: server returned status %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength

	teeReader := io.TeeReader(resp.Body, &SetupProgressReader{totalSize: totalSize, progressCallback: progressCallback})

	gzReader, err := gzip.NewReader(teeReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, gzReader)
	if err != nil {
		return fmt.Errorf("failed to write to output file: %v", err)
	}

	return nil
}

func (pr *SetupProgressReader) Write(p []byte) (int, error) {
	n := len(p)
	pr.bytesRead += int64(n)
	if pr.progressCallback != nil {
		progress := float64(pr.bytesRead) / float64(pr.totalSize) * 100
		pr.progressCallback(progress)
	}
	return n, nil
}

func SetupDownloadAndExtract(selectedGroup []SetupSibling, progressDbCallback func(string, float64), progressAiCallback func(string, float64)) error {
	for _, part := range selectedGroup {
		err := SetupGunzipFile(part.Rfilename, options.dbPath, func(progress float64) {
			if progressDbCallback != nil {
				progressDbCallback(part.Rfilename, progress)
			}
		})
		if err != nil {
			return fmt.Errorf("error downloading and extracting file %s: %v", part.Rfilename, err)
		} else {
			db, _ = NewDBHandler(options.dbPath)
		}
	}

	ggufFile := SetupGetGGUFFileName(selectedGroup[0].Rfilename)
	if ggufFile != "" {
		if _, err := os.Stat(ggufFile); err == nil {
			return fmt.Errorf("%s already exists", ggufFile)
		}

		err := SetupDownloadFile("models/"+ggufFile, ggufFile, func(progress float64) {
			if progressAiCallback != nil {
				progressAiCallback(ggufFile, progress)
			}
		})
		if err != nil {
			return fmt.Errorf("error downloading .gguf file: %v", err)
		} else {
			if err := aiInit(); err == nil {
				ai = true
			}
		}
	}

	return nil
}

func Setup() {
	datasetInfo, err := SetupFetchDatasetInfo()
	if err != nil {
		fmt.Println("Error fetching dataset info:", err)
		return
	}

	dbFiles := SetupFilterDBFiles(datasetInfo.Siblings)
	if len(dbFiles) == 0 {
		fmt.Println("No valid database files found.")
		return
	}

	fileGroups := make(map[string][]SetupSibling)
	for _, dbFile := range dbFiles {
		baseName := strings.Split(dbFile.Rfilename, ".db")[0]
		fileGroups[baseName] = append(fileGroups[baseName], dbFile)
	}

	var groupKeys []string
	for key := range fileGroups {
		groupKeys = append(groupKeys, key)
	}

	sort.Strings(groupKeys)
	for i, key := range groupKeys {
		fmt.Printf("%d. %s\n", i+1, key)
	}

	fmt.Print("Choose a file by number: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	var choice int
	_, err = fmt.Sscanf(input, "%d", &choice)
	if err != nil || choice < 1 || choice > len(groupKeys) {
		fmt.Println("Invalid choice.")
		return
	}

	selectedGroup := fileGroups[groupKeys[choice-1]]

	if _, err := os.Stat(options.dbPath); err == nil {
		fmt.Println("A db already exists, please remove it and try again.")
		return
	}

	err = SetupDownloadAndExtract(selectedGroup, func(file string, progress float64) {
		fmt.Printf("\rDownloading %s: %.2f%%", file, progress)
		if progress >= 100 {
			fmt.Println()
		}
	}, func(file string, progress float64) {
		fmt.Printf("\rDownloading %s: %.2f%%", file, progress)
	})
	fmt.Println()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Setup ready.")
}
