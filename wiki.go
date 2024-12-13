// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func calculateHash(texts []string) string {
	hasher := md5.New()
	for _, text := range texts {
		hasher.Write([]byte(text))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func extractNumber(s string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(s)
	if match != "" {
		num, err := strconv.Atoi(match)
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

type CombinedCloser struct {
	gzipCloser io.Closer
	respCloser io.Closer
}

func (cc CombinedCloser) Close() error {
	if err := cc.gzipCloser.Close(); err != nil {
		return err
	}
	return cc.respCloser.Close()
}

type ArticleBody struct {
	HTML json.RawMessage `json:"html"`
}

type Article struct {
	MainEntity struct {
		Identifier string `json:"identifier"`
	} `json:"main_entity"`
	Name        string      `json:"name"`
	ArticleBody ArticleBody `json:"article_body"`
	Identifier  int         `json:"identifier"`
}

func downloadAndExtractFile(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar file: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			return struct {
				io.Reader
				io.Closer
			}{
				Reader: tarReader,
				Closer: CombinedCloser{
					gzipCloser: gzipReader,
					respCloser: resp.Body,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no regular file found in tar archive")
}

func openAndExtractLocalFile(filePath string) (io.ReadCloser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar file: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			return struct {
				io.Reader
				io.Closer
			}{
				Reader: tarReader,
				Closer: CombinedCloser{
					gzipCloser: gzipReader,
					respCloser: file,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no regular file found in tar archive")
}

type FileReader interface {
	Read([]byte) (int, error)
	Close() error
}

func processTarArchive(tarReader *tar.Reader) error {
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar file: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			log.Printf("Processing file: %s\n", header.Name)
			if err := processJSONLFile(tarReader); err != nil {
				log.Printf("Error processing file %s: %v\n", header.Name, err)
				continue // Continue with next file even if this one fails
			}
		}
	}
	return nil
}

func downloadAndProcessFile(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	return processTarArchive(tarReader)
}

func processLocalFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	return processTarArchive(tarReader)
}

func processJSONLFile(reader io.Reader) error {
	jsonDecoder := json.NewDecoder(reader)
	for {
		var article Article
		if err := jsonDecoder.Decode(&article); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error decoding JSONL: %v", err)
		}

		if article.ArticleBody.HTML == nil {
			log.Println("Debug: article_body.html is empty or not present in this record")
			continue
		}

		var htmlContent string
		if err := json.Unmarshal(article.ArticleBody.HTML, &htmlContent); err != nil {
			log.Printf("Error unmarshaling article_body.html: %v\n", err)
			continue
		}

		output := ExtractContentFromHTML(htmlContent, article.MainEntity.Identifier, article.Name, article.Identifier)

		if output != nil && db != nil {
			if err := db.SaveArticle(*output); err != nil {
				log.Printf("Error saving to database: %v\n", err)
				continue
			}
			log.Printf("Saved article %d to database\n", article.Identifier)
		}
	}
	return nil
}

func hasExternalLink(node *html.Node) bool {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "external") {
				return true
			}
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if hasExternalLink(c) {
			return true
		}
	}
	return false
}

func ExtractContentFromHTML(htmlContent string, articleID string, articleTitle string, identifier int) *OutputArticle {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML: %v\n", err)
		return nil
	}

	var lastHeading string
	var power int
	groupedItems := make(map[string]map[string]interface{}) // Group by `sub`

	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "table", "style", "script", "math":
				return // Skip these elements
			case "sup":
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "reference") {
						return // Skip reference sup elements
					}
				}
			case "li":
				if hasExternalLink(n) {
					return // Skip li with external link
				}
				if n.Parent != nil && (n.Parent.Data == "ul" || n.Parent.Data == "ol") {
					for _, attr := range n.Parent.Attr {
						if attr.Key == "class" && strings.Contains(attr.Val, "references") {
							return
						}
					}
				}
				// Process valid li
				processLiElement(n, &lastHeading, &power, groupedItems)
			case "p":
				processTextElement(n, &lastHeading, &power, groupedItems)
			case "h1", "h2", "h3", "h4", "h5", "h6":
				textContent := collectTextFromNode(n)
				if strings.TrimSpace(textContent) != "" {
					lastHeading = textContent
					power = extractNumber(n.Data)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}
	extractText(doc)

	var items []map[string]interface{}
	for _, item := range groupedItems {
		// Get the text array and ensure it's not empty
		texts := item["text"].([]string)
		if len(texts) > 0 {
			// Create a new "text" array with hash-text objects
			var textEntries []map[string]string
			for _, text := range texts {
				textEntries = append(textEntries, map[string]string{
					"hash": calculateHash([]string{text}),
					"text": text,
				})
			}

			// Add the modified item to the final output
			items = append(items, map[string]interface{}{
				"sub":  item["sub"],
				"pow":  item["pow"],
				"text": textEntries, // Replace the plain text array with hash-text objects
			})
		}
	}
	if len(items) == 0 {
		return nil
	}

	output := OutputArticle{
		Title:  articleTitle,
		Entity: articleID,
		Items:  items,
		ID:     identifier,
	}

	// If db is not provided print JSON to stdout
	if db == nil {
		jsonData, err := json.Marshal(output)
		if err != nil {
			log.Printf("Error marshaling JSON: %v\n", err)
			return nil
		}
		fmt.Println(string(jsonData))
	}
	return &output
}

func processLiElement(node *html.Node, lastHeading *string, power *int, groupedItems map[string]map[string]interface{}) {
	textContent := collectTextFromNode(node)
	if strings.TrimSpace(textContent) != "" {
		// Add bullet point
		textContent = "\u2022 " + textContent
	}
	processTextElementWithText(textContent, lastHeading, power, groupedItems)
}
func processTextElementWithText(textContent string, lastHeading *string, power *int, groupedItems map[string]map[string]interface{}) {
	trimmedText := strings.TrimSpace(textContent)
	if trimmedText == "" {
		return
	}
	subKey := *lastHeading
	if subKey == "" {
		subKey = "" // Explicitly set for clarity
	}

	if _, exists := groupedItems[subKey]; !exists {
		groupedItems[subKey] = map[string]interface{}{
			"sub":  *lastHeading, // Keep the actual heading (can be empty)
			"pow":  *power,
			"text": []string{},
		}
	}
	if trimmedText != *lastHeading {
		groupedItems[subKey]["text"] = append(groupedItems[subKey]["text"].([]string), trimmedText)
	}

}

func processTextElement(node *html.Node, lastHeading *string, power *int, groupedItems map[string]map[string]interface{}) {
	textContent := collectTextFromNode(node)
	processTextElementWithText(textContent, lastHeading, power, groupedItems)
}

func collectTextFromNode(node *html.Node) string {
	var textContent string

	if node.Type == html.TextNode {
		textContent += node.Data
	} else if node.Type == html.ElementNode {
		switch node.Data {
		case "style", "script", "math", "table":
			return textContent // Skip these elements
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			textContent += collectTextFromNode(c)
		}
	}
	return textContent
}
