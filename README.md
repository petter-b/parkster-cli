# Parkster CLI

[![CI](https://github.com/petter-b/parkster-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/petter-b/parkster-cli/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/petter-b/parkster-cli/badge.svg)](https://codecov.io/gh/petter-b/parkster-cli)

A command-line tool for managing [Parkster](https://parkster.com) parking sessions. Start, stop, extend, and check parking status from your terminal.

## Install

```bash
# Build from source
make build
./bin/parkster --help

# Or install to $GOPATH/bin
make install
```

Requires Go 1.22+.

## Quick Start

```bash
# Store credentials in OS keychain
parkster auth login

# Start a 30-minute parking session
parkster start --zone 17429 --duration 30

# Check active parkings
parkster status

# Extend by 15 minutes
parkster extend --minutes 15

# Stop parking
parkster stop
```

## Commands

| Command | Description |
|---------|-------------|
| `parkster start` | Start a parking session |
| `parkster stop` | Stop an active parking session |
| `parkster extend` | Extend parking duration |
| `parkster status` | View active parking sessions |
| `parkster auth login` | Store credentials in OS keychain |
| `parkster auth logout` | Remove stored credentials |
| `parkster auth status` | Check authentication status |
| `parkster version` | Show version information |

### Start Parking

```bash
parkster start --zone 17429 --duration 30
parkster start --zone 17429 --duration 60 --car ABC123 --payment pay123
```

Flags: `--zone` (required), `--duration` (default: 30), `--car`, `--payment`

If you have a single car and payment method, they are auto-selected.

### Stop / Extend

```bash
parkster stop                        # auto-selects if only one active
parkster stop --parking-id 123456

parkster extend --minutes 15         # auto-selects if only one active
parkster extend --minutes 30 --parking-id 123456
```

## Authentication

Credentials are resolved in this order:

1. CLI flags: `--email` and `--password`
2. Environment variables: `PARKSTER_EMAIL` and `PARKSTER_PASSWORD`
3. OS keychain (stored via `parkster auth login`)

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PARKSTER_EMAIL` | Account email |
| `PARKSTER_PASSWORD` | Account password |
| `PARKSTER_DEBUG` | Enable debug output (`1` or `true`) |
| `PARKSTER_FORMAT` | Default output format (`plain`/`json`/`tsv`) |

## Output Formats

```bash
parkster status                  # human-readable (default)
parkster status --format json    # JSON output
parkster status --format tsv     # tab-separated values
```

## Development

```bash
make build          # Build binary
make test           # Run tests
make test-cover     # Run tests with coverage report
make lint           # Run linter (requires golangci-lint)
make fmt            # Format code
```

## License

MIT
