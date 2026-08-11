SHELL := /bin/bash
GO ?= go
OUT ?= dist
ADDR ?= :8080

.PHONY: all site serve check evidence verify-commits fmt vet clean

all: site

# Build and verify. The build exits non-zero if any check fails.
site:
	$(GO) run . -out $(OUT)

# Build, verify, then serve locally for review.
serve:
	$(GO) run . -out $(OUT) -serve $(ADDR)

# `site` already runs every check; this is an explicit alias for CI readability.
check: site verify-commits

# Regenerate the published proof artefact from repository artefacts.
# Requires `make demo` at the repository root first: the raw origin bytes are
# read from the content-addressed store that produces.
evidence:
	$(GO) run . -evidence

# The published commit identifiers must exist in this repository. A history
# rewrite changes every hash, and a public evidence page citing commits that no
# longer exist is worse than one citing none.
verify-commits:
	@set -euo pipefail; \
	fail=0; \
	for c in $$(python3 -c "import json;d=json.load(open('data/project-status.json'))['gate'];print(' '.join(d[k] for k in ('baseline_commit','implementation_commit','evidence_commit','merge_commit')))"); do \
	  if git -C .. cat-file -e "$$c^{commit}" 2>/dev/null; then \
	    echo "  ok    $$c"; \
	  else \
	    echo "  MISSING $$c — data/project-status.json cites a commit not in this repository"; \
	    fail=1; \
	  fi; \
	done; \
	exit $$fail

fmt:
	$(GO) fmt ./...
	test -z "$$(gofmt -l .)"

vet:
	$(GO) vet ./...

clean:
	rm -rf $(OUT)
