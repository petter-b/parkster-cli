package commands

import (
	"github.com/petter-b/parkster-cli/internal/auth"
	"github.com/petter-b/parkster-cli/internal/parkster"
)

// newAPIClient creates an API client. Replaced in tests with a mock.
var newAPIClient = func(username, password string) parkster.API {
	return parkster.NewClient(username, password)
}

// getCredentials retrieves auth credentials. Replaced in tests to avoid keychain.
// When a caller is detected and stdin is not a TTY (agent mode), it passes
// the caller name to update the keychain item description.
var getCredentials = func() (string, string, auth.CredentialSource, error) {
	if detectedCaller.Name != "" && !isStdinTTY() {
		return auth.GetCredentialsWithCaller(detectedCaller.Name)
	}
	return auth.GetCredentials()
}

// saveCredentials stores auth credentials. Replaced in tests to avoid keychain.
var saveCredentials = auth.SaveCredentials

// deleteCredentials removes auth credentials. Replaced in tests to avoid keychain.
var deleteCredentials = auth.DeleteCredentials
