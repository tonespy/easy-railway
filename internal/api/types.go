package api

import "fmt"

// GraphQLRequest is the JSON body sent to the GraphQL endpoint.
type GraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// GraphQLError represents one entry in the GraphQL errors array.
type GraphQLError struct {
	Message string `json:"message"`
}

// HTTPError is returned when the HTTP response has a non-2xx status.
type HTTPError struct {
	StatusCode int
	Body       string
}

// Error returns a formatted HTTP error message.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("api: HTTP %d: %s", e.StatusCode, e.Body)
}

// QueryError is returned when GraphQL returns an errors array.
type QueryError struct {
	Message string
}

// Error returns the GraphQL error message.
func (e *QueryError) Error() string {
	return fmt.Sprintf("graphql: %s", e.Message)
}
