# Load Tests

## Prerequisites

Install [k6](https://k6.io/docs/getting-started/installation/):

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D68
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

## Running Tests

```bash
# Run with default settings (localhost:8080)
k6 run loadtests/load.js

# Run against a custom base URL
BASE_URL=http://staging.needly.app k6 run loadtests/load.js

# Run with JSON output for CI
k6 run --out json=loadtests/results.json loadtests/load.js

# Run with summary
k6 run --summary-export=loadtests/summary.json loadtests/load.js
```

## Thresholds

- p95 response time must be under 500ms
- Error rate must be below 10%
