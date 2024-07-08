progname := metrics-server
builder = $(shell which podman 2>/dev/null|| which docker 2>/dev/null || echo "No builder found"; exit 1)

all: build

.PHONY: build
build:
	$(shell mkdir -p build)
	go build -o build/$(progname)

fmt:
	gofmt -s -w .

container:
	$(builder) build -t ${IMG} .

build-container: container

.PHONY: clean
clean:
	rm -rf build/