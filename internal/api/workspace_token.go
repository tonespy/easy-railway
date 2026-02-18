package api

import (
	"context"
	"fmt"
)

const queryWorkspace = `
  query Workspace($id: String!) {
    workspace(workspaceId: $id) {
      id
      name
    }
  }
`

// WorkspaceData is the shape of the "workspace" query response.
type WorkspaceData struct {
	Workspace WorkspaceInfo `json:"workspace"`
}

// WorkspaceInfo holds the fields returned by the "workspace" query.
type WorkspaceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ValidateWorkspaceToken calls the "workspace" GraphQL query to validate a workspace token.
func (c *Client) ValidateWorkspaceToken(ctx context.Context, token, workspaceID string) (*WorkspaceInfo, error) {
	var resp struct {
		Data   WorkspaceData  `json:"data"`
		Errors []GraphQLError `json:"errors,omitempty"`
	}

	err := c.do(ctx, token, queryWorkspace, map[string]any{
		"id": workspaceID,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("validate workspace token: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, &QueryError{Message: resp.Errors[0].Message}
	}

	return &resp.Data.Workspace, nil
}
