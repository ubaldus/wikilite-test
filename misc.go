// Copyright (C) 2024 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// deflateText compresses a string using gzip
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

// inflateText decompresses a gzipped byte array back to string
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
