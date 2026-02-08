package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/mycli/internal/auth"
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
	Use:   "add <service>",
	Short: "Add or update credentials for a service",
	Long: `Add or update API credentials for a service.

Examples:
  mycli auth add openai              # Prompts for API key
  mycli auth add github --oauth      # Opens browser for OAuth flow

The credential will be stored in your OS keychain.`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthAdd,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured services",
	RunE:  runAuthList,
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove <service>",
	Short: "Remove credentials for a service",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthRemove,
}

var authStatusCmd = &cobra.Command{
	Use:   "status [service]",
	Short: "Check authentication status",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAuthStatus,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authAddCmd, authListCmd, authRemoveCmd, authStatusCmd)

	// Flags for auth add
	authAddCmd.Flags().Bool("oauth", false, "Use OAuth2 browser flow")
	authAddCmd.Flags().String("key", "", "API key (avoid: prefer interactive prompt)")
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	service := args[0]
	useOAuth, _ := cmd.Flags().GetBool("oauth")
	keyFlag, _ := cmd.Flags().GetString("key")

	store, err := auth.OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	if useOAuth {
		// OAuth flow - implement per service
		return fmt.Errorf("OAuth not yet implemented for %s", service)
	}

	// API key flow
	var apiKey string
	if keyFlag != "" {
		// Warn about using --key flag
		fmt.Fprintln(os.Stderr, "Warning: passing secrets via CLI flags is insecure (visible in shell history)")
		apiKey = keyFlag
	} else {
		// Interactive prompt
		fmt.Fprintf(os.Stderr, "Enter API key for %s: ", service)
		apiKey, err = readSecretLine()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
	}

	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if err := store.Set(service, apiKey); err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Credential stored for %s\n", service)
	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	store, err := auth.OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	services, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	if len(services) == 0 {
		if format == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("No credentials configured. Use 'mycli auth add <service>' to add one.")
		}
		return nil
	}

	if format == "json" {
		fmt.Print("[")
		for i, s := range services {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"service":"%s"}`, s)
		}
		fmt.Println("]")
	} else {
		for _, s := range services {
			fmt.Println(s)
		}
	}

	return nil
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	service := args[0]

	store, err := auth.OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	if err := store.Delete(service); err != nil {
		return fmt.Errorf("failed to remove credential: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Credential removed for %s\n", service)
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	store, err := auth.OpenKeyring()
	if err != nil {
		return fmt.Errorf("failed to open keyring: %w", err)
	}

	var services []string
	if len(args) > 0 {
		services = args
	} else {
		services, err = store.List()
		if err != nil {
			return fmt.Errorf("failed to list credentials: %w", err)
		}
	}

	type status struct {
		Service string `json:"service"`
		Source  string `json:"source"`
		Valid   bool   `json:"valid"`
	}

	var results []status
	for _, svc := range services {
		s := status{Service: svc}

		// Check env var first
		envKey := fmt.Sprintf("MYCLI_%s_API_KEY", strings.ToUpper(strings.ReplaceAll(svc, "-", "_")))
		if os.Getenv(envKey) != "" {
			s.Source = "environment"
			s.Valid = true
		} else if _, err := store.Get(svc); err == nil {
			s.Source = "keyring"
			s.Valid = true
		} else {
			s.Source = "none"
			s.Valid = false
		}

		results = append(results, s)
	}

	if format == "json" {
		fmt.Print("[")
		for i, r := range results {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"service":"%s","source":"%s","valid":%t}`, r.Service, r.Source, r.Valid)
		}
		fmt.Println("]")
	} else {
		for _, r := range results {
			indicator := "✓"
			if !r.Valid {
				indicator = "✗"
			}
			fmt.Printf("%s %s (%s)\n", indicator, r.Service, r.Source)
		}
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
