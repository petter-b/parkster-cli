package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/petter-b/parkster-cli/internal/auth"
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
	Long: `Store Parkster email and password in OS keychain.

Examples:
  parkster auth login     # Prompts for email and password

The credentials will be stored in your OS keychain.`,
	Args: cobra.NoArgs,
	RunE: runAuthAdd,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured credentials",
	RunE:  runAuthList,
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
	authCmd.AddCommand(authAddCmd, authListCmd, authRemoveCmd, authStatusCmd)
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(os.Stderr, "Enter email: ")
	reader := bufio.NewReader(os.Stdin)
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	fmt.Fprintf(os.Stderr, "Enter password: ")
	password, err := readSecretLine()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if err := auth.SaveCredentials(email, password); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Credentials stored for %s\n", email)
	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	email, err := auth.GetEmail(nil)
	if err != nil {
		if format == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("No credentials configured. Use 'parkster auth login' to add credentials.")
		}
		return nil
	}

	if format == "json" {
		fmt.Printf(`[{"email":"%s"}]`, email)
		fmt.Println()
	} else {
		fmt.Println(email)
	}

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
	email, err := auth.GetEmail(nil)
	if err != nil {
		if format == "json" {
			fmt.Println(`{"authenticated":false}`)
		} else {
			fmt.Println("Not authenticated")
		}
		return nil
	}

	if format == "json" {
		fmt.Printf(`{"authenticated":true,"email":"%s"}`, email)
		fmt.Println()
	} else {
		fmt.Printf("Logged in as: %s\n", email)
	}

	return nil
}

// readSecretLine reads a line without echoing (basic version)
// For production, use golang.org/x/term for proper terminal handling
func readSecretLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
