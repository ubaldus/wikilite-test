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
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// calculateHash generates an MD5 hash for the given text array.
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

// CombinedCloser is a custom type that implements io.Closer by combining two Closers
type CombinedCloser struct {
	gzipCloser io.Closer
	respCloser io.Closer
}

// Close closes both gzipCloser and respCloser
func (cc CombinedCloser) Close() error {
	if err := cc.gzipCloser.Close(); err != nil {
		return err
	}
	return cc.respCloser.Close()
}

// ArticleBody represents the article_body field in the JSON
type ArticleBody struct {
	HTML json.RawMessage `json:"html"`
}

// Article represents the structure of each JSONL record
type Article struct {
	MainEntity struct {
		Identifier string `json:"identifier"`
	} `json:"main_entity"`
	Name        string      `json:"name"`
	ArticleBody ArticleBody `json:"article_body"`
	Identifier int `json:"identifier"`
}

// downloadAndExtractFile downloads a tar.gz file and extracts its first file.
func downloadAndExtractFile(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Wrap the response body with a gzip reader
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}

	// Create a tar reader to process the decompressed data
	tarReader := tar.NewReader(gzipReader)

	// Extract the first file from the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar file: %v", err)
		}

		// Return the first regular file found
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

// openAndExtractLocalFile opens and extracts the tar.gz file locally.
func openAndExtractLocalFile(filePath string) (io.ReadCloser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}

	// Create a gzip reader to decompress the file
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}

	// Create a tar reader to process the decompressed data
	tarReader := tar.NewReader(gzipReader)

	// Extract the first file from the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, fmt.Errorf("error reading tar file: %v", err)
		}

		// Return the first regular file found
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

// collectText recursively collects text from HTML nodes
func collectText(node *html.Node) string {
	var textContent string

	// Handle text nodes
	if node.Type == html.TextNode {
		textContent += node.Data
	} else if node.Type == html.ElementNode {
		// Skip <sup class="reference"> and its children
		if node.Data == "sup" {
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "reference") {
					return ""
				}
			}
		}

		// Skip <cite> tags entirely
		if node.Data == "cite" {
			return ""
		}

		// Skip <ol class="references"> and <ul class="references">
		if node.Data == "ol" || node.Data == "ul" {
			for _, attr := range node.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "references") {
					return ""
				}
			}
		}

		// Recursively collect text from children, preserving the skipTable flag
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			textContent += collectText(c)
		}
	}

	return textContent
}

func ExtractContentFromHTML(htmlContent string, articleID string, articleTitle string, identifier int) {
	// Parse the HTML content using the html package
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing HTML: %v\n", err)
		return
	}

	var lastHeading string
	var power int
	groupedItems := make(map[string]map[string]interface{}) // Group by `sub`

	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		// Check if the node is a <p> or <h1> to <h6> element
		if n.Type == html.ElementNode {
			switch n.Data {
			case "table":
				return
			case "ul", "ol", "p", "h1", "h2", "h3", "h4", "h5", "h6":
				textContent := collectText(n)
				if strings.TrimSpace(textContent) != "" {
					// If it's a heading (h1-h6), update the last heading
					if n.Data == "h1" || n.Data == "h2" || n.Data == "h3" || n.Data == "h4" || n.Data == "h5" || n.Data == "h6" {
						lastHeading = textContent
						power = extractNumber(n.Data)
					}

					text := strings.TrimSpace(textContent)
					if text != "" {
						// Use lastHeading if available, otherwise use an empty string for `sub`
						subKey := lastHeading
						if subKey == "" {
							subKey = "" // Explicitly set for clarity
						}

						// Initialize the group if it doesn't exist
						if _, exists := groupedItems[subKey]; !exists {
							groupedItems[subKey] = map[string]interface{}{
								"sub":  lastHeading, // Keep the actual heading (can be empty)
								"pow":  power,
								"text": []string{},
							}
						}

						// Avoid adding the `sub` value itself to the `text` array
						if text != lastHeading {
							groupedItems[subKey]["text"] = append(groupedItems[subKey]["text"].([]string), text)
						}
					}
				}
			}
		}

		// Recursively process child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	// Start extracting from the root of the document
	extractText(doc)

	// Convert groupedItems map to an `items` array, filtering out empty text arrays
	var items []map[string]interface{}
	for _, item := range groupedItems {
		// Get the text array and ensure it's not empty
		texts := item["text"].([]string)
		if len(texts) > 0 {
			// Create a new "text" array with hash-text objects
			var textEntries []map[string]string
			for _, text := range texts {
				textEntries = append(textEntries, map[string]string{
					"hash": calculateHash([]string{text}), // Hash for each text entry
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
		return
	}

	// Prepare the final JSON object for this article
	output := map[string]interface{}{
		"title": articleTitle,
		"entity":    articleID,
		"items": items,
		"id": identifier,
	}

	// Convert to JSON and print in JSONL format
	jsonData, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(jsonData))
}

// FileReader interface defines the methods needed to read from a source
type FileReader interface {
	Read([]byte) (int, error)
	Close() error
}

// processJSONLFile handles reading and processing a single JSONL file
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
			fmt.Fprintf(os.Stderr, "Debug: article_body.html is empty or not present in this record")
			continue
		}

		var htmlContent string
		if err := json.Unmarshal(article.ArticleBody.HTML, &htmlContent); err != nil {
			fmt.Fprintf(os.Stderr, "Error unmarshaling article_body.html: %v\n", err)
			continue
		}

		unescapedHTML := html.UnescapeString(htmlContent)
		ExtractContentFromHTML(unescapedHTML, article.MainEntity.Identifier, article.Name, article.Identifier)
	}
	return nil
}

// processTarArchive processes all files in the tar archive
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
			fmt.Fprintf(os.Stderr, "Processing file: %s\n", header.Name)
			if err := processJSONLFile(tarReader); err != nil {
				fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", header.Name, err)
				continue // Continue with next file even if this one fails
			}
		}
	}
	return nil
}

// downloadAndProcessFile downloads and processes a tar.gz file from a URL
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

// processLocalFile processes a local tar.gz file
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <url_or_local_file>")
		os.Exit(1)
	}

	arg := os.Args[1]
	var err error

	if strings.HasPrefix(arg, "http") {
		err = downloadAndProcessFile(arg)
	} else {
		err = processLocalFile(arg)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing file: %v\n", err)
		os.Exit(1)
	}
}
