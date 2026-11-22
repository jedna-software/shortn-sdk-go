package shortn

import (
	"context"
	"net/http"
	"net/url"
)

// TagsService covers /workspaces/{id}/tags and /tags/{id}. These endpoints
// answer their objects without a data wrapper.
type TagsService service

// Tag labels links in a workspace.
type Tag struct {
	ID            int64          `json:"id"`
	PublicID      string         `json:"public_id"`
	UserID        int64          `json:"user_id"`      // only Tags.List sets it
	WorkspaceID   int64          `json:"workspace_id"` // only Tags.List sets it
	WorkspaceWide bool           `json:"workspace_wide"`
	Name          string         `json:"name"`
	Color         string         `json:"color"`
	URLCount      int64          `json:"url_count"`
	TeamPublicIDs []string       `json:"team_public_ids"`
	TeamNames     []string       `json:"team_names"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"` // only Tags.List sets it
	Raw           map[string]any `json:"-"`
}

func (t *Tag) UnmarshalJSON(b []byte) error {
	type plain Tag
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// TagList is the answer of Tags.List. Tags is nil when there are none.
type TagList struct {
	Tags []Tag          `json:"tags"`
	Raw  map[string]any `json:"-"`
}

func (t *TagList) UnmarshalJSON(b []byte) error {
	type plain TagList
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// TagListOptions filters Tags.List. Zero values are not sent.
type TagListOptions struct {
	// TeamPublicIDs returns workspace-wide tags and those of every listed team.
	TeamPublicIDs []string
	// Team "none" returns only workspace-wide tags.
	Team string
	// ReportScope treats the teams as a report's scope (report_scope=1).
	ReportScope bool
}

// CreateTagParams is the body of Tags.Create.
type CreateTagParams struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	// TeamPublicIDs limits the tag to teams; leave it empty for a
	// workspace-wide tag.
	TeamPublicIDs []string `json:"team_public_ids,omitempty"`
}

// UpdateTagParams is the body of Tags.Update: change Name and/or Color, or
// replace the teams with TeamPublicIDs, not both in one request.
type UpdateTagParams struct {
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
	// TeamPublicIDs is the tag's complete team list; Strings() makes it
	// workspace-wide.
	TeamPublicIDs *[]string `json:"team_public_ids,omitempty"`
}

// TagDeleteResult is the answer of Tags.Delete.
type TagDeleteResult struct {
	Message      string         `json:"message"`
	LinksUpdated int64          `json:"links_updated"`
	Raw          map[string]any `json:"-"`
}

func (r *TagDeleteResult) UnmarshalJSON(b []byte) error {
	type plain TagDeleteResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// List lists a workspace's tags, sorted by name
// (GET /workspaces/{public_id}/tags). opts may be nil.
func (s *TagsService) List(ctx context.Context, workspacePublicID string, opts *TagListOptions) (*TagList, error) {
	q := url.Values{}
	if opts != nil {
		setList(q, "team_public_ids", opts.TeamPublicIDs)
		setString(q, "team", opts.Team)
		if opts.ReportScope {
			q.Set("report_scope", "1")
		}
	}
	var out TagList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("workspaces", workspacePublicID, "tags"), query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates a tag (POST /workspaces/{public_id}/tags).
func (s *TagsService) Create(ctx context.Context, workspacePublicID string, params CreateTagParams) (*Tag, error) {
	var out Tag
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("workspaces", workspacePublicID, "tags"), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get reads a tag by public_id (GET /tags/{public_id}).
func (s *TagsService) Get(ctx context.Context, publicID string) (*Tag, error) {
	var out Tag
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("tags", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes a tag (PUT /tags/{public_id}) and returns it.
func (s *TagsService) Update(ctx context.Context, publicID string, params UpdateTagParams) (*Tag, error) {
	var out Tag
	if err := s.client.do(ctx, request{method: http.MethodPut, path: pathEscape("tags", publicID), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a tag and removes it from its links (DELETE /tags/{public_id}).
func (s *TagsService) Delete(ctx context.Context, publicID string) (*TagDeleteResult, error) {
	var out TagDeleteResult
	if err := s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("tags", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
