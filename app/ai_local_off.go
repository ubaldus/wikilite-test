// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

//go:build !aiLocal

package main

import "fmt"

func localAiEnabled() bool {
	return false
}

func localAiInit(modelPath string) error {
	return fmt.Errorf("local embeddings are not supported on this platform")
}

func localAiEmbeddings(input string) ([]float32, error) {
	return nil, fmt.Errorf("local embeddings are not supported on this platform")
}
