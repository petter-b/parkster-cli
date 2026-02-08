# mycli

A CLI tool template following patterns from [steipete's CLI ecosystem](https://github.com/steipete).

## Features

- **Secure credential storage** via OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- **OAuth2 browser flow** for services requiring it
- **JSON output** for scripting and AI agent integration
- **XDG-compliant** configuration (`~/.config/mycli/`)

## Install

```bash
# From source
go install github.com/yourorg/mycli/cmd/mycli@latest

# Or build locally
make build
./bin/mycli --help
```

## Quick Start

```bash
# Add API credentials
mycli auth add myservice
# Enter API key when prompted

# Or set via environment variable
export MYCLI_MYSERVICE_API_KEY=sk-xxx
```

## Usage

```bash
# Show help
mycli --help

# List configured services
mycli auth list

# Check auth status
mycli auth status

# JSON output for scripting
mycli auth list --format json
```

## Configuration

Config file: `~/.config/mycli/config.yaml`

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
| `MYCLI_FORMAT` | Default output format (plain/json/tsv) |
| `MYCLI_DEBUG` | Enable debug output (1/true) |
| `MYCLI_<SERVICE>_API_KEY` | API key for a service |

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
