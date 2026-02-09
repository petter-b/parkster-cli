package commands

import "github.com/petter-b/parkster-cli/internal/parkster"

// newAPIClient creates an API client. Replaced in tests with a mock.
var newAPIClient = func(username, password string) parkster.API {
	return parkster.NewClient(username, password)
}
