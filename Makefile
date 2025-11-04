.PHONY: all build clean lint static

all: build

build: lint
	@mkdir -p build
	@cd build && cmake .. && cmake --build .

static: lint
	@mkdir -p build
	@cd build && cmake -DBUILD_STATIC=ON .. && cmake --build .

clean:
	@rm -rf build
	@rm -f wikilite wikilite.exe

lint:
	@gofmt -w ./app
