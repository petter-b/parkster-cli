# Parkster CLI

A command-line tool for managing Parkster parking sessions.

## Features

- **Secure credential storage** via OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- **OAuth2 browser flow** for services requiring it
- **JSON output** for scripting and AI agent integration
- **XDG-compliant** configuration (`~/.config/parkster/`)

## Install

```bash
# From source
go install github.com/yourorg/parkster/cmd/parkster@latest

# Or build locally
make build
./bin/parkster --help
```

## Quick Start

```bash
# Add API credentials
parkster auth add myservice
# Enter API key when prompted

# Or set via environment variable
export PARKSTER_MYSERVICE_API_KEY=sk-xxx
```

## Usage

```bash
# Show help
parkster --help

# List configured services
parkster auth list

# Check auth status
parkster auth status

# JSON output for scripting
parkster auth list --format json
```

## Configuration

Config file: `~/.config/parkster/config.yaml`

```yaml
output_format: plain  # plain, json, tsv
timeout: 30s
debug: false
default_account: myservice

services:
  myservice:
    base_url: https://api.example.com/v1
    timeout: 60s
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PARKSTER_FORMAT` | Default output format (plain/json/tsv) |
| `PARKSTER_DEBUG` | Enable debug output (1/true) |
| `PARKSTER_<SERVICE>_API_KEY` | API key for a service |

## Development

```bash
# Build
make build

# Test
make test

# Lint (requires golangci-lint)
make lint

# Format
make fmt

# Run with debug
make dev ARGS="auth list"
```

## Adding a New Service Integration

1. Create client: `internal/client/myservice.go`
2. Add commands: `internal/commands/myservice.go`
3. Update README with usage

See `CLAUDE.md` for detailed patterns and examples.

## License

MIT
