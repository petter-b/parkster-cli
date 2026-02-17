package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication credentials",
	Long: `Manage authentication credentials.

Credentials are stored using (in order of preference):
- OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- Plaintext file (~/.config/parkster/credentials.json)

Environment variables (PARKSTER_USERNAME/PARKSTER_PASSWORD) are used as a last resort.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("unknown subcommand %q for %q", args[0], cmd.Name())
	},
}

var authAddCmd = &cobra.Command{
	Use:   "login",
	Short: "Store Parkster credentials",
	Long: `Store Parkster username and password in OS keychain.

Examples:
  parkster auth login     # Prompts for username and password

The credentials will be stored in your OS keychain.`,
	Args: cobra.NoArgs,
	RunE: runAuthAdd,
}

var authRemoveCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove Parkster credentials",
	Args:  cobra.NoArgs,
	RunE:  runAuthRemove,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Args:  cobra.NoArgs,
	RunE:  runAuthStatus,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authAddCmd, authRemoveCmd, authStatusCmd)
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintf(os.Stderr, "Enter username (email or phone): ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	fmt.Fprintf(os.Stderr, "Enter password: ")
	var password string
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		pw, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = string(pw)
	} else {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		password = strings.TrimSpace(line)
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Validate credentials against API before storing
	client := newAPIClient(username, password)
	if _, err := client.Login(); err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	source, err := saveCredentials(username, password)
	if err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	if source == auth.SourceFile {
		fmt.Fprintf(os.Stderr, "Credentials stored for %s (file: %s)\n", username, auth.CredentialsFilePath())
	} else {
		fmt.Fprintf(os.Stderr, "Credentials stored for %s (%s)\n", username, source)
	}
	return nil
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	if err := deleteCredentials(); err != nil {
		if errors.Is(err, auth.ErrNoCredentials) {
			fmt.Fprintln(os.Stderr, "No credentials to remove")
			return nil
		}
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Credentials removed")
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	type authStatus struct {
		Authenticated bool   `json:"authenticated"`
		Username      string `json:"username,omitempty"`
		Source        string `json:"source,omitempty"`
		Error         string `json:"error,omitempty"`
	}

	username, password, source, err := getCredentials()
	if err != nil {
		return authRequiredError()
	}

	// Validate credentials against API
	client := newAPIClient(username, password)
	_, loginErr := client.Login()

	mode := OutputMode()

	if loginErr != nil {
		// Credentials exist but are invalid
		if mode != output.ModeHuman {
			output.PrintError(fmt.Sprintf("credentials found but authentication failed (%s)", source), mode)
			return nil
		}
		fmt.Fprintf(os.Stderr, "Credentials found but authentication failed (%s)\n", source)
		return nil
	}

	status := authStatus{Authenticated: true, Username: username, Source: string(source)}
	if mode != output.ModeHuman {
		return output.PrintSuccess(status, mode)
	}
	fmt.Fprintf(os.Stderr, "Logged in as: %s (%s)\n", username, source)
	return nil
}
