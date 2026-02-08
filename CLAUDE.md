# CLI Tool Development Guide

This is a Go CLI template following patterns from github.com/steipete's CLI ecosystem (gogcli, sonoscli, summarize, etc.).

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

1. Create file: `internal/commands/myfeature.go`
2. Define command with `RunE`
3. Add to root in `init()`: `rootCmd.AddCommand(myFeatureCmd)`
4. If it needs a client, create `internal/client/myservice.go`

## Adding a New API Integration

1. Create client: `internal/client/servicename.go`
2. Add auth command extension in `internal/commands/auth.go`
3. Create feature commands that use the client

## Configuration Priority

1. CLI flags (highest)
2. Environment variables
3. Config file (~/.config/mycli/config.yaml)
4. Defaults (lowest)

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
