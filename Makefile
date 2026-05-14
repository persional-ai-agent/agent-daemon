SHELL := /bin/bash

.PHONY: all clean test contract-test contract-replay contract-ws-replay contract-coverage contract-diff diag-check contract-check contract-release

all:
	go build -o agent-daemon cmd/agentd/main.go 

clean:
	@rm -f ./agent-daemon

test:
	go test ./...

contract-test:
	go test ./internal/api ./internal/cli ./ui-tui

contract-replay:
	CONTRACT_REPLAY_REPORT="$(PWD)/artifacts/contract-replay.json" go test ./internal/api -run TestContractReplay -count=1

contract-ws-replay:
	CONTRACT_WS_REPLAY_REPORT="$(PWD)/artifacts/contract-ws-replay.json" go test ./internal/api -run TestContractWSReplay -count=1

contract-coverage:
	go run ./scripts/contract_coverage \
		-openapi docs/api/ui-chat-contract.openapi.yaml \
		-replay internal/api/testdata/replay/cases.json \
		-ws-schema docs/api/ws-chat-events.schema.json \
		-ws-replay internal/api/testdata/replay/ws_cases.json \
		-report artifacts/contract-coverage.json \
		-enforce-core=true

contract-diff:
	go run ./scripts/contract_diff.go \
		-base docs/api/versions/v1/ui-chat-contract.openapi.yaml \
		-target docs/api/ui-chat-contract.openapi.yaml \
		-report artifacts/contract-diff.json

diag-check:
	go test ./scripts/diag_bundle -count=1

contract-check: contract-test contract-replay contract-ws-replay contract-coverage contract-diff diag-check

contract-release: contract-check
	mkdir -p docs/api/versions/v1
	cp docs/api/ui-chat-contract.openapi.yaml docs/api/versions/v1/ui-chat-contract.openapi.yaml
	cp docs/api/contract-versioning.md docs/api/versions/v1/contract-versioning.md
