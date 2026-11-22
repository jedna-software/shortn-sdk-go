package shortn

import (
	"context"
	"net/http"
	"strconv"
)

// DomainsService covers /domains. Create, Verify and Delete need an
// account-wide API key.
type DomainsService service

// DNSRecord is a record to create at the DNS provider.
type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
	TTL   string `json:"ttl"`
}

// DomainDNS lists the records a custom domain needs: the TXT record, plus a
// CNAME for a subdomain or A records for a root domain.
type DomainDNS struct {
	VerificationTXT *DNSRecord  `json:"verification_txt"`
	SubdomainCNAME  *DNSRecord  `json:"subdomain_cname"`
	ApexARecords    []DNSRecord `json:"apex_a_records"`
}

// Domain is a custom domain. Domains are identified by a numeric ID.
type Domain struct {
	ID          int64  `json:"id"`
	Domain      string `json:"domain"`
	IsOwner     bool   `json:"is_owner"`
	OwnerName   string `json:"owner_name"`
	CanManage   bool   `json:"can_manage"`
	AccessLabel string `json:"access_label"`
	// Status is ready, in_progress, not_ready, inactive or deleting.
	Status                   string     `json:"status"`
	DNSStatus                string     `json:"dns_status"`
	TLSStatus                string     `json:"tls_status"`
	ProvisioningStatus       string     `json:"provisioning_status"`
	TLSPhase                 string     `json:"tls_phase"`
	IsActive                 bool       `json:"is_active"`
	IsVerified               bool       `json:"is_verified"`
	SSLEnabled               bool       `json:"ssl_enabled"`
	Ready                    bool       `json:"ready"`
	LinksUsing               int64      `json:"links_using"`
	TeamAssignmentCount      int64      `json:"team_assignment_count"`
	VerificationToken        string     `json:"verification_token"`
	VerificationFailureCount int64      `json:"verification_failure_count"`
	WorkspacePublicIDs       []string   `json:"workspace_public_ids"`
	DNS                      *DomainDNS `json:"dns"`
	CreatedAt                string     `json:"created_at"`
	VerifiedAt               string     `json:"verified_at"`

	Raw map[string]any `json:"-"`
}

func (d *Domain) UnmarshalJSON(b []byte) error {
	type plain Domain
	return decodeObject(b, (*plain)(d), &d.Raw)
}

// DomainList is the answer of Domains.List.
type DomainList struct {
	Data []Domain       `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (l *DomainList) UnmarshalJSON(b []byte) error {
	type plain DomainList
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// CreateDomainParams is the body of Domains.Create.
type CreateDomainParams struct {
	// Domain is a hostname such as go.example.com.
	Domain string `json:"domain"`
}

// DomainResponse is the answer of Domains.Create.
type DomainResponse struct {
	Domain Domain         `json:"domain"`
	Raw    map[string]any `json:"-"`
}

func (r *DomainResponse) UnmarshalJSON(b []byte) error {
	type plain DomainResponse
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// DomainVerification is what a DNS check found. TXT and Routing hold the
// per-record details.
type DomainVerification struct {
	Verified bool           `json:"verified"`
	Error    string         `json:"error"`
	TXT      map[string]any `json:"txt"`
	Routing  map[string]any `json:"routing"`
	Raw      map[string]any `json:"-"`
}

func (v *DomainVerification) UnmarshalJSON(b []byte) error {
	type plain DomainVerification
	return decodeObject(b, (*plain)(v), &v.Raw)
}

// DomainVerifyResult is the answer of Domains.Verify.
type DomainVerifyResult struct {
	Domain       Domain             `json:"domain"`
	Verification DomainVerification `json:"verification"`
	Raw          map[string]any     `json:"-"`
}

func (r *DomainVerifyResult) UnmarshalJSON(b []byte) error {
	type plain DomainVerifyResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// DomainDeleteResult is the answer of Domains.Delete.
type DomainDeleteResult struct {
	Deleted bool           `json:"deleted"`
	ID      int64          `json:"id"`
	Domain  Domain         `json:"domain"`
	Raw     map[string]any `json:"-"`
}

func (r *DomainDeleteResult) UnmarshalJSON(b []byte) error {
	type plain DomainDeleteResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// List lists the domains you own and those assigned to your workspaces
// (GET /domains).
func (s *DomainsService) List(ctx context.Context) (*DomainList, error) {
	var out DomainList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/domains"}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create adds a custom domain (POST /domains). Create its DNS records, then
// call Verify.
func (s *DomainsService) Create(ctx context.Context, params CreateDomainParams) (*DomainResponse, error) {
	var out DomainResponse
	if err := s.client.do(ctx, request{method: http.MethodPost, path: "/domains", body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Verify checks a domain's DNS records now (POST /domains/{id}/verify).
func (s *DomainsService) Verify(ctx context.Context, id int64) (*DomainVerifyResult, error) {
	var out DomainVerifyResult
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("domains", strconv.FormatInt(id, 10), "verify")}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a domain that no link uses (DELETE /domains/{id}).
func (s *DomainsService) Delete(ctx context.Context, id int64) (*DomainDeleteResult, error) {
	var out DomainDeleteResult
	if err := s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("domains", strconv.FormatInt(id, 10))}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
