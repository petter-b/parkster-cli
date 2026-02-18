# Parkster CLI Development Guide

## Project Configuration

- **GitHub Username**: `petter-b`
- **Module Path**: `github.com/petter-b/parkster-cli`
- **Repository**: Private (planned to be made public later)
- **Binary Name**: `parkster`

## Patterns Reference

**For CLI patterns, testing, auth, and command checklists**, use the `designing-cli-commands` skill.
It covers: command scaffolding, deps.go dependency injection, error handling (errSilent pattern), selection logic (auto-select when unambiguous), HTTP Basic Auth (Pattern C), TDD workflows, testing with httptest, credential management, and Cobra command checklists.

This file contains **Parkster-specific** implementation details only.

## Quick Reference

- **API Documentation**: See [API.md](./API.md) for complete Parkster API reference
- **Design Document**: See [docs/plans/2026-02-08-parkster-cli-mvp-design.md](./docs/plans/2026-02-08-parkster-cli-mvp-design.md)

## Testing

**Use the `superpowers:test-driven-development` skill for all new features.**
Test files live in `*_test.go` alongside implementation.
Use build tags (`//go:build integration`) to separate unit and integration tests.

```bash
make test          # Run all tests
make test-cover    # Run with coverage (opens browser)
make test-integration  # Integration tests (sources .env)
```

See [TESTING.md](./TESTING.md) for the Python test suite that validates the API.

**Test zones:**

| Country | Zone ID | Location | Rate |
|---------|---------|----------|------|
| Sweden | 17429 | Ericsson Kista, Stockholm | 10 SEK/hour |
| Germany | 7713 | Berlin (code 100028) | 3 EUR/hour |
| Austria | 25624 | Salzburg (code 006001) | 2.20 EUR/hour |

## Parkster API Specifics

These are **Parkster-specific quirks** that differ from typical REST APIs:

1. **Form-Encoded POST/PUT (NOT JSON)** — All mutations use `application/x-www-form-urlencoded`. The API rejects JSON bodies.

2. **Device Parameters Required on EVERY Request** — Must mimic iOS app (`platform=ios`, `platformVersion=26.2`, `version=626`, `locale=en_US`, `clientTime=<unix_ms>`). API rejects custom platform identifiers. For GET: query string. For POST: form body.

3. **Fee Zone ID Required to Start Parking** — Zone search doesn't include `feeZoneId`. Must fetch zone details via `GetZone()` first, then use `details.FeeZone.ID` when starting.

4. **Extend Parking Uses "offset" Not "timeout"** — The `offset` parameter *adds* minutes. Using `timeout` would set the absolute timeout.

See `internal/parkster/client.go` for implementation.

## HTTP Basic Auth

Uses Pattern C from the `designing-cli-commands` skill.
Credential priority: keyring > file > env vars.
See `internal/auth/` for implementation.

## API Client Methods

See `internal/parkster/client.go` for full signatures.

- `Login() (*User, error)`
- `GetActiveParkings() ([]Parking, error)`
- `GetZone(zoneID int) (*Zone, error)`
- `StartParking(zoneID, feeZoneID, carID int, paymentID string, timeout int) (*Parking, error)`
- `StopParking(parkingID int) (*Parking, error)`
- `ExtendParking(parkingID, minutes int) (*Parking, error)`

## Data Types

See `internal/parkster/types.go` for all struct definitions (User, Car, PaymentAccount, Zone, FeeZone, Currency, Parking).

## Authentication

```bash
# Store credentials
parkster auth login
# Prompts for email, then password

# Or use environment variables
export PARKSTER_USERNAME=user@example.com
export PARKSTER_PASSWORD=password123

# Or use flags
parkster start --email user@example.com --password secret --zone 17429 --duration 30
```

## Multi-Country Support

Parkster operates in multiple countries with the same API:

| Country | Code | Currency | Test Zone |
|---------|------|----------|-----------|
| Sweden | SE | SEK (kr) | 17429 (Ericsson Kista, Stockholm) |
| Germany | DE | EUR | 7713 (Berlin, code 100028) |
| Austria | AT | EUR | 25624 (Salzburg, code 006001) |

**License plate formats:**
- SE: `ABC123` (3 letters + 3 digits)
- DE: `B-AB-1234` (city code format)
- AT: `W-12345` (city code format)

## Command Examples

### Start Parking

```bash
# Basic usage
parkster start --zone 17429 --duration 30

# With explicit car selection
parkster start --zone 17429 --duration 30 --car ABC123

# Full explicit mode (good for testing/CI)
parkster start \
  --email user@example.com \
  --password secret \
  --zone 17429 \
  --duration 30 \
  --car ABC123 \
  --payment pay123 \
  --json
```

See [Start Parking Workflow](#start-parking-workflow) below for the detailed API call trace.

### Stop Parking

```bash
# Auto-stop if only one active
parkster stop

# Explicit parking ID
parkster stop --parking-id 123456
```

### Extend Parking

```bash
# Add 30 minutes (auto if only one active)
parkster extend --minutes 30

# Explicit parking ID
parkster extend --minutes 30 --parking-id 123456
```

### Status

```bash
# View active parkings
parkster status

# JSON output for AI agents
parkster status --json
```

## Start Parking Workflow

```
User: parkster start --zone 17429 --duration 30

1. auth.GetEmail() → "user@example.com" (keyring > file > env)
2. auth.GetPassword() → "password123" (keyring > file > env)
3. client = parkster.NewClient(email, password)
4. user = client.Login()
   → GET /people/login (with device params + Basic Auth)
   → Returns: {cars: [{id: 67890, licenseNbr: "ABC123"}], ...}
5. Select car: only one → use it
6. Select payment: user.PaymentAccounts[0]
7. zone = client.GetZone(17429)
   → GET /parking-zones/17429
   → Returns: {id: 17429, feeZone: {id: 27545}}
8. parking = client.StartParking(17429, 27545, 67890, paymentID, 30)
   → POST /parkings/short-term (form-encoded + device params in body!)
   → Returns: {id: 123456, status: "ACTIVE", cost: 0.0}
9. output.PrintSuccess(parking, OutputMode())
```

## Go Idioms (Learned)

- **Auth headers:** Use `req.SetBasicAuth(user, pass)` — never manual `base64.StdEncoding.EncodeToString`
- **Parameter shadowing:** Use `format` not `fmt_` when a parameter would shadow the `fmt` package
- **Sentinel errors:** Give descriptive messages even if the error is caught before printing: `errors.New("silent error: already printed")` not `errors.New("")`
- **Error wrapping:** Always `fmt.Errorf("context: %w", err)` — bare `return err` loses context
- **Document asymmetries:** If GET retries but POST doesn't, add a comment explaining why
- **Parent commands:** Don't manually handle "no args → help" for Cobra commands with subcommands — Cobra does this automatically
- **DRY HTTP methods:** When POST/PUT share the same body, extract a shared `mutate(method, path, data)` helper

## Remember

1. **Use `designing-cli-commands` skill** for CLI patterns, testing, and auth
2. **Parkster quirks**:
   - Form-encoded POST (not JSON)
   - Device params everywhere
   - Fee zone ID required
   - Extend uses offset
3. **See API.md** — complete API reference with examples
4. **KISS/YAGNI** — don't add features until needed
5. **Output flags**: Use `--json` and `-q`/`--quiet`, not `--format`
