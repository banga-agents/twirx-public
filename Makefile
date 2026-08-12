SHELL := /bin/bash
GO ?= go
CC ?= gcc
CLANG ?= clang
BIN_DIR ?= bin
FUZZ_WORKERS ?= 4
FUZZ_TIME ?= 3s

.PHONY: all build test test-go test-go-fuzz test-c test-c-fuzz test-e2e test-snapshot demo demo-e2 demo-e3 demo-e3-worker demo-e4-agent demo-e4-investigation demo-semantic-snapshot stress-e2 stress-e3-500 stress-semantic-snapshot stress-semantic-snapshot-scale clean docs-check fmt vet benchmark benchmark-e4-universe verify-e4-worldstate generate-e2 generate-e3 generate-e3-admission generate-e3-s1 generate-e4-ontology generate-e4-vectors force-go

C_VERIFIER_SOURCES := verifier/c/main.c verifier/c/observation.c verifier/c/sha256.c
C_VERIFIER_HEADERS := verifier/c/observation.h verifier/c/sha256.h
E2_C_VERIFIER_SOURCES := verifier/c/e2_main.c verifier/c/e2.c verifier/c/observation.c verifier/c/sha256.c
E2_C_ARTIFACT_SOURCES := verifier/c/e2_artifact_main.c verifier/c/e2.c
E2_C_HEADERS := verifier/c/e2.h verifier/c/observation.h verifier/c/sha256.h
DP_C_VERIFIER_SOURCES := verifier/c/dataplane_main.c verifier/c/dataplane.c
DP_C_HEADERS := verifier/c/dataplane.h
C_WARNINGS := -Wall -Wextra -Werror -Wconversion -Wshadow -Wpedantic

all: build test

build: $(BIN_DIR)/tw $(BIN_DIR)/tw-test-origin $(BIN_DIR)/twirx-lab $(BIN_DIR)/twirx-stress $(BIN_DIR)/twirx-atlas $(BIN_DIR)/twirx-observer-worker $(BIN_DIR)/twirx-admission $(BIN_DIR)/twirx-egress-worker $(BIN_DIR)/twirx-snapshot $(BIN_DIR)/twirx-archive $(BIN_DIR)/twirx-archive-acquire $(BIN_DIR)/twirx-ontology $(BIN_DIR)/twirx-e4-agent $(BIN_DIR)/twirx-e4-worldstate $(BIN_DIR)/twirx-e4-opportunity $(BIN_DIR)/twirx-e4-capacity $(BIN_DIR)/tw-verify-c $(BIN_DIR)/tw-verify-result-c $(BIN_DIR)/tw-verify-e2-artifact-c $(BIN_DIR)/tw-verify-data-plane-c

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

$(BIN_DIR)/twirx-ontology: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-ontology

$(BIN_DIR)/twirx-e4-agent: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-e4-agent

$(BIN_DIR)/twirx-e4-worldstate: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-e4-worldstate

$(BIN_DIR)/twirx-e4-opportunity: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-e4-opportunity

$(BIN_DIR)/twirx-e4-capacity: force-go | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/twirx-e4-capacity

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
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalCBOR$$' -fuzztime=$(FUZZ_TIME) ./internal/observation
	$(GO) test -run='^$$' -fuzz='^FuzzJSONPointer$$' -fuzztime=$(FUZZ_TIME) ./internal/adapter
	$(GO) test -run='^$$' -fuzz='^FuzzDecodeManifest$$' -fuzztime=$(FUZZ_TIME) ./internal/adapter
	$(GO) test -run='^$$' -fuzz='^FuzzResultExtraction$$' -fuzztime=$(FUZZ_TIME) ./internal/adapter
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalResult$$' -fuzztime=$(FUZZ_TIME) ./internal/e2format
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalManifest$$' -fuzztime=$(FUZZ_TIME) ./internal/proofbundle
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalTransport$$' -fuzztime=$(FUZZ_TIME) ./internal/transportevidence
	$(GO) test -run='^$$' -fuzz='^FuzzSelectionJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/atlas
	$(GO) test -run='^$$' -fuzz='^FuzzRegistryJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/atlas
	$(GO) test -run='^$$' -fuzz='^FuzzPolicySetJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/atlas
	$(GO) test -run='^$$' -fuzz='^FuzzParseAndEvaluate$$' -fuzztime=$(FUZZ_TIME) ./internal/robotstxt
	$(GO) test -run='^$$' -fuzz='^FuzzJobJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/observatoryworker
	$(GO) test -run='^$$' -fuzz='^FuzzDecisionJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/admission
	$(GO) test -run='^$$' -fuzz='^FuzzWorkOrderJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/egressworker
	$(GO) test -run='^$$' -fuzz='^FuzzUnmarshalDataPlane$$' -fuzztime=$(FUZZ_TIME) ./internal/dataplane
	$(GO) test -run='^$$' -fuzz='^FuzzPacketSegmentJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/snapshotartifact
	$(GO) test -run='^$$' -fuzz='^FuzzProofIndexJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/snapshotartifact
	$(GO) test -run='^$$' -fuzz='^FuzzQueryRequestJSON$$' -fuzztime=$(FUZZ_TIME) ./cmd/twirx-snapshot
	$(GO) test -run='^$$' -fuzz='^FuzzWorkOrderJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/archiveimport
	$(GO) test -run='^$$' -fuzz='^FuzzIndexResponse$$' -fuzztime=$(FUZZ_TIME) ./internal/archiveimport
	$(GO) test -run='^$$' -fuzz='^FuzzCompressedWARC$$' -fuzztime=$(FUZZ_TIME) ./internal/archiveimport
	$(GO) test -run='^$$' -fuzz='^FuzzManifestJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/archiveacquire
	$(GO) test -run='^$$' -fuzz='^FuzzExtractTitle$$' -fuzztime=$(FUZZ_TIME) ./internal/archiveprofile
	$(GO) test -run='^$$' -fuzz='^FuzzModuleSource$$' -fuzztime=$(FUZZ_TIME) ./internal/ontologyfabric
	$(GO) test -run='^$$' -fuzz='^FuzzCompileWorldBank$$' -fuzztime=$(FUZZ_TIME) ./internal/universeimport
	$(GO) test -run='^$$' -fuzz='^FuzzCompileGrantsFetch$$' -fuzztime=$(FUZZ_TIME) ./internal/universeimport
	$(GO) test -run='^$$' -fuzz='^FuzzCompileGrantsBulkProjection$$' -fuzztime=$(FUZZ_TIME) ./internal/universeimport
	$(GO) test -run='^$$' -fuzz='^FuzzProjectGrantsXML$$' -fuzztime=$(FUZZ_TIME) ./internal/opportunitypilot
	$(GO) test -run='^$$' -fuzz='^FuzzOpenNative$$' -fuzztime=$(FUZZ_TIME) ./internal/universesnapshot
	$(GO) test -run='^$$' -fuzz='^FuzzOpenColumnar$$' -fuzztime=$(FUZZ_TIME) ./internal/universesnapshot
	$(GO) test -run='^$$' -fuzz='^FuzzOpenCompact$$' -fuzztime=$(FUZZ_TIME) ./internal/universesnapshot
	$(GO) test -run='^$$' -fuzz='^FuzzPlanJSON$$' -fuzztime=$(FUZZ_TIME) ./internal/worldstatepilot

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
	./scripts/test-c-e4-ontology.sh $(BIN_DIR)/tw-verify-data-plane-c-sanitized
	./scripts/test-c-e4-worldstate-release.sh $(BIN_DIR)/tw-verify-data-plane-c-sanitized
	./scripts/test-c-e4-opportunity-sample.sh $(BIN_DIR)/tw-verify-data-plane-c-sanitized generated/e4/releases/grants-gov-20260811-c-sample

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

demo-e4-agent: $(BIN_DIR)/twirx-e4-agent
	$(BIN_DIR)/twirx-e4-agent --root . --scenario world-state.controlled-development

demo-e4-investigation: $(BIN_DIR)/twirx-e4-agent
	$(BIN_DIR)/twirx-e4-agent --root . \
		--investigation utility.controlled-world-and-opportunity

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

benchmark-e4-universe:
	$(GO) test ./internal/universesnapshot -run='^$$' \
		-bench='Benchmark(Native|Columnar|Compact)ExactSlotQuery' \
		-benchmem -benchtime=20x -count=3

verify-e4-worldstate: $(BIN_DIR)/twirx-e4-worldstate
	$(BIN_DIR)/twirx-e4-worldstate verify-release --root . \
		--release generated/e4/releases/world-bank-e2-matrix

generate-e2: $(BIN_DIR)/twirx-lab
	$(BIN_DIR)/twirx-lab generate --root . --out generated/e2

generate-e3: $(BIN_DIR)/twirx-atlas
	$(BIN_DIR)/twirx-atlas metrics --root . --out generated/e3/atlas-metrics.json

generate-e3-admission: $(BIN_DIR)/twirx-admission
	$(BIN_DIR)/twirx-admission render --root . --admissions atlas/admissions --out generated/e3/admission

generate-e3-s1:
	$(GO) run ./cmd/twirx-s1-vectors -out conformance/e3-s1/vectors.tsv

generate-e4-ontology: $(BIN_DIR)/twirx-ontology
	$(BIN_DIR)/twirx-ontology compile --root . --out generated/e4/ontology

generate-e4-vectors:
	$(GO) run ./cmd/twirx-e4-vectors -out conformance/e4-ontology/vectors.tsv

clean:
	rm -rf $(BIN_DIR) var
