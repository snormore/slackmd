# Default recipe
default: check

# Run all checks (build, test, vet)
check: build test vet

# Build all packages
build:
    go build ./...

# Run tests with race detection and coverage
test:
    go test -race -cover ./...

# Run go vet
vet:
    go vet ./...

# Run the CLI on an example file
example file="examples/long-form.md":
    go run ./cmd/slackmd {{file}}

# Post an example to Slack (requires SLACK_WEBHOOK_URL)
post file="examples/long-form.md":
    go run ./cmd/slackmd -post {{file}}

# Run the smoke test (requires SLACK_WEBHOOK_URL)
smoke:
    go run ./cmd/slackmd-smoke
