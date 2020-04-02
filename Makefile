ROOT :=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

ifeq ($(GOBIN),)
	BIN := $(ROOT)/bin
else
	BIN := $(GOBIN)
endif

ifeq ($(OS),Windows_NT)
    EXE := .exe
endif

ARCHONSPEXE := $(BIN)/archonSP/archonSP$(EXE)
ARCHONEXE := $(BIN)/archon/archon$(EXE)

.PHONY: clean

all: $(ARCHONSPEXE) $(ARCHONEXE)

$(ARCHONSPEXE):
	cd $(ROOT)/cmd/archonSP && go build -o $(ARCHONSPEXE)

$(ARCHONEXE):
	cd $(ROOT)/cmd/archon && go build -o $(ARCHONEXE)

clean:
	rm $(ARCHONSPEXE) $(ARCHONEXE)
