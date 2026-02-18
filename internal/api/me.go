package api

import (
	"context"
	"fmt"
)

const queryMe = `
  query Me {
    me {
      id
      name
      email
    }
  }
`

// MeData is the shape of the "me" query response.
type MeData struct {
	Me MeUser `json:"me"`
}

// MeUser holds the fields returned by the "me" query.
type MeUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Me calls the "me" GraphQL query and returns the authenticated user.
func (c *Client) Me(ctx context.Context, token string) (*MeUser, error) {
	var resp struct {
		Data   MeData         `json:"data"`
		Errors []GraphQLError `json:"errors,omitempty"`
	}

	if err := c.do(ctx, token, queryMe, nil, &resp); err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, &QueryError{Message: resp.Errors[0].Message}
	}

	return &resp.Data.Me, nil
}
