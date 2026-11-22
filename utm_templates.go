package shortn

import (
	"context"
	"net/http"
	"net/url"
)

// UTMTemplatesService covers /workspaces/{id}/utm-templates and
// /utm-templates/{id}.
type UTMTemplatesService service

// UTMTemplate is a saved set of UTM parameters. An empty utm_* value means
// the parameter isn't set.
type UTMTemplate struct {
	PublicID          string   `json:"public_id"`
	WorkspacePublicID string   `json:"workspace_public_id"`
	TeamPublicID      string   `json:"team_public_id"`
	TeamName          string   `json:"team_name"`
	TeamPublicIDs     []string `json:"team_public_ids"`
	TeamNames         []string `json:"team_names"`
	WorkspaceWide     bool     `json:"workspace_wide"`
	Name              string   `json:"name"`
	UTMSource         string   `json:"utm_source"`
	UTMMedium         string   `json:"utm_medium"`
	UTMCampaign       string   `json:"utm_campaign"`
	UTMTerm           string   `json:"utm_term"`
	UTMContent        string   `json:"utm_content"`
	UTMID             string   `json:"utm_id"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`

	Raw map[string]any `json:"-"`
}

func (t *UTMTemplate) UnmarshalJSON(b []byte) error {
	type plain UTMTemplate
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// UTMTemplateList is the answer of UTMTemplates.List.
type UTMTemplateList struct {
	Data []UTMTemplate  `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (l *UTMTemplateList) UnmarshalJSON(b []byte) error {
	type plain UTMTemplateList
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// UTMTemplateResponse is the answer of UTMTemplates.Create and Update.
type UTMTemplateResponse struct {
	Data UTMTemplate `json:"data"`
	// MoveImpact is set by Update: the links the change affects ({"count",
	// "links"}). The API sends it inside data, as data.move_impact; it is
	// also in Data.Raw["move_impact"].
	MoveImpact map[string]any `json:"-"`
	Raw        map[string]any `json:"-"`
}

func (r *UTMTemplateResponse) UnmarshalJSON(b []byte) error {
	type plain UTMTemplateResponse
	if err := decodeObject(b, (*plain)(r), &r.Raw); err != nil {
		return err
	}
	r.MoveImpact, _ = r.Data.Raw["move_impact"].(map[string]any)
	return nil
}

// UTMTemplateListOptions filters UTMTemplates.List. Without options the API
// lists only workspace-wide templates.
type UTMTemplateListOptions struct {
	// IncludeTeamScoped adds the team templates you can see.
	IncludeTeamScoped bool
	// TeamPublicIDs returns workspace-wide templates and those of every
	// listed team.
	TeamPublicIDs []string
}

// CreateUTMTemplateParams is the body of UTMTemplates.Create.
type CreateUTMTemplateParams struct {
	Name        string `json:"name,omitempty"`
	UTMSource   string `json:"utm_source,omitempty"`
	UTMMedium   string `json:"utm_medium,omitempty"`
	UTMCampaign string `json:"utm_campaign,omitempty"`
	UTMTerm     string `json:"utm_term,omitempty"`
	UTMContent  string `json:"utm_content,omitempty"`
	UTMID       string `json:"utm_id,omitempty"`
	// TeamPublicIDs limits the template to teams; leave it empty for a
	// workspace-wide template.
	TeamPublicIDs []string `json:"team_public_ids,omitempty"`
}

// UpdateUTMTemplateParams is the body of UTMTemplates.Update. PUT replaces
// the template: a value left empty is cleared, and an empty Name becomes
// "Untitled Template".
type UpdateUTMTemplateParams struct {
	Name        string `json:"name,omitempty"`
	UTMSource   string `json:"utm_source,omitempty"`
	UTMMedium   string `json:"utm_medium,omitempty"`
	UTMCampaign string `json:"utm_campaign,omitempty"`
	UTMTerm     string `json:"utm_term,omitempty"`
	UTMContent  string `json:"utm_content,omitempty"`
	UTMID       string `json:"utm_id,omitempty"`
	// TeamPublicIDs is the complete team list; Strings() makes the template
	// workspace-wide, nil keeps the current teams.
	TeamPublicIDs *[]string `json:"team_public_ids,omitempty"`
	// WorkspacePublicID moves the template to another workspace.
	WorkspacePublicID string `json:"workspace_public_id,omitempty"`
	// LostAccessAction is remove or copy, for links the template would no
	// longer reach. Without it the API first answers 409 with code
	// lost_access_action_required: read the affected links from
	// APIError.Raw["impact"].
	LostAccessAction string `json:"lost_access_action,omitempty"`
}

// UTMTemplateDeleteOptions controls UTMTemplates.Delete.
type UTMTemplateDeleteOptions struct {
	// Links is keep (the API's default: links keep the values as their own)
	// or clear (the values are removed from the links too).
	Links string
}

// UTMTemplateDeleteResult is the answer of UTMTemplates.Delete.
type UTMTemplateDeleteResult struct {
	Data struct {
		PublicID     string `json:"public_id"`
		Deleted      bool   `json:"deleted"`
		Links        string `json:"links"`
		LinksUpdated int64  `json:"links_updated"`
	} `json:"data"`
	Raw map[string]any `json:"-"`
}

func (r *UTMTemplateDeleteResult) UnmarshalJSON(b []byte) error {
	type plain UTMTemplateDeleteResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// List lists a workspace's UTM templates
// (GET /workspaces/{public_id}/utm-templates). opts may be nil.
func (s *UTMTemplatesService) List(ctx context.Context, workspacePublicID string, opts *UTMTemplateListOptions) (*UTMTemplateList, error) {
	q := url.Values{}
	if opts != nil {
		if opts.IncludeTeamScoped {
			q.Set("include_team_scoped", "1")
		}
		setList(q, "team_public_ids", opts.TeamPublicIDs)
	}
	var out UTMTemplateList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("workspaces", workspacePublicID, "utm-templates"), query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates a UTM template (POST /workspaces/{public_id}/utm-templates).
func (s *UTMTemplatesService) Create(ctx context.Context, workspacePublicID string, params CreateUTMTemplateParams) (*UTMTemplateResponse, error) {
	var out UTMTemplateResponse
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("workspaces", workspacePublicID, "utm-templates"), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update replaces a UTM template (PUT /utm-templates/{public_id}).
func (s *UTMTemplatesService) Update(ctx context.Context, templatePublicID string, params UpdateUTMTemplateParams) (*UTMTemplateResponse, error) {
	var out UTMTemplateResponse
	if err := s.client.do(ctx, request{method: http.MethodPut, path: pathEscape("utm-templates", templatePublicID), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a UTM template (DELETE /utm-templates/{public_id}). opts
// may be nil.
func (s *UTMTemplatesService) Delete(ctx context.Context, templatePublicID string, opts *UTMTemplateDeleteOptions) (*UTMTemplateDeleteResult, error) {
	q := url.Values{}
	if opts != nil {
		setString(q, "links", opts.Links)
	}
	var out UTMTemplateDeleteResult
	if err := s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("utm-templates", templatePublicID), query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
