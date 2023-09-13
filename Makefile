GOCC=go
MKDIR_P=mkdir -p
GIT_SUBM=git submodule

BIN_PATH=./build
BIN="./build/looking-up-ipfs"

.PHONY: build install dependencies  clean

build:
	$(GOCC) build -o $(BIN)


install:
	$(GOCC) install


dependencies:
	$(GIT_SUBM) update --init
	cd go-libp2p-kad-dht && git checkout origin/cid-hoarder


clean:
	rm -r $(BIN_PATH)

