package shortn

import (
	"context"
	"net/http"
)

// AnalyticsService covers /analytics. Its calls count toward the analytics
// rate limit.
type AnalyticsService service

// LinkAnalytics is a link's all-time click totals.
type LinkAnalytics struct {
	TotalClicks  int64 `json:"total_clicks"`
	UniqueClicks int64 `json:"unique_clicks"`
	// ClicksByCountry maps country names (and "Unknown") to clicks.
	ClicksByCountry map[string]int64 `json:"clicks_by_country"`
	Raw             map[string]any   `json:"-"`
}

func (a *LinkAnalytics) UnmarshalJSON(b []byte) error {
	type plain LinkAnalytics
	return decodeObject(b, (*plain)(a), &a.Raw)
}

// Link reads a link's click totals by public_id
// (GET /analytics/links/{public_id}). The endpoint takes no query
// parameters: the totals cover the link's whole history.
func (s *AnalyticsService) Link(ctx context.Context, publicID string) (*LinkAnalytics, error) {
	var out LinkAnalytics
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("analytics", "links", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
