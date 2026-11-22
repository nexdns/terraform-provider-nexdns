package api

import "fmt"

func (c *Client) GetZone(id string) (*Zone, error) {
	var zone Zone
	err := c.do("GET", "/zones/"+id, nil, &zone)
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

func (c *Client) GetZoneByName(name string) (*ZoneListItem, error) {
	var zones []ZoneListItem
	err := c.do("GET", fmt.Sprintf("/zones?search=%s&per_page=100", name), nil, &zones)
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].Name == name {
			return &zones[i], nil
		}
	}
	return nil, &APIError{StatusCode: 404, Code: "not_found", Message: fmt.Sprintf("zone %q not found", name)}
}

func (c *Client) CreateZone(req CreateZoneRequest) (*ZoneListItem, error) {
	var zone ZoneListItem
	err := c.do("POST", "/zones", req, &zone)
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

func (c *Client) DeleteZone(id string) error {
	return c.do("DELETE", "/zones/"+id, nil, nil)
}

func (c *Client) ListNSGroups() ([]NSGroup, error) {
	var groups []NSGroup
	err := c.do("GET", "/ns-groups", nil, &groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// MoveZone moves a zone into another nameserver group, the only change the
// API allows on an existing zone.
func (c *Client) MoveZone(id, nsGroup string) (*Zone, error) {
	var zone Zone
	err := c.do("PATCH", "/zones/"+id, MoveZoneRequest{NSGroup: nsGroup}, &zone)
	if err != nil {
		return nil, err
	}

	return &zone, nil
}
