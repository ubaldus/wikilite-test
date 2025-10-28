// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

#ifndef LLAMA_EMBEDDINGS_H
#define LLAMA_EMBEDDINGS_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h> 

#ifdef __cplusplus
extern "C" {
#endif

int llama_embeddings_init(const char* model_path, int n_threads);

int llama_embeddings_get_dimension(void);

float* llama_embeddings_get(const char* text, int* n_embd_out);

void llama_embeddings_free_output(float* output);

void llama_embeddings_free(void);

void llama_copy_memory_buffer(const void* buf, size_t size);

#ifdef __cplusplus
}
#endif

#endif
