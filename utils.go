// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"compress/flate"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"io"
	"regexp"
	"strconv"
	"strings"
)

//go:embed assets
var assets embed.FS

func calculateHash(texts []string) string {
	hasher := md5.New()
	for _, text := range texts {
		hasher.Write([]byte(text))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func cleanHashes(hashes []string) []string {
	cleanedHashes := make([]string, len(hashes))

	for i, hash := range hashes {
		cleanedHash := strings.ToLower(strings.ReplaceAll(hash, "-", ""))
		cleanedHashes[i] = cleanedHash
	}

	return cleanedHashes
}

func extractNumberFromString(s string) int {
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

func TextInflate(data []byte) string {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	var out bytes.Buffer
	_, err := io.Copy(&out, reader)
	if err != nil {
		return ""
	}

	return out.String()
}

func TextDeflate(text string) []byte {
	var out bytes.Buffer

	writer, err := flate.NewWriter(&out, flate.DefaultCompression)
	if err != nil {
		return nil
	}
	defer writer.Close()

	_, err = writer.Write([]byte(text))
	if err != nil {
		return nil
	}

	err = writer.Close()
	if err != nil {
		return nil
	}

	return out.Bytes()
}
