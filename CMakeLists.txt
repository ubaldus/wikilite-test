cmake_minimum_required(VERSION 3.14)
project(wikilite C CXX)

option(BUILD_STATIC "Force static linking on Linux" OFF)

include(FetchContent)

set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)

find_program(GO_EXECUTABLE go)
if(NOT GO_EXECUTABLE)
    message(FATAL_ERROR "Go compiler not found. Please install Go or add it to your PATH.")
endif()

set(GO_BUILD_TAGS "fts5")
set(GO_LDFLAGS "-s -w")
set(EXT_LDFLAGS "")

set(LOCAL_EMBEDDINGS_SUPPORTED OFF)
if(CMAKE_SYSTEM_NAME MATCHES "Darwin")
    set(LOCAL_EMBEDDINGS_SUPPORTED ON)
elseif(CMAKE_SYSTEM_NAME MATCHES "Linux")
    set(LOCAL_EMBEDDINGS_SUPPORTED ON)
elseif(CMAKE_SYSTEM_NAME MATCHES "Windows")
    set(LOCAL_EMBEDDINGS_SUPPORTED ON)
endif()

if(LOCAL_EMBEDDINGS_SUPPORTED)
    message(STATUS "Local embeddings support is enabled.")
    set(CMAKE_POSITION_INDEPENDENT_CODE ON)

    set(LLAMA_CPP_PATCHED_SOURCE_DIR ${CMAKE_BINARY_DIR}/llama_cpp-src)

    if(UNIX)
        find_program(PATCH_EXECUTABLE patch REQUIRED)
        
        if(NOT EXISTS "${LLAMA_CPP_PATCHED_SOURCE_DIR}/CMakeLists.txt")
            message(STATUS "Preparing llama.cpp...")
            
            execute_process(
                COMMAND ${CMAKE_COMMAND} -E copy_directory 
                        "${CMAKE_CURRENT_SOURCE_DIR}/llama.cpp" 
                        "${LLAMA_CPP_PATCHED_SOURCE_DIR}"
                RESULT_VARIABLE copy_res
                TIMEOUT 120
            )
            
            if(NOT copy_res EQUAL 0)
                message(FATAL_ERROR "Failed to copy llama.cpp directory (error: ${copy_res})")
            endif()
            message(STATUS "Copy completed successfully.")
            
            if(NOT EXISTS "${LLAMA_CPP_PATCHED_SOURCE_DIR}/CMakeLists.txt")
                message(FATAL_ERROR "Copy failed - CMakeLists.txt not found in patched directory")
            endif()
            
            message(STATUS "Applying patch to llama.cpp...")
            
            if(NOT EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/src/llama_cpp.patch")
                message(FATAL_ERROR "Patch file not found at: ${CMAKE_CURRENT_SOURCE_DIR}/src/llama_cpp.patch")
            endif()
            
            execute_process(
                COMMAND ${PATCH_EXECUTABLE} -p2 --batch --forward -i "${CMAKE_CURRENT_SOURCE_DIR}/src/llama_cpp.patch"
                WORKING_DIRECTORY "${LLAMA_CPP_PATCHED_SOURCE_DIR}"
                RESULT_VARIABLE patch_res
                OUTPUT_VARIABLE patch_out
                ERROR_VARIABLE patch_err
                TIMEOUT 30
            )
            
            if(patch_res EQUAL 0)
                message(STATUS "Patch applied successfully")
            else()
                message(FATAL_ERROR "Failed to patch llama.cpp.\nReturn code: ${patch_res}\nOutput: ${patch_out}\nError: ${patch_err}")
            endif()
        else()
            message(STATUS "Using existing patched llama.cpp from previous build")
        endif()

        FetchContent_Declare(
            llama_cpp
            SOURCE_DIR ${LLAMA_CPP_PATCHED_SOURCE_DIR}
        )
    else()
        FetchContent_Declare(
            llama_cpp
            SOURCE_DIR ${CMAKE_CURRENT_SOURCE_DIR}/llama.cpp
        )
    endif()

    set(LLAMA_BUILD_COMMON ON)
    set(LLAMA_CUBLAS OFF)
    set(LLAMA_HIPBLAS OFF)
    set(LLAMA_CLBLAST OFF)
    set(LLAMA_METAL OFF)
    set(LLAMA_VULKAN OFF)
    set(GGML_BLAS OFF)
    set(GGML_METAL OFF)
    set(GGML_CUDA OFF)
    set(GGML_NATIVE OFF)
    set(LLAMA_BUILD_TESTS OFF)
    set(LLAMA_BUILD_EXAMPLES OFF)
    set(LLAMA_CURL OFF)
    set(GGML_OPENMP OFF)
    
    if(WIN32)
        set(BUILD_SHARED_LIBS ON)
        set(GGML_BACKEND_DL ON)
        set(GGML_CPU_ALL_VARIANTS ON)
    else()
        set(BUILD_SHARED_LIBS OFF)
        set(GGML_BACKEND_DL OFF)
        set(GGML_CPU_ALL_VARIANTS OFF)
    endif()

    FetchContent_MakeAvailable(llama_cpp)

    add_library(embedding_wrapper STATIC
        src/llama_embeddings.cpp
        src/llama_embeddings.h
    )
    set_target_properties(embedding_wrapper PROPERTIES POSITION_INDEPENDENT_CODE ON)
    target_link_libraries(embedding_wrapper PRIVATE common llama ggml)
    target_compile_features(embedding_wrapper PRIVATE cxx_std_17)
    target_include_directories(embedding_wrapper PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/src)
    set_property(TARGET embedding_wrapper PROPERTY ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)

    string(APPEND GO_BUILD_TAGS " aiInternal")
    set(EXT_LDFLAGS " -L${CMAKE_BINARY_DIR}/bin")
    
    if (CMAKE_SYSTEM_NAME MATCHES "Linux" AND BUILD_STATIC)
        string(APPEND EXT_LDFLAGS " -static -lstdc++")
    endif()
    
    if(APPLE)
        string(APPEND EXT_LDFLAGS " -lc++ -framework Accelerate -framework CoreFoundation -framework Security")
    endif()
    
    string(APPEND EXT_LDFLAGS " -lembedding_wrapper -lcommon -lllama -lggml -lm")
else()
    message(STATUS "Local embeddings support is disabled.")
endif()

if(WIN32)
    set(TARGET_NAME "wikilite.exe")
else()
    set(TARGET_NAME "wikilite")
endif()

set(LDFLAGS_CONTENT "${GO_LDFLAGS}")
if(EXT_LDFLAGS)
  string(APPEND LDFLAGS_CONTENT " -extldflags '${EXT_LDFLAGS}'")
endif()

file(GLOB_RECURSE GO_FILES "app/*.go")

add_custom_command(
    OUTPUT ${CMAKE_BINARY_DIR}/bin/${TARGET_NAME}
    COMMAND ${GO_EXECUTABLE} build
            -v -tags "${GO_BUILD_TAGS}"
            -ldflags "${LDFLAGS_CONTENT}"
            -o ${CMAKE_BINARY_DIR}/bin/${TARGET_NAME}
            ./app
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    DEPENDS ${GO_FILES}
    VERBATIM
)

add_custom_target(wikilite ALL DEPENDS ${CMAKE_BINARY_DIR}/bin/${TARGET_NAME})
if(LOCAL_EMBEDDINGS_SUPPORTED)
    add_dependencies(wikilite embedding_wrapper)
endif()
