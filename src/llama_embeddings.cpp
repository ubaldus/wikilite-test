// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

#include "llama_embeddings.h"
#include "common.h"
#include "llama.h"
#include "ggml.h"
#include <vector>
#include <string>

static common_params g_params;
static llama_model* g_model = nullptr;
static llama_context* g_ctx = nullptr;
static bool g_initialized = false;

void silent_log_callback(ggml_log_level level, const char * text, void * user_data) {
    (void)level;
    (void)text;
    (void)user_data;
}

int llama_embeddings_init(const char* model_path) {
    llama_log_set(silent_log_callback, NULL);

    if (g_initialized) {
        return 0;
    }

    g_params.model.path = model_path;
    g_params.embedding = true;
    g_params.embd_normalize = 2;
    g_params.warmup = false;

    if (g_params.n_parallel == 1) {
        g_params.kv_unified = true;
    }

    g_params.n_gpu_layers = 999;

    common_init_result llama_init = common_init_from_params(g_params);
    g_model = llama_init.model.release();
    g_ctx = llama_init.context.release();

    if (g_model == nullptr || g_ctx == nullptr) {
        llama_log_set(NULL, NULL);
        fprintf(stderr, "Error: Failed to initialize model or context from '%s'\n", model_path);
        return 1;
    }

    g_initialized = true;
    return 0;
}

int llama_embeddings_get_dimension(void) {
    if (!g_initialized) return -1;
    return llama_model_n_embd(g_model);
}

float* llama_embeddings_get(const char* text, int* n_embd_out) {
    if (!g_initialized || !text || n_embd_out == NULL) {
        if (n_embd_out) *n_embd_out = 0;
        return NULL;
    }

    std::vector<llama_token> tokens = common_tokenize(g_ctx, text, true, true);
    if (tokens.empty()) {
        *n_embd_out = 0;
        return NULL;
    }

    llama_batch batch = llama_batch_init(tokens.size(), 0, 1);
    for (size_t i = 0; i < tokens.size(); ++i) {
        common_batch_add(batch, tokens[i], i, { 0 }, true);
    }

    llama_memory_clear(llama_get_memory(g_ctx), true);

    if (llama_decode(g_ctx, batch) < 0) {
        fprintf(stderr, "Error: llama_decode failed\n");
        llama_batch_free(batch);
        *n_embd_out = 0;
        return NULL;
    }

    enum llama_pooling_type pooling_type = llama_pooling_type(g_ctx);
    if (pooling_type == LLAMA_POOLING_TYPE_NONE) {
        fprintf(stderr, "Warning: model does not have a pooling type. This wrapper requires a pooling model.\n");
        llama_batch_free(batch);
        *n_embd_out = 0;
        return NULL;
    }

    const int n_embd = llama_model_n_embd(g_model);
    float* output = (float*)malloc(n_embd * sizeof(float));
    if (!output) {
        fprintf(stderr, "Error: Failed to allocate memory\n");
        llama_batch_free(batch);
        *n_embd_out = 0;
        return NULL;
    }

    const float* embd_in = llama_get_embeddings_seq(g_ctx, 0);
    if (embd_in == nullptr) {
        fprintf(stderr, "Error: failed to get sequence embeddings\n");
        free(output);
        llama_batch_free(batch);
        *n_embd_out = 0;
        return nullptr;
    }

    common_embd_normalize(embd_in, output, n_embd, g_params.embd_normalize);

    llama_batch_free(batch);
    *n_embd_out = 1;
    return output;
}

void llama_embeddings_free_output(float* output) {
    if (output) {
        free(output);
    }
}

void llama_embeddings_free(void) {
    if (g_initialized) {
        if (g_ctx) llama_free(g_ctx);
        if (g_model) llama_model_free(g_model);
        llama_backend_free();
        g_model = NULL;
        g_ctx = NULL;
        g_initialized = false;
    }
}
