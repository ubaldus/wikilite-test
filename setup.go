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
	baseName := strings.TrimSuffix(dbFile, ".db.gz")
	parts := strings.Split(baseName, ".")
	if len(parts) == 2 {
		return parts[1] + ".gguf"
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

	out, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

	if _, err := os.Stat("wikilite.db"); err == nil {
		fmt.Println("A wikilite.db already exists in the current directory.")
		return
	}

	for _, part := range selectedGroup {
		fmt.Println("Downloading and extracting", part.Rfilename)
		err = SetupGunzipFile(part.Rfilename, "wikilite.db", func(progress float64) {
			fmt.Printf("\rDownloaded %.2f%%", progress)
		})
		fmt.Println()
		if err != nil {
			fmt.Println("Error downloading and extracting file:", err)
			return
		}
	}
	fmt.Println("Saved as wikilite.db")

	ggufFile := SetupGetGGUFFileName(selectedGroup[0].Rfilename)
	if ggufFile != "" {
		if _, err := os.Stat(ggufFile); err == nil {
			fmt.Printf("%s already exists in the current directory.\n", ggufFile)
			return
		}

		fmt.Println("Downloading gguf model:", ggufFile)
		err = SetupDownloadFile("models/"+ggufFile, ggufFile, func(progress float64) {
			fmt.Printf("\rDownloaded %.2f%%", progress)
		})
		fmt.Println()
		if err != nil {
			fmt.Println("Error downloading .gguf file:", err)
			return
		}
		fmt.Println("Saved as", ggufFile)
	}
}
