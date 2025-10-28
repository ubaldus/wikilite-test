// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

//go:build !aiInternal

package main

func aiInternal() bool {
	return false
}

func aiEmbeddings(input string) ([]float32, error) {
	return aiApiEmbeddings(input)
}
