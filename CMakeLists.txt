cmake_minimum_required(VERSION 3.14)
project(wikilite C CXX)

set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)

set(LLAMA_BUILD_COMMON ON CACHE BOOL "Enable common utils" FORCE)

# Force a CPU-only build by disabling all GPU backends
set(LLAMA_CUBLAS OFF CACHE BOOL "Disable CUDA support" FORCE)
set(LLAMA_HIPBLAS OFF CACHE BOOL "Disable HIPBLAS support" FORCE)
set(LLAMA_CLBLAST OFF CACHE BOOL "Disable OpenCL support" FORCE)
set(LLAMA_METAL OFF CACHE BOOL "Disable Metal support" FORCE)
set(LLAMA_VULKAN OFF CACHE BOOL "Disable Vulkan support" FORCE)
set(GGML_BLAS OFF CACHE BOOL "Disable BLAS support" FORCE)
set(GGML_METAL OFF CACHE BOOL "Disable Metal support" FORCE)
set(GGML_CUDA OFF CACHE BOOL "Disable CUDA support" FORCE)
set(GGML_NATIVE OFF CACHE BOOL "Disable native architecture optimizations" FORCE)

# Disable other optional features
set(LLAMA_BUILD_TESTS OFF CACHE BOOL "Disable llama tests" FORCE)
set(LLAMA_BUILD_EXAMPLES OFF CACHE BOOL "Disable llama examples" FORCE)
set(LLAMA_CURL OFF CACHE BOOL "Disable curl support" FORCE)
set(GGML_OPENMP OFF CACHE BOOL "Disable OpenMP support" FORCE)

if(WIN32)
    set(BUILD_SHARED_LIBS ON CACHE BOOL "Build shared libraries" FORCE)
    set(GGML_BACKEND_DL ON CACHE BOOL "Enable backend DL" FORCE)
    set(GGML_CPU_ALL_VARIANTS ON CACHE BOOL "Enable all CPU variants for runtime dispatch" FORCE)
else()
    set(BUILD_SHARED_LIBS OFF CACHE BOOL "Build shared libraries" FORCE)
    set(GGML_BACKEND_DL OFF CACHE BOOL "Disable backend DL" FORCE)
    set(GGML_CPU_ALL_VARIANTS OFF CACHE BOOL "Disable all CPU variants for runtime dispatch" FORCE)
endif()

add_subdirectory(llama.cpp)

add_library(embedding_wrapper STATIC 
    src/llama_embeddings.cpp 
    src/llama_embeddings.h 
)

set_target_properties(embedding_wrapper PROPERTIES
        POSITION_INDEPENDENT_CODE ON
)

target_link_libraries(embedding_wrapper PRIVATE
        common llama ggml
)

target_compile_features(embedding_wrapper PRIVATE cxx_std_17)

target_include_directories(embedding_wrapper PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/src)

set_target_properties(embedding_wrapper PROPERTIES
        ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin
        LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin
        RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin
)
