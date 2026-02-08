# CLI Tool Development Guide

This is a Go CLI template following patterns from github.com/steipete's CLI ecosystem (gogcli, sonoscli, summarize, etc.).

## Before You Start

**READ EXISTING CODE FIRST** before proposing new patterns:

1. Study `internal/commands/auth.go` - reference implementation showing all patterns
2. Check `internal/output/output.go` - output formatting already implemented
3. Review `internal/commands/root.go` - global flags and debug logging
4. Read `internal/auth/keyring.go` - credential management patterns

**Don't reinvent - reuse and extend:**
- Output? Use `output.Print(data, format)`
- Debug? Use `debugLog("message")`
- Errors? Wrap with `fmt.Errorf("context: %w", err)`
- Auth? Extend existing keyring pattern

**KISS and YAGNI principles:**
- ❌ Don't add config file until proven needed (flags + env vars usually enough for MVP)
- ❌ Don't add helper functions until duplicated 3+ times
- ❌ Don't add abstractions speculatively
- ✅ Inline logic first, extract only when painful
- ✅ Start simple, add complexity when needed

## Quick Start

```bash
# Build
make build

# Run
./bin/mycli --help

# Add authentication
./bin/mycli auth add myaccount

# Test with JSON output
./bin/mycli auth list --format json
```

## Architecture

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Single binary, fast startup, cross-platform |
| CLI Framework | spf13/cobra | Industry standard, subcommands, auto-help |
| Credential Storage | 99designs/keyring | Cross-platform OS keychain integration |
| Config Location | XDG (~/.config/mycli/) | Standard, respects $XDG_CONFIG_HOME |

## Directory Structure

```
mycli/
├── cmd/mycli/
│   └── main.go              # Entry point - minimal, just calls Execute()
├── internal/
│   ├── commands/            # Cobra command definitions
│   │   ├── root.go          # Root cmd + global flags (--format, --debug)
│   │   ├── auth.go          # auth add/list/remove subcommands
│   │   └── version.go       # version command
│   ├── auth/
│   │   ├── keyring.go       # Secure credential storage
│   │   └── oauth.go         # OAuth2 browser flow (if needed)
│   ├── client/              # API client(s) - add your integrations here
│   │   └── client.go
│   └── config/
│       └── config.go        # Config file loading (YAML)
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── CLAUDE.md                # This file
```

## Code Patterns

### 1. Command Structure (Cobra)

Always use `RunE` (returns error) not `Run`:

```go
var myCmd = &cobra.Command{
    Use:   "mycommand [args]",
    Short: "One-line description",
    Long:  `Longer description with examples.`,
    Args:  cobra.ExactArgs(1),  // or MinimumNArgs, MaximumNArgs
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get flags
        format, _ := cmd.Flags().GetString("format")
        
        // Do work
        result, err := doSomething(args[0])
        if err != nil {
            return fmt.Errorf("failed to do something: %w", err)
        }
        
        // Output
        return output.Print(result, format)
    },
}
```

### 2. Error Handling

Wrap errors with context, let Cobra handle printing:

```go
// Good
return fmt.Errorf("failed to connect to %s: %w", addr, err)

// Bad
fmt.Println("Error:", err)
return err
```

### 3. Output Formatting

Support machine-readable output for AI agents:

```go
// internal/output/output.go
func Print(data any, format string) error {
    switch format {
    case "json":
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(data)
    case "tsv":
        // Tab-separated for easy parsing
        return printTSV(data)
    default: // "plain"
        return printPlain(data)
    }
}
```

### 4. Credential Storage

Never store secrets in config files:

```go
// Store
err := auth.SetCredential("service-name", "api-key-value")

// Retrieve (checks env var first, then keyring)
key, err := auth.GetCredential("service-name")
```

### 5. Debug/Verbose Output

Debug info goes to stderr, data to stdout:

```go
if debug {
    fmt.Fprintf(os.Stderr, "DEBUG: connecting to %s\n", url)
}
// Actual output to stdout
fmt.Println(result)
```

## Authentication Patterns

### Pattern A: API Key (Simple)

For services with static API keys:

```go
// User sets via: mycli auth add servicename
// Or env var: MYCLI_SERVICENAME_API_KEY=xxx

key, err := auth.GetCredential("servicename")
client := api.NewClient(key)
```

### Pattern B: OAuth2 Browser Flow

For services requiring OAuth (Google, GitHub, etc.):

```go
// mycli auth add user@example.com --services gmail,drive

// 1. Open browser to auth URL
// 2. Start local HTTP server on localhost:8085
// 3. Receive callback with auth code
// 4. Exchange for tokens
// 5. Store refresh token in keyring
```

## Adding a New Command

**Process:**
1. Study `internal/commands/auth.go` first (reference implementation)
2. Create file: `internal/commands/myfeature.go`
3. Copy the pattern from existing commands
4. Define command with `RunE`
5. Add to root in `init()`: `rootCmd.AddCommand(myFeatureCmd)`
6. If it needs a client, create `internal/client/myservice.go`

**Checklist for new commands:**
- [ ] Uses `RunE` (returns error), not `Run`
- [ ] Required params as flags (e.g., `--zone`, `--duration`), not positional args (better for AI agents)
- [ ] Uses `output.Print(data, format)` for data output to stdout
- [ ] Uses `debugLog("message")` for debug output to stderr
- [ ] Uses `fmt.Fprintf(os.Stderr, "...")` for user status messages
- [ ] Wraps errors with `fmt.Errorf("context: %w", err)`
- [ ] Respects `--format` flag (json/tsv/plain)
- [ ] No interactive prompts (use flags + error messages instead)

**Complete example:**

```go
// internal/commands/park.go
package commands

import (
    "fmt"
    "github.com/spf13/cobra"
    "yourorg/mycli/internal/auth"
    "yourorg/mycli/internal/client"
    "yourorg/mycli/internal/output"
)

var parkCmd = &cobra.Command{
    Use:   "park",
    Short: "Start a parking session",
    RunE:  runPark,
}

func init() {
    rootCmd.AddCommand(parkCmd)
    // All required params as flags (better for AI agents than positional args)
    parkCmd.Flags().Int("zone", 0, "Parking zone ID (required)")
    parkCmd.Flags().Int("duration", 30, "Duration in minutes")
    parkCmd.MarkFlagRequired("zone")
}

func runPark(cmd *cobra.Command, args []string) error {
    // 1. Get flags
    zoneID, _ := cmd.Flags().GetInt("zone")
    duration, _ := cmd.Flags().GetInt("duration")

    // 2. Get credentials (flag → env → keyring)
    email, err := auth.GetEmail(cmd)
    if err != nil {
        return fmt.Errorf("authentication required: %w", err)
    }
    password, err := auth.GetPassword(cmd)
    if err != nil {
        return fmt.Errorf("authentication required: %w", err)
    }

    // 3. Create client
    apiClient := client.NewClient(email, password)

    // 4. Debug logging (to stderr)
    debugLog("starting parking at zone %d for %d minutes", zoneID, duration)

    // 5. Call API
    result, err := apiClient.StartParking(zoneID, duration)
    if err != nil {
        return fmt.Errorf("failed to start parking: %w", err)
    }

    // 6. User status message (to stderr)
    fmt.Fprintf(os.Stderr, "Parking started successfully\n")

    // 7. Output data (to stdout, respects --format flag)
    return output.Print(result, GetFormat())
}
```

## Adding a New API Integration

1. Create client: `internal/client/servicename.go`
2. Add auth command extension in `internal/commands/auth.go`
3. Create feature commands that use the client

## Configuration Priority

1. CLI flags (highest)
2. Environment variables
3. Config file (~/.config/mycli/config.yaml)
4. Defaults (lowest)

**Complete credential example:**

```go
// internal/commands/root.go - Add global credential flags
rootCmd.PersistentFlags().String("email", "", "Account email")
rootCmd.PersistentFlags().String("password", "", "Account password")

// internal/auth/keyring.go - Priority implementation
func GetEmail(cmd *cobra.Command) (string, error) {
    // 1. Check CLI flag (highest priority)
    if email, _ := cmd.Flags().GetString("email"); email != "" {
        return email, nil
    }
    // 2. Check environment variable
    if email := os.Getenv("MYCLI_EMAIL"); email != "" {
        return email, nil
    }
    // 3. Check keyring
    email, err := keyring.Get("mycli", "email")
    if err != nil {
        return "", fmt.Errorf("no credentials found (use --email flag, MYCLI_EMAIL env var, or 'mycli auth login')")
    }
    return email, nil
}
```

**Usage examples:**
```bash
# CLI flags (override everything)
mycli command --email user@example.com --password secret

# Environment variables
export MYCLI_EMAIL=user@example.com
export MYCLI_PASSWORD=secret
mycli command

# Stored credentials (from keyring)
mycli auth login  # Stores in keyring
mycli command     # Uses stored credentials
```

## Testing

```bash
make test          # Run all tests
make test-verbose  # With verbose output
make lint          # Run golangci-lint
```

## Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/spf13/cobra | v1.8.0 | CLI framework |
| github.com/99designs/keyring | v1.2.2 | Secure credential storage |
| golang.org/x/oauth2 | v0.16.0 | OAuth2 flows |
| gopkg.in/yaml.v3 | v3.0.1 | Config file parsing |

## AI Agent Friendliness

These CLIs are designed for AI agent integration. Follow these patterns:

**1. Use flags, not positional args (for required params):**
```bash
# ✅ Good - self-documenting, order-independent
mycli park --zone 17429 --duration 30

# ❌ Avoid - order matters, unclear what values mean
mycli park 17429 30
```

**2. No interactive prompts:**
```go
// ❌ Bad - blocks AI agents
fmt.Print("Enter zone ID: ")
fmt.Scanln(&zoneID)

// ✅ Good - require via flags, provide helpful error
if zoneID == 0 {
    return fmt.Errorf("--zone flag required")
}
```

**3. Machine-readable errors with context:**
```go
// When multiple options exist, output them in structured format
if len(cars) > 1 && carFlag == "" {
    output.Print(cars, format)  // AI can parse JSON
    return fmt.Errorf("multiple cars found, use --car flag to specify")
}
```

**4. Structured output (JSON/TSV):**
```bash
# AI agents can parse and act on this
mycli park --zone 17429 --duration 30 --format json
{
  "id": 123456,
  "zone": "17429",
  "status": "active",
  "cost": 0.0
}
```

## Reference Implementations

Study these for patterns:

- **github.com/steipete/gogcli** - OAuth2 + Google APIs, multi-account
- **github.com/steipete/sonoscli** - Local network discovery, device control
- **github.com/steipete/summarize** - TypeScript, API keys, local daemon
- **github.com/steipete/Peekaboo** - Swift, macOS system integration

## Common Tasks

### "Add support for ServiceX API"

1. Get API docs, identify auth method (API key vs OAuth)
2. Create `internal/client/servicex.go` with API methods
3. Add auth storage in `internal/commands/auth.go`
4. Create commands in `internal/commands/servicex.go`
5. Document in README.md

### "Add a new subcommand"

```go
// internal/commands/newcmd.go
package commands

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
    Use:   "new",
    Short: "Does something new",
    RunE:  runNew,
}

func init() {
    rootCmd.AddCommand(newCmd)
    newCmd.Flags().String("option", "default", "An option")
}

func runNew(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

### "Support a config file option"

1. Add field to `Config` struct in `internal/config/config.go`
2. Add corresponding flag in command
3. In `RunE`, check: flag → env → config → default
