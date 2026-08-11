SHELL := /bin/bash
GO ?= go
CC ?= gcc
CLANG ?= clang
BIN_DIR ?= bin
FUZZ_WORKERS ?= 4

.PHONY: all build test test-go test-go-fuzz test-c test-c-fuzz test-e2e test-snapshot demo demo-e2 demo-e3 demo-e3-worker demo-semantic-snapshot stress-e2 stress-e3-500 stress-semantic-snapshot stress-semantic-snapshot-scale clean docs-check fmt vet benchmark generate-e2 generate-e3 generate-e3-admission generate-e3-s1 force-go

C_VERIFIER_SOURCES := verifier/c/main.c verifier/c/observation.c verifier/c/sha256.c
C_VERIFIER_HEADERS := verifier/c/observation.h verifier/c/sha256.h
E2_C_VERIFIER_SOURCES := verifier/c/e2_main.c verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c
E2_C_ARTIFACT_SOURCES := verifier/c/e2_artifact_main.c verifier/c/e2.c
E2_C_HEADERS := verifier/c/e2.h verifier/c/observation.h verifier/c/sha256.h
DP_C_VERIFIER_SOURCES := verifier/c/dataplane_main.c verifier/c/dataplane.c
DP_C_HEADERS := verifier/c/dataplane.h
C_WARNINGS := -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic

all: build test

build: $(BIN_DIR)/tw $(BIN_DIR)/tw-test-origin $(BIN_DIR)/twirx-lab $(BIN_DIR)/twirx-stress $(BIN_DIR)/twirx-atlas $(BIN_DIR)/twirx-observer-worker $(BIN_DIR)/twirx-admission $(BIN_DIR)/twirx-egress-worker $(BIN_DIR)/twirx-snapshot $(BIN_DIR)/twirx-archive $(BIN_DIR)/twirx-archive-acquire $(BIN_DIR)/tw-verify-c $(BIN_DIR)/tw-verify-result-c $(BIN_DIR)/tw-verify-e2-artifact-c $(BIN_DIR)/tw-verify-data-plane-c

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

force-go:

$(BIN_DIR)/tw: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/tw

$(BIN_DIR)/tw-test-origin: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/tw-test-origin

$(BIN_DIR)/twirx-lab: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-lab

$(BIN_DIR)/twirx-stress: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-stress

$(BIN_DIR)/twirx-atlas: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-atlas

$(BIN_DIR)/twirx-observer-worker: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-observer-worker

$(BIN_DIR)/twirx-admission: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-admission

$(BIN_DIR)/twirx-egress-worker: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-egress-worker

$(BIN_DIR)/twirx-snapshot: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-snapshot

$(BIN_DIR)/twirx-archive: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-archive

$(BIN_DIR)/twirx-archive-acquire: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-archive-acquire

$(BIN_DIR)/tw-verify-c: $(C_VERIFIER_SOURCES) $(C_VERIFIER_HEADERS) | $(BIN_DIR)
	$(CC) -std=c2x -O2 $(C_WARNINGS) -o $@ $(C_VERIFIER_SOURCES)

$(BIN_DIR)/tw-verify-result-c: $(E2_C_VERIFIER_SOURCES) $(E2_C_HEADERS) | $(BIN_DIR)
	$(CC) -std=c2x -O2 $(C_WARNINGS) -o $@ $(E2_C_VERIFIER_SOURCES)

$(BIN_DIR)/tw-verify-e2-artifact-c: $(E2_C_ARTIFACT_SOURCES) verifier/c/e2.h | $(BIN_DIR)
	$(CC) -std=c2x -O2 $(C_WARNINGS) -o $@ $(E2_C_ARTIFACT_SOURCES)

$(BIN_DIR)/tw-verify-data-plane-c: $(DP_C_VERIFIER_SOURCES) $(DP_C_HEADERS) | $(BIN_DIR)
	$(CC) -std=c2x -O2 $(C_WARNINGS) -o $@ $(DP_C_VERIFIER_SOURCES)

test: test-go test-go-fuzz test-c test-c-fuzz test-e2e test-snapshot docs-check

test-go:
	$(GO) test ./...

test-go-fuzz: export GOMAXPROCS := $(FUZZ_WORKERS)
test-go-fuzz:
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalCBOR$$' -fuzztime=1s ./internal/observation
	$(GO) test -run='^$$' -fuzz='^FuzzJSONPointer$$' -fuzztime=1s ./internal/adapter
	$(GO) test -run='^$$' -fuzz='^FuzzDecodeManifest$$' -fuzztime=1s ./internal/adapter
	$(GO) test -run='^$$' -fuzz='^FuzzResultExtraction$$' -fuzztime=1s ./internal/adapter
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalResult$$' -fuzztime=1s ./internal/e2format
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalManifest$$' -fuzztime=1s ./internal/proofbundle
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalTransport$$' -fuzztime=1s ./internal/transportevidence
	$(GO) test -run='^$$' -fuzz='^FuzzSelectionJSON$$' -fuzztime=1s ./internal/atlas
	$(GO) test -run='^$$' -fuzz='^FuzzRegistryJSON$$' -fuzztime=1s ./internal/atlas
	$(GO) test -run='^$$' -fuzz='^FuzzPolicySetJSON$$' -fuzztime=1s ./internal/atlas
	$(GO) test -run='^$$' -fuzz='^FuzzParseAndEvaluate$$' -fuzztime=1s ./internal/robotstxt
	$(GO) test -run='^$$' -fuzz='^FuzzJobJSON$$' -fuzztime=1s ./internal/observatoryworker
	$(GO) test -run='^$$' -fuzz='^FuzzDecisionJSON$$' -fuzztime=1s ./internal/admission
	$(GO) test -run='^$$' -fuzz='^FuzzWorkOrderJSON$$' -fuzztime=1s ./internal/egressworker
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalDataPlane$$' -fuzztime=1s ./internal/dataplane
	$(GO) test -run='^$$' -fuzz='^FuzzPacketSegmentJSON$$' -fuzztime=1s ./internal/snapshotartifact
	$(GO) test -run='^$$' -fuzz='^FuzzProofIndexJSON$$' -fuzztime=1s ./internal/snapshotartifact
	$(GO) test -run='^$$' -fuzz='^FuzzQueryRequestJSON$$' -fuzztime=1s ./cmd/twirx-snapshot
	$(GO) test -run='^$$' -fuzz='^FuzzWorkOrderJSON$$' -fuzztime=1s ./internal/archiveimport
	$(GO) test -run='^$$' -fuzz='^FuzzIndexResponse$$' -fuzztime=1s ./internal/archiveimport
	$(GO) test -run='^$$' -fuzz='^FuzzCompressedWARC$$' -fuzztime=1s ./internal/archiveimport
	$(GO) test -run='^$$' -fuzz='^FuzzManifestJSON$$' -fuzztime=1s ./internal/archiveacquire
	$(GO) test -run='^$$' -fuzz='^FuzzExtractTitle$$' -fuzztime=1s ./internal/archiveprofile

test-c: $(BIN_DIR)/tw-verify-c $(BIN_DIR)/twirx-lab
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-c-sanitized $(C_VERIFIER_SOURCES)
	./scripts/test-c-verifier.sh $(BIN_DIR)/tw-verify-c-sanitized
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-result-c-sanitized $(E2_C_VERIFIER_SOURCES)
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-e2-artifact-c-sanitized $(E2_C_ARTIFACT_SOURCES)
	./scripts/test-c-e2.sh $(BIN_DIR)/tw-verify-result-c-sanitized $(BIN_DIR)/tw-verify-e2-artifact-c-sanitized $(BIN_DIR)/twirx-lab
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-data-plane-c-sanitized $(DP_C_VERIFIER_SOURCES)
	./scripts/test-c-dataplane.sh $(BIN_DIR)/tw-verify-data-plane-c-sanitized

test-c-fuzz: $(BIN_DIR)/twirx-lab | $(BIN_DIR)
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=fuzzer,address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-c-fuzz verifier/c/fuzz_observation.c verifier/c/observation.c
	./scripts/test-c-fuzz.sh $(BIN_DIR)/tw-verify-c-fuzz
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=fuzzer,address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-e2-c-fuzz verifier/c/fuzz_e2.c verifier/c/e2.c
	./scripts/test-c-e2-fuzz.sh $(BIN_DIR)/tw-verify-e2-c-fuzz $(BIN_DIR)/twirx-lab
	$(CLANG) -std=c2x -O1 -g $(C_WARNINGS) \
		-fsanitize=fuzzer,address,undefined -fno-omit-frame-pointer \
		-o $(BIN_DIR)/tw-verify-data-plane-c-fuzz verifier/c/fuzz_dataplane.c verifier/c/dataplane.c
	./scripts/test-c-dataplane-fuzz.sh $(BIN_DIR)/tw-verify-data-plane-c-fuzz

test-e2e: build
	./scripts/test-e2e.sh

test-snapshot: $(BIN_DIR)/twirx-snapshot $(BIN_DIR)/tw-verify-data-plane-c
	./scripts/test-semantic-snapshot.sh

demo: build
	./scripts/demo.sh

demo-e2: build
	./scripts/demo-e2.sh

demo-e3: build
	./scripts/demo-e3.sh

demo-e3-worker: build
	./scripts/demo-e3-worker.sh

demo-semantic-snapshot: build
	./scripts/demo-semantic-snapshot.sh

stress-e2: build
	./scripts/stress-e2.sh

stress-e3-500: build
	./scripts/stress-e3-500.sh

stress-semantic-snapshot: build
	./scripts/stress-semantic-snapshot.sh

stress-semantic-snapshot-scale: build
	TW_SNAPSHOT_SCALE_FIXTURE_PACKETS=25000 \
	TW_SNAPSHOT_STRESS_QUERY=examples/semantic-query-scale-fixture.json \
	TW_SNAPSHOT_INCLUDE_FIXTURES=1 \
	./scripts/stress-semantic-snapshot.sh

docs-check:
	./scripts/check-docs.sh

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

benchmark:
	$(GO) test -bench=. -benchmem ./internal/adapter ./internal/labengine ./internal/snapshotruntime

generate-e2: $(BIN_DIR)/twirx-lab
	$(BIN_DIR)/twirx-lab generate --root . --out generated/e2

generate-e3: $(BIN_DIR)/twirx-atlas
	$(BIN_DIR)/twirx-atlas metrics --root . --out generated/e3/atlas-metrics.json

generate-e3-admission: $(BIN_DIR)/twirx-admission
	$(BIN_DIR)/twirx-admission render --root . --admissions atlas/admissions --out generated/e3/admission

generate-e3-s1:
	$(GO) run ./cmd/twirx-s1-vectors -out conformance/e3-s1/vectors.tsv

clean:
	rm -rf $(BIN_DIR) var
