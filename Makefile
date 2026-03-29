BINARY  := fzcd
BUILD   := ./build
CMD     := .

GOFLAGS := -ldflags="-s -w"

.PHONY: all build clean install

all: build

build:
	@mkdir -p $(BUILD)
	go mod tidy
	go build $(GOFLAGS) -o $(BUILD)/$(BINARY) $(CMD)
	@echo "Built $(BUILD)/$(BINARY)"

clean:
	rm -rf $(BUILD)

install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(BUILD)/$(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed to $(HOME)/.local/bin/$(BINARY)"
