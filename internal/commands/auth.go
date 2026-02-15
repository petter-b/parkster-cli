package commands

import (
	"bufio"
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
	Long: `Manage authentication credentials stored in your OS keychain.

Credentials are stored securely using:
- macOS: Keychain
- Linux: Secret Service (GNOME Keyring, KWallet)
- Windows: Credential Manager

Environment variables take precedence over stored credentials.`,
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
	fmt.Fprintf(os.Stderr, "Enter username (email or phone): ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	fmt.Fprintf(os.Stderr, "Enter password: ")
	password, err := readSecretLine()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if err := auth.SaveCredentials(username, password); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Credentials stored for %s\n", username)
	return nil
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	if err := auth.DeleteCredentials(); err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Credentials removed\n")
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	type authStatus struct {
		Authenticated bool   `json:"authenticated"`
		Username      string `json:"username,omitempty"`
	}

	username, _, err := auth.GetCredentials(nil)
	if err != nil {
		status := authStatus{Authenticated: false}
		mode := OutputMode()
		if mode == output.ModeJSON {
			return output.PrintSuccess(status, mode)
		}
		fmt.Fprintln(os.Stderr, "Not authenticated")
		return nil
	}

	status := authStatus{Authenticated: true, Username: username}
	mode := OutputMode()
	if mode == output.ModeJSON {
		return output.PrintSuccess(status, mode)
	}
	fmt.Fprintf(os.Stderr, "Logged in as: %s\n", username)
	return nil
}

// readSecretLine reads a line without echoing the input to the terminal.
func readSecretLine() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return "", err
		}
		return string(password), nil
	}
	// Non-terminal (piped input): read normally
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
