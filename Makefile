# Diretório do binário
BINARY=bin/setup

# Diretório do código-fonte
SRC=.

.PHONY: all build clean

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) $(SRC)/main.go

clean:
	rm -rf bin
