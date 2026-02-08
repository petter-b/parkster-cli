# Parkster CLI Development Guide

## Project Configuration

- **GitHub Username**: `petter-b`
- **Module Path**: `github.com/petter-b/parkster-cli`
- **Repository**: Private (planned to be made public later)
- **Binary Name**: `parkster`

**This project uses the CLI template from `../cli-template`. Read that file first for:**
- General CLI patterns (commands, errors, output, auth)
- KISS/YAGNI principles
- AI agent friendliness
- Testing patterns

This file contains **Parkster-specific** implementation details only.

## Quick Reference

- **API Documentation**: See [API.md](./API.md) for complete Parkster API reference
- **Design Document**: See [docs/plans/2026-02-08-parkster-cli-mvp-design.md](./docs/plans/2026-02-08-parkster-cli-mvp-design.md)
- **Template Guide**: See [../cli-template/CLAUDE.md](../cli-template/CLAUDE.md)

## Parkster API Specifics

### Critical Implementation Notes

These are **Parkster-specific quirks** that differ from typical REST APIs:

#### 1. Form-Encoded POST/PUT (NOT JSON)

All mutations use `application/x-www-form-urlencoded`, **not JSON**:

```go
// ✅ CORRECT - Form-encoded
data := url.Values{}
data.Set("parkingZoneId", "7713")
data.Set("timeout", "30")
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
req.Body = strings.NewReader(data.Encode())

// ❌ WRONG - JSON doesn't work!
json.Marshal(payload)  // Parkster API rejects this
```

#### 2. Device Parameters Required on EVERY Request

```go
// Required on ALL requests (GET and POST)
platform=cli
platformVersion=1.0
version=1
locale=en_US
clientTime=<unix_timestamp_ms>

// For GET: add to query string
url := fmt.Sprintf("%s?platform=cli&clientTime=%d...", endpoint, time.Now().UnixMilli())

// For POST: ALSO add to form body
data := url.Values{}
data.Set("parkingZoneId", "123")
data.Set("platform", "cli")        // Duplicate in body!
data.Set("clientTime", fmt.Sprintf("%d", time.Now().UnixMilli()))
```

**Implementation:**
```go
// internal/parkster/client.go
func (c *Client) deviceParams() url.Values {
    params := url.Values{}
    params.Set("platform", "cli")
    params.Set("platformVersion", "1.0")
    params.Set("version", "1")
    params.Set("locale", "en_US")
    params.Set("clientTime", fmt.Sprintf("%d", time.Now().UnixMilli()))
    return params
}
```

#### 3. Fee Zone ID Required to Start Parking

Zone search returns basic info. **Must fetch zone details separately** to get `feeZoneId`:

```go
// ❌ WRONG - zone search doesn't include feeZoneId
searchResp := client.SearchZones(lat, lon)
client.StartParking(searchResp[0].ID, ...)  // Missing feeZoneId!

// ✅ CORRECT - fetch details first
searchResp := client.SearchZones(lat, lon)
details := client.GetZone(searchResp[0].ID)
client.StartParking(details.ID, details.FeeZone.ID, ...)  // Now have both IDs
```

#### 4. Extend Parking Uses "offset" Not "timeout"

```go
// ❌ WRONG - timeout would SET absolute timeout
data.Set("timeout", "60")  // Would set timeout to 60 minutes total

// ✅ CORRECT - offset ADDS minutes
data.Set("offset", "30")   // Adds 30 more minutes
```

## HTTP Basic Auth

Parkster uses HTTP Basic Auth with email/password:

```go
// internal/parkster/client.go
func (c *Client) get(path string, params url.Values) (*http.Response, error) {
    // Merge device params
    for k, v := range c.deviceParams() {
        params[k] = v
    }

    url := fmt.Sprintf("%s%s?%s", BaseURL, path, params.Encode())
    req, _ := http.NewRequest("GET", url, nil)

    // Basic Auth header
    auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.password))
    req.Header.Set("Authorization", "Basic "+auth)
    req.Header.Set("Accept", "application/json")

    return c.http.Do(req)
}

func (c *Client) post(path string, data url.Values) (*http.Response, error) {
    // Merge device params into BODY (not just query string!)
    for k, v := range c.deviceParams() {
        data[k] = v
    }

    url := fmt.Sprintf("%s%s", BaseURL, path)
    req, _ := http.NewRequest("POST", url, strings.NewReader(data.Encode()))

    // Basic Auth header
    auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.password))
    req.Header.Set("Authorization", "Basic "+auth)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    return c.http.Do(req)
}
```

## API Client Structure

```go
// internal/parkster/client.go
package parkster

const BaseURL = "https://api.parkster.se/api/mobile/v2"

type Client struct {
    http     *http.Client
    email    string
    password string
}

func NewClient(email, password string) *Client {
    return &Client{
        http:     &http.Client{Timeout: 30 * time.Second},
        email:    email,
        password: password,
    }
}

// API methods (MVP)
func (c *Client) Login() (*User, error)
func (c *Client) GetActiveParkings() ([]Parking, error)
func (c *Client) GetZone(zoneID int) (*Zone, error)
func (c *Client) StartParking(zoneID, feeZoneID, carID int, paymentID string, timeout int) (*Parking, error)
func (c *Client) StopParking(parkingID int) (*Parking, error)
func (c *Client) ExtendParking(parkingID, minutes int) (*Parking, error)
```

## Data Types

```go
// internal/parkster/types.go
package parkster

type User struct {
    ID              int              `json:"id"`
    Email           string           `json:"email"`
    AccountType     string           `json:"accountType"`
    Cars            []Car            `json:"cars"`
    PaymentAccounts []PaymentAccount `json:"paymentAccounts"`
}

type Car struct {
    ID          int    `json:"id"`
    LicenseNbr  string `json:"licenseNbr"`
    CountryCode string `json:"countryCode"`
}

type PaymentAccount struct {
    PaymentAccountID string `json:"paymentAccountId"`
}

type Zone struct {
    ID      int     `json:"id"`
    Name    string  `json:"name"`
    FeeZone FeeZone `json:"feeZone"`
}

type FeeZone struct {
    ID       int      `json:"id"`
    Currency Currency `json:"currency"`
}

type Currency struct {
    Code   string `json:"code"`
    Symbol string `json:"symbol"`
}

type Parking struct {
    ID          int     `json:"id"`
    ParkingZone Zone    `json:"parkingZone"`
    Car         Car     `json:"car"`
    StartTime   string  `json:"startTime"`
    Timeout     int     `json:"timeout"`
    Cost        float64 `json:"cost"`
    Status      string  `json:"status"`
}
```

## Authentication

Parkster uses email/password credentials (Pattern C from template):

```bash
# Store credentials
parkster auth login
# Prompts for email, then password

# Or use environment variables
export PARKSTER_EMAIL=user@example.com
export PARKSTER_PASSWORD=password123

# Or use flags
parkster start --email user@example.com --password secret --zone 17429 --duration 30
```

**Priority:** CLI flags > env vars > keyring (same as template)

## Multi-Country Support

Parkster operates in multiple countries with the same API:

| Country | Code | Currency | Test Zone |
|---------|------|----------|-----------|
| Sweden | SE | SEK (kr) | 17429 (Ericsson Kista, Stockholm) |
| Germany | DE | EUR (€) | 7713 (Berlin, code 100028) |
| Austria | AT | EUR (€) | 25624 (Salzburg, code 006001) |

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
  --format json
```

**Flow:**
1. Get user profile (cars, payment accounts)
2. Select car (flag or auto if only one)
3. Select payment (flag or auto if only one)
4. Get zone details (need feeZoneId)
5. Start parking
6. Output result

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
parkster status --format json
```

## Testing

See [TESTING.md](./TESTING.md) for Python test suite that validates the API.

```bash
# Run Python API tests
cd /path/to/parkster-cli
python3 test_api.py

# Test zones from API.md
# Sweden: 17429 (Ericsson Kista) - 10 SEK/hour
# Germany: 7713 (Berlin) - €3/hour
# Austria: 25624 (Salzburg) - €2.20/hour
```

## Common Workflows

### Start Parking Workflow

```
User: parkster start --zone 17429 --duration 30

1. auth.GetEmail() → "user@example.com" (flag > env > keyring)
2. auth.GetPassword() → "password123" (flag > env > keyring)
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
9. output.Print(parking, "plain")
```

## Post-MVP Features

See design document for full list. Key additions:

**Zone management:**
- `parkster zones search --lat 59.373 --lon 17.893`
- `parkster zones info <zone-id>`

**Car management:**
- `parkster cars list`
- `parkster cars add <license> --country SE`
- `parkster cars remove <license>`

**History & cost:**
- `parkster history`
- `parkster cost-estimate --zone 17429 --duration 30`

**Config file:**
- `~/.config/parkster/config.yaml`
- `preferred_car`, `preferred_payment`, `default_country`

## Remember

1. **Read ../cli-template/CLAUDE.md first** - it has the patterns
2. **Parkster quirks**:
   - Form-encoded POST (not JSON)
   - Device params everywhere
   - Fee zone ID required
   - Extend uses offset
3. **See API.md** - complete API reference with examples
4. **KISS/YAGNI** - don't add features until needed
