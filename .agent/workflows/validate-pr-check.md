---
description: How to validate the PR check workflow locally
---

# Validate PR Check Locally

You can validate the PR check workflow in two ways: manually running the commands or using `act` to simulate GitHub Actions.

## Option 1: Manual Validation (Fastest)

Run the following commands in your terminal to mimic the workflow steps:

1. **Validate Code**
   ```bash
   # Check formatting
   test -z $(gofmt -l .)
   
   # Run static analysis
   go vet ./...
   
   # Run tests
   go test -v ./...
   ```

2. **Verify Builds (Cross-compilation)**
   ```bash
   # Linux
   GOOS=linux GOARCH=amd64 go build -v ./cmd/witr
   GOOS=linux GOARCH=arm64 go build -v ./cmd/witr
   
   # macOS
   GOOS=darwin GOARCH=amd64 go build -v ./cmd/witr
   GOOS=darwin GOARCH=arm64 go build -v ./cmd/witr
   ```

## Option 2: Using `act` (Docker required)

If you have [act](https://github.com/nektos/act) installed, you can run the workflow in a container:

```bash
# Run the specific job
act -j validate
act -j build

# Or run the whole workflow for a pull_request event
act pull_request
```
