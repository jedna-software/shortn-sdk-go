package shortn

import (
	"context"
	"net/http"
)

// DestinationsService covers /workspaces/{id}/destinations: where reports
// and alerts are sent. These endpoints answer without a data wrapper.
type DestinationsService service

// Destination is an email list, webhook, or Slack, Discord or Microsoft
// Teams channel.
type Destination struct {
	ID       int64  `json:"id"`
	PublicID string `json:"public_id"`
	Name     string `json:"name"`
	// Type is email, webhook, slack, discord or msteams.
	Type string `json:"type"`
	// Value is the email recipients (comma-separated) or the webhook URL.
	Value         string         `json:"value"`
	TeamPublicID  string         `json:"team_public_id"`
	TeamName      string         `json:"team_name"`
	TeamPublicIDs []string       `json:"team_public_ids"`
	TeamNames     []string       `json:"team_names"`
	WorkspaceWide bool           `json:"workspace_wide"`
	CreatedAt     string         `json:"created_at"`
	Raw           map[string]any `json:"-"`
}

func (d *Destination) UnmarshalJSON(b []byte) error {
	type plain Destination
	return decodeObject(b, (*plain)(d), &d.Raw)
}

// CreateDestinationParams is the body of Destinations.Create.
type CreateDestinationParams struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Config is {"to": [...]} for email and {"webhook_url": "https://..."}
	// for the other types.
	Config map[string]any `json:"config"`
	// TeamPublicIDs limits the destination to teams; leave it empty for a
	// workspace-wide destination.
	TeamPublicIDs []string `json:"team_public_ids,omitempty"`
}

// UpdateDestinationParams is the body of Destinations.Update. Name and Type
// are required every time.
type UpdateDestinationParams struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Config replaces the configuration; nil keeps it.
	Config map[string]any `json:"config,omitempty"`
	// TeamPublicIDs is the complete team list; Strings() makes the
	// destination workspace-wide, nil keeps the current teams.
	TeamPublicIDs *[]string `json:"team_public_ids,omitempty"`
}

// DestinationUpdateResult is the answer of Destinations.Update. The team
// fields are set only when the update sent TeamPublicIDs.
type DestinationUpdateResult struct {
	Success       bool             `json:"success"`
	WorkspaceWide *bool            `json:"workspace_wide"`
	TeamPublicIDs []string         `json:"team_public_ids"`
	TeamNames     []string         `json:"team_names"`
	Teams         []TeamAssignment `json:"teams"`
	Raw           map[string]any   `json:"-"`
}

func (r *DestinationUpdateResult) UnmarshalJSON(b []byte) error {
	type plain DestinationUpdateResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// List lists a workspace's destinations
// (GET /workspaces/{public_id}/destinations). The API answers a bare array.
func (s *DestinationsService) List(ctx context.Context, workspacePublicID string) ([]Destination, error) {
	var out []Destination
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("workspaces", workspacePublicID, "destinations")}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create creates a destination (POST /workspaces/{public_id}/destinations).
func (s *DestinationsService) Create(ctx context.Context, workspacePublicID string, params CreateDestinationParams) (*Destination, error) {
	var out Destination
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("workspaces", workspacePublicID, "destinations"), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes a destination
// (PUT /workspaces/{public_id}/destinations/{destination_public_id}).
func (s *DestinationsService) Update(ctx context.Context, workspacePublicID, destinationPublicID string, params UpdateDestinationParams) (*DestinationUpdateResult, error) {
	var out DestinationUpdateResult
	if err := s.client.do(ctx, request{method: http.MethodPut, path: pathEscape("workspaces", workspacePublicID, "destinations", destinationPublicID), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a destination
// (DELETE /workspaces/{public_id}/destinations/{destination_public_id}).
func (s *DestinationsService) Delete(ctx context.Context, workspacePublicID, destinationPublicID string) (*SuccessResult, error) {
	var out SuccessResult
	if err := s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("workspaces", workspacePublicID, "destinations", destinationPublicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
