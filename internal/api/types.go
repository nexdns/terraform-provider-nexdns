package api

import (
	"encoding/json"
	"errors"
)

type APIResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Meta   json.RawMessage `json:"meta,omitempty"`
}

type APIError struct {
	StatusCode int
	Code       string              `json:"code"`
	Message    string              `json:"message"`
	Details    map[string][]string `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

// IsNotFound reports whether err is an API 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}

type Zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	Nameservers []string `json:"nameservers"`
	NSGroup     *struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"ns_group"`
	RecordsCount int    `json:"records_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ZoneListItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	NSGroup string `json:"ns_group"`
}

// Record is a record in API responses. Type-specific values (priority,
// weight, port, flags, tag, keytag and so on) are nested under Fields.
type Record struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Content  string                 `json:"content"`
	TTL      int                    `json:"ttl"`
	Disabled bool                   `json:"disabled"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

type DNSSECStatus struct {
	Enabled   bool        `json:"enabled"`
	Keys      []DNSSECKey `json:"keys"`
	DSRecords []string    `json:"ds_records"`
}

type DNSSECKey struct {
	Type      string `json:"type"`
	Algorithm string `json:"algorithm"`
	Bits      int    `json:"bits"`
	Active    bool   `json:"active"`
	DNSKEY    string `json:"dnskey"`
}

type NSGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CreateZoneRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	NSGroup  string `json:"ns_group,omitempty"`
	MasterIP string `json:"master_ip,omitempty"`
}

type CreateRecordRequest struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	TTL          int    `json:"ttl,omitempty"`
	Priority     *int   `json:"priority,omitempty"`
	Weight       *int   `json:"weight,omitempty"`
	Port         *int   `json:"port,omitempty"`
	Flags        *int   `json:"flags,omitempty"`
	Tag          string `json:"tag,omitempty"`
	KeyTag       *int   `json:"keytag,omitempty"`
	Algorithm    *int   `json:"algorithm,omitempty"`
	DigestType   *int   `json:"digest_type,omitempty"`
	Usage        *int   `json:"usage,omitempty"`
	Selector     *int   `json:"selector,omitempty"`
	MatchingType *int   `json:"matching_type,omitempty"`
}

type UpdateRecordRequest struct {
	Name         *string `json:"name,omitempty"`
	Content      *string `json:"content,omitempty"`
	TTL          *int    `json:"ttl,omitempty"`
	Priority     *int    `json:"priority,omitempty"`
	Weight       *int    `json:"weight,omitempty"`
	Port         *int    `json:"port,omitempty"`
	Flags        *int    `json:"flags,omitempty"`
	Tag          *string `json:"tag,omitempty"`
	KeyTag       *int    `json:"keytag,omitempty"`
	Algorithm    *int    `json:"algorithm,omitempty"`
	DigestType   *int    `json:"digest_type,omitempty"`
	Usage        *int    `json:"usage,omitempty"`
	Selector     *int    `json:"selector,omitempty"`
	MatchingType *int    `json:"matching_type,omitempty"`
}

// MoveZoneRequest is the body of PATCH /zones/{id}.
type MoveZoneRequest struct {
	NSGroup string `json:"ns_group"`
}
