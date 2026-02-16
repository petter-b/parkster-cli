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
var getCredentials = auth.GetCredentials

// saveCredentials stores auth credentials. Replaced in tests to avoid keychain.
var saveCredentials = auth.SaveCredentials

// deleteCredentials removes auth credentials. Replaced in tests to avoid keychain.
var deleteCredentials = auth.DeleteCredentials
