package shortn

import (
	"context"
	"net/http"
)

// WorkspacesService covers /workspaces.
type WorkspacesService service

// Workspace is a workspace the API key can reach.
type Workspace struct {
	PublicID    string `json:"public_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPersonal  bool   `json:"is_personal"`
	Active      bool   `json:"active"`
	// Role is owner, administrator, collaborator or read_only.
	Role        string         `json:"role"`
	TeamCount   int64          `json:"team_count"`
	MemberCount int64          `json:"member_count"`
	LinkCount   int64          `json:"link_count"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	Raw         map[string]any `json:"-"`
}

func (w *Workspace) UnmarshalJSON(b []byte) error {
	type plain Workspace
	return decodeObject(b, (*plain)(w), &w.Raw)
}

// WorkspaceList is the answer of Workspaces.List.
type WorkspaceList struct {
	Data []Workspace    `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (w *WorkspaceList) UnmarshalJSON(b []byte) error {
	type plain WorkspaceList
	return decodeObject(b, (*plain)(w), &w.Raw)
}

// WorkspaceResponse is the answer of Workspaces.Get.
type WorkspaceResponse struct {
	Data Workspace      `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (w *WorkspaceResponse) UnmarshalJSON(b []byte) error {
	type plain WorkspaceResponse
	return decodeObject(b, (*plain)(w), &w.Raw)
}

// List lists the workspaces the key can reach (GET /workspaces).
func (s *WorkspacesService) List(ctx context.Context) (*WorkspaceList, error) {
	var out WorkspaceList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/workspaces"}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get reads one active workspace (GET /workspaces/{public_id}).
func (s *WorkspacesService) Get(ctx context.Context, publicID string) (*WorkspaceResponse, error) {
	var out WorkspaceResponse
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("workspaces", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
