// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

//go:embed assets
var assets embed.FS

func deflateText(text string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write([]byte(text)); err != nil {
		return nil, fmt.Errorf("compression write error: %v", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("compression close error: %v", err)
	}

	return buf.Bytes(), nil
}

func inflateText(compressed []byte) (string, error) {
	if len(compressed) == 0 {
		return "", nil
	}

	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", fmt.Errorf("decompression reader error: %v", err)
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("decompression read error: %v", err)
	}

	return string(decompressed), nil
}

func float32ToBlob(floats []float32) ([]byte, error) {
	bytes := make([]byte, len(floats)*4) // 4 bytes per float32
	for i, float := range floats {
		binary.LittleEndian.PutUint32(bytes[i*4:(i+1)*4], uint32(math.Float32bits(float)))
	}
	return bytes, nil
}

func blobToFloat32(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("length of input bytes is not a multiple of 4")
	}
	numFloats := len(data) / 4
	floats := make([]float32, numFloats)

	for i := 0; i < numFloats; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		floats[i] = math.Float32frombits(bits)
	}

	return floats, nil
}

func calculateHash(texts []string) string {
	hasher := md5.New()
	for _, text := range texts {
		hasher.Write([]byte(text))
	}
	return hex.EncodeToString(hasher.Sum(nil))
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

func cleanHashes(hashes []string) []string {
	cleanedHashes := make([]string, len(hashes))

	for i, hash := range hashes {
		cleanedHash := strings.ToLower(strings.ReplaceAll(hash, "-", ""))
		cleanedHashes[i] = cleanedHash
	}

	return cleanedHashes
}
