# Parkster CLI

[![CI](https://github.com/petter-b/parkster-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/petter-b/parkster-cli/actions/workflows/ci.yml)

A command-line tool for managing [Parkster](https://parkster.com) parking sessions. Start, stop, change, and check parking status from your terminal.

## Install

### Homebrew (macOS/Linux)

```bash
brew install petter-b/tap/parkster
```

### Download binary

Download the latest release from [GitHub Releases](https://github.com/petter-b/parkster-cli/releases).

### Build from source

Requires Go 1.24+.

```bash
git clone https://github.com/petter-b/parkster-cli.git
cd parkster-cli
make build
./bin/parkster --help
```

## Quick Start

```bash
# Store credentials in OS keychain
parkster auth login

# Start a 30-minute parking session
parkster start --zone 80500 --duration 30 --lat 59.373 --lon 17.893

# Check active parkings
parkster status

# Change end time
parkster change --duration 15

# Stop parking
parkster stop
```

## Commands

| Command | Description |
|---------|-------------|
| `parkster start` | Start a parking session |
| `parkster stop` | Stop an active parking session |
| `parkster change` | Change parking end time |
| `parkster status` | View active parking sessions |
| `parkster profile` | Show account info, cars, payments, and favorite zones |
| `parkster zones search` | Search for zones near GPS coordinates |
| `parkster zones info` | Show details for a zone by sign code |
| `parkster auth login` | Store credentials in OS keychain |
| `parkster auth logout` | Remove stored credentials |
| `parkster auth status` | Check authentication status |
| `parkster version` | Show version information |
| `parkster completion` | Generate shell completion scripts |

### Start Parking

```bash
parkster start --zone 80500 --duration 30 --lat 59.373 --lon 17.893
parkster start --zone 80500 --duration 60 --lat 59.373 --lon 17.893 --car ABC123
```

Flags: `--zone` (required), `--duration` or `--until`, `--car`, `--payment`, `--dry-run`, `--lat`, `--lon`, `--radius`

If you have a single car and payment method, they are auto-selected.

### Stop / Change

```bash
parkster stop                        # auto-stops if only one active
parkster stop --parking-id 123456

parkster change --duration 60         # set end time to 60 min from now
parkster change --until 18:30         # set end time (also accepts HH.MM or HH)
parkster change --duration 60 --parking-id 123456
```

## Authentication

Credentials are resolved in this order:

1. OS keychain (stored via `parkster auth login`)
2. Plaintext file (`~/.config/parkster/credentials.json`)
3. Environment variables: `PARKSTER_USERNAME` and `PARKSTER_PASSWORD`

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PARKSTER_USERNAME` | Account username (email or phone number) |
| `PARKSTER_PASSWORD` | Account password |
| `PARKSTER_DEBUG` | Enable debug output (`1` or `true`) |

## Output Formats

```bash
parkster status                  # human-readable (default)
parkster status --json           # JSON with envelope
parkster status --quiet          # suppress status messages
parkster status --debug          # show debug output on stderr
```

## Uninstall

```bash
# Remove the binary
rm $(which parkster)

# Remove stored credentials
parkster auth logout

# Remove config directory (optional)
rm -rf ~/.config/parkster
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
