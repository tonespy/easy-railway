package api

import (
	"context"
	"fmt"
)

//nolint:gosec // GraphQL query name, not a credential.
const queryProjectToken = `
  query {
    projectToken {
      id
      name
      projectId
      environmentId
    }
  }
`

// ProjectTokenData is the shape of the "projectToken" query response.
type ProjectTokenData struct {
	ProjectToken ProjectTokenInfo `json:"projectToken"`
}

// ProjectTokenInfo holds the fields returned by the "projectToken" query.
type ProjectTokenInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ProjectID     string `json:"projectId"`
	EnvironmentID string `json:"environmentId"`
}

// ValidateProjectToken calls the "projectToken" GraphQL query using the Project-Access-Token header.
func (c *Client) ValidateProjectToken(ctx context.Context, token string) (*ProjectTokenInfo, error) {
	var resp struct {
		Data   ProjectTokenData `json:"data"`
		Errors []GraphQLError   `json:"errors,omitempty"`
	}

	err := c.doWithHeaders(ctx, map[string]string{
		"Content-Type":         "application/json",
		"Project-Access-Token": token,
	}, queryProjectToken, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("validate project token: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, &QueryError{Message: resp.Errors[0].Message}
	}

	return &resp.Data.ProjectToken, nil
}
