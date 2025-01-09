// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type WikiCombinedCloser struct {
	gzipCloser io.Closer
	respCloser io.Closer
}

type WikiFileReader interface {
	Read([]byte) (int, error)
	Close() error
}

func WikiImport(path string) error {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return wikiDownloadAndProcessFile(path)
	} else {
		return wikiProcessLocalFile(path)
	}
}

func (cc WikiCombinedCloser) Close() error {
	if err := cc.gzipCloser.Close(); err != nil {
		return err
	}
	return cc.respCloser.Close()
}

func wikiDownloadAndExtractFile(url string) (io.ReadCloser, error) {
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
				Closer: WikiCombinedCloser{
					gzipCloser: gzipReader,
					respCloser: resp.Body,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no regular file found in tar archive")
}

func wikiOpenAndExtractLocalFile(filePath string) (io.ReadCloser, error) {
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
				Closer: WikiCombinedCloser{
					gzipCloser: gzipReader,
					respCloser: file,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no regular file found in tar archive")
}

func wikiProcessTarArchive(tarReader *tar.Reader) error {
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
			if err := wikiProcessJSONLFile(tarReader); err != nil {
				log.Printf("Error processing file %s: %v\n", header.Name, err)
				continue // Continue with next file even if this one fails
			}
		}
	}
	return nil
}

func wikiDownloadAndProcessFile(url string) error {
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
	return wikiProcessTarArchive(tarReader)
}

func wikiProcessLocalFile(filePath string) error {
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
	return wikiProcessTarArchive(tarReader)
}

func wikiProcessJSONLFile(reader io.Reader) error {
	jsonDecoder := json.NewDecoder(reader)
	for {
		var art InputArticle
		if err := jsonDecoder.Decode(&art); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error decoding JSONL: %v", err)
		}

		if art.ArticleBody.HTML == "" {
			log.Println("Debug: article_body.html is empty or not present in this record")
			continue
		}

		output := wikiExtractContentFromHTML(art.ArticleBody.HTML, art.MainEntity.Identifier, art.Name, art.Identifier)

		if output != nil && db != nil {
			if err := db.SaveArticle(*output); err != nil {
				log.Printf("Error saving to database: %v\n", err)
				continue
			}
			log.Printf("Saved article %d to database\n", art.Identifier)
		}
	}
	return nil
}

func wikiHasExternalLink(node *html.Node) bool {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "external") {
				return true
			}
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if wikiHasExternalLink(c) {
			return true
		}
	}
	return false
}

func wikiCollectTextFromNode(node *html.Node) string {
	var textContent string

	if node.Type == html.TextNode {
		textContent += node.Data
	} else if node.Type == html.ElementNode {
		switch node.Data {
		case "style", "script", "math", "table":
			return textContent
		case "sup":
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "reference") {
					return textContent
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			textContent += wikiCollectTextFromNode(c)
		}
	}
	return textContent
}

func wikiProcessLiElement(node *html.Node, lastHeading *string, power *int, groupedItems *[]map[string]interface{}) {
	textContent := wikiCollectTextFromNode(node)
	if strings.TrimSpace(textContent) != "" {
		// Add bullet point
		textContent = "\u2022 " + textContent
	}
	wikiProcessTextElementWithText(textContent, lastHeading, power, groupedItems)
}

func wikiProcessTextElementWithText(textContent string, lastHeading *string, power *int, groupedItems *[]map[string]interface{}) {
	trimmedText := strings.TrimSpace(textContent)
	if trimmedText == "" {
		return
	}
	subKey := *lastHeading
	if subKey == "" {
		subKey = "" // Explicitly set for clarity
	}

	found := false
	for i, item := range *groupedItems {
		if item["title"] == subKey {
			(*groupedItems)[i]["text"] = append((*groupedItems)[i]["text"].([]string), trimmedText)
			found = true
			break
		}
	}

	if !found {
		*groupedItems = append(*groupedItems, map[string]interface{}{
			"title": *lastHeading,
			"pow":   *power,
			"text":  []string{trimmedText},
		})
	}
}

func wikiProcessTextElement(node *html.Node, lastHeading *string, power *int, groupedItems *[]map[string]interface{}) {
	textContent := wikiCollectTextFromNode(node)
	wikiProcessTextElementWithText(textContent, lastHeading, power, groupedItems)
}

func wikiExtractContentFromHTML(htmlContent string, articleID string, articleTitle string, identifier int) *OutputArticle {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML: %v\n", err)
		return nil
	}

	var lastHeading string
	var power int

	groupedItems := []map[string]interface{}{}

	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "table", "style", "script", "math":
				return
			case "sup":
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "reference") {
						return
					}
				}
			case "ul", "ol":
				var liTexts []string
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Data == "li" {
						if wikiHasExternalLink(c) {
							continue
						}
						if c.Parent != nil {
							for _, attr := range c.Parent.Attr {
								if attr.Key == "class" && strings.Contains(attr.Val, "references") {
									continue
								}
							}
						}
						textContent := wikiCollectTextFromNode(c)
						if strings.TrimSpace(textContent) != "" {
							liTexts = append(liTexts, "\u2022 "+textContent)
						}
					}
				}
				if len(liTexts) > 0 {
					wikiProcessTextElementWithText(strings.Join(liTexts, "\n"), &lastHeading, &power, &groupedItems)
				}
			case "p":
				wikiProcessTextElement(n, &lastHeading, &power, &groupedItems)
			case "h1", "h2", "h3", "h4", "h5", "h6":
				textContent := wikiCollectTextFromNode(n)
				if strings.TrimSpace(textContent) != "" {
					lastHeading = textContent
					power = extractNumberFromString(n.Data)
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
		texts := item["text"].([]string)
		if len(texts) > 0 {
			var textEntries []map[string]string
			for _, text := range texts {
				if text != item["title"] {
					textEntries = append(textEntries, map[string]string{
						"hash": calculateHash([]string{text}),
						"text": text,
					})
				}
			}

			items = append(items, map[string]interface{}{
				"title": item["title"],
				"pow":   item["pow"],
				"text":  textEntries,
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
	return &output
}
