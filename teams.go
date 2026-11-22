package shortn

import (
	"context"
	"net/http"
	"net/url"
)

// TeamsService covers /workspaces/{id}/teams and /teams/{slug}. These
// endpoints answer without a data wrapper.
type TeamsService service

// Team groups links and people inside a workspace. Single teams are
// addressed by slug; other endpoints take the team's public_id.
type Team struct {
	ID                int64  `json:"id"`
	PublicID          string `json:"public_id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Description       string `json:"description"`
	WorkspaceID       int64  `json:"workspace_id"`
	WorkspacePublicID string `json:"workspace_public_id"`
	OwnerUserID       int64  `json:"owner_user_id"`
	Active            bool   `json:"active"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	// The counts are left out (zero) when they're 0.
	MemberCount   int64          `json:"member_count"`
	LinkCount     int64          `json:"link_count"`
	TemplateCount int64          `json:"template_count"`
	Raw           map[string]any `json:"-"`
}

func (t *Team) UnmarshalJSON(b []byte) error {
	type plain Team
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// TeamList is the answer of Teams.List. Teams is nil when there are none.
type TeamList struct {
	Teams []Team         `json:"teams"`
	Raw   map[string]any `json:"-"`
}

func (l *TeamList) UnmarshalJSON(b []byte) error {
	type plain TeamList
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// CreateTeamParams is the body of Teams.Create.
type CreateTeamParams struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreatedTeam is the answer of Teams.Create.
type CreatedTeam struct {
	ID       int64          `json:"id"`
	Slug     string         `json:"slug"`
	PublicID string         `json:"public_id"`
	Raw      map[string]any `json:"-"`
}

func (t *CreatedTeam) UnmarshalJSON(b []byte) error {
	type plain CreatedTeam
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// UpdateTeamParams is the body of Teams.Update; only set fields are sent.
type UpdateTeamParams struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// TeamOptions picks the workspace a slug is looked up in. A slug is unique
// only within its workspace: with an account-wide key that reaches several
// workspaces, an ambiguous slug answers 409 unless Workspace is set.
type TeamOptions struct {
	// Workspace is a workspace public_id (sent as ?workspace=).
	Workspace string
}

func (o *TeamOptions) query() url.Values {
	q := url.Values{}
	if o != nil {
		setString(q, "workspace", o.Workspace)
	}
	return q
}

// List lists a workspace's active teams, newest first
// (GET /workspaces/{public_id}/teams).
func (s *TeamsService) List(ctx context.Context, workspacePublicID string) (*TeamList, error) {
	var out TeamList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("workspaces", workspacePublicID, "teams")}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates a team (POST /workspaces/{public_id}/teams).
func (s *TeamsService) Create(ctx context.Context, workspacePublicID string, params CreateTeamParams) (*CreatedTeam, error) {
	var out CreatedTeam
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("workspaces", workspacePublicID, "teams"), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get reads a team by slug (GET /teams/{slug}). opts may be nil.
func (s *TeamsService) Get(ctx context.Context, slug string, opts *TeamOptions) (*Team, error) {
	var out Team
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("teams", slug), query: opts.query()}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update renames a team or changes its description (PUT /teams/{slug}) and
// returns it. opts may be nil.
func (s *TeamsService) Update(ctx context.Context, slug string, params UpdateTeamParams, opts *TeamOptions) (*Team, error) {
	var out Team
	if err := s.client.do(ctx, request{method: http.MethodPut, path: pathEscape("teams", slug), query: opts.query(), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete archives a team (DELETE /teams/{slug}). opts may be nil.
func (s *TeamsService) Delete(ctx context.Context, slug string, opts *TeamOptions) (*SuccessResult, error) {
	var out SuccessResult
	if err := s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("teams", slug), query: opts.query()}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
